//go:build js && wasm

// Copyright 2017 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package keys provides APIs to manage configured keys and load them into an
// SSH agent.
package keys

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"syscall/js"

	"github.com/ScaleFT/sshkeys"
	"github.com/google/chrome-ssh-agent/go/dom"
	"github.com/norunners/vert"
	"github.com/youmark/pkcs8"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// ID is a unique identifier for a configured key.
type ID string

const (
	// InvalidID is a special ID that will not be assigned to any key.
	InvalidID ID = ""
)

// ConfiguredKey is a key configured for use.
type ConfiguredKey struct {
	// Id is the unique ID for this key.
	ID string `js:"id"`
	// Name is a name allocated to key.
	Name string `js:"name"`
	// Encrypted indicates if the key is encrypted and requires a passphrase
	// to load.
	Encrypted bool `js:"encrypted"`
}

// LoadedKey is a key loaded into the agent.
type LoadedKey struct {
	// Type is the type of key loaded in the agent (e.g., 'ssh-rsa').
	Type string `js:"type"`
	// InternalBlob is the public key material for the loaded key. Must
	// be exported to be handled correctly in conversion to/from js.Value.
	InternalBlob string `js:"blob"`
	// Comment is a comment for the loaded key.
	Comment string `js:"comment"`
}

// SetBlob sets the given public key material for the loaded key.
func (k *LoadedKey) SetBlob(b []byte) {
	// Store as base64-encoded string. Two simpler solutions did not appear
	// to work:
	// - Storing as a []byte resulted in data not being passed via Chrome's
	//   messaging.
	// - Casting to a string resulted in different data being read from the
	//   field.
	k.InternalBlob = base64.StdEncoding.EncodeToString(b)
}

// Blob returns the public key material for the loaded key.
func (k *LoadedKey) Blob() []byte {
	b, err := base64.StdEncoding.DecodeString(k.InternalBlob)
	if err != nil {
		dom.LogError("failed to decode key blob: %v", err)
		return nil
	}

	return b
}

// ID returns the unique ID corresponding to the key.  If the ID cannot be
// determined, then InvalidID is returned.
//
// The ID for a key loaded into the agent is stored in the Comment field as
// a string in a particular format.
func (k *LoadedKey) ID() ID {
	if !strings.HasPrefix(k.Comment, commentPrefix) {
		return InvalidID
	}

	return ID(strings.TrimPrefix(k.Comment, commentPrefix))
}

// Manager provides an API for managing configured keys and loading them into
// an SSH agent.
type Manager interface {
	// Configured returns the full set of keys that are configured. The
	// callback is invoked with the result.
	Configured(callback func(keys []*ConfiguredKey, err error))

	// Add configures a new key.  name is a human-readable name describing
	// the key, and pemPrivateKey is the PEM-encoded private key.  callback
	// is invoked when complete.
	Add(name string, pemPrivateKey string, callback func(err error))

	// Remove removes the key with the specified ID.  callback is invoked
	// when complete.
	//
	// Note that it might be nice to return an error here, but
	// the underlying Chrome APIs don't make it trivial to determine
	// if the requested key was removed, or ignored because it didn't
	// exist.  This could be improved, but it doesn't seem worth it at
	// the moment.
	Remove(id ID, callback func(err error))

	// Loaded returns the full set of keys loaded into the agent. The
	// callback is invoked with the result.
	Loaded(callback func(keys []*LoadedKey, err error))

	// Load loads a new key into to the agent, using the passphrase to
	// decrypt the private key.  callback is invoked when complete.
	//
	// NOTE: Unencrypted private keys are not currently supported.
	Load(id ID, passphrase string, callback func(err error))

	// Unload unloads a key from the agent. callback is invoked when
	// complete.
	Unload(key *LoadedKey, callback func(err error))
}

// PersistentStore provides access to underlying storage.  See chrome.Storage
// for details on the methods; using this interface allows for alternate
// implementations during testing.
type PersistentStore interface {
	// Set stores new data. See chrome.Storage.Set() for details.
	Set(data map[string]js.Value, callback func(err error))

	// Get gets data from storage. See chrome.Storage.Get() for details.
	Get(callback func(data map[string]js.Value, err error))

	// Delete deletes data from storage. See chrome.Storage.Delete() for
	// details.
	Delete(keys []string, callback func(err error))
}

// NewManager returns a Manager implementation that can manage keys in the
// supplied agent, and store configured keys in the supplied storage.
func NewManager(agt agent.Agent, storage PersistentStore) Manager {
	return &manager{
		agent:   agt,
		storage: storage,
	}
}

// manager is an implementation of Manager.
type manager struct {
	agent   agent.Agent
	storage PersistentStore
}

// storedKey is the raw object stored in persistent storage for a configured
// key.
type storedKey struct {
	ID            string `js:"id"`
	Name          string `js:"name"`
	PEMPrivateKey string `js:"pemPrivateKey"`
}

// OpenSSH determines if the private key is an OpenSSH-formatted key.
func (s *storedKey) OpenSSH() bool {
	block, _ := pem.Decode([]byte(s.PEMPrivateKey))
	if block == nil {
		// Attempt to handle this gracefully and guess that it isn't
		// OpenSSH formatted. If the key is not properly formatted,
		// we'll complain when it is loaded.
		return false
	}

	return block.Type == "OPENSSH PRIVATE KEY"
}

// PKCS8 determines if the private key is a PKCS#8 formatted key.
func (s *storedKey) PKCS8() bool {
	block, _ := pem.Decode([]byte(s.PEMPrivateKey))
	if block == nil {
		// Attempt to handle this gracefully and guess that it isn't
		// PKCS#8 formatted. If the key is not properly formatted,
		// we'll complain when it is loaded.
		return false
	}

	// Types used for PKCS#8 keys:
	// https://github.com/kjur/jsrsasign/wiki/Tutorial-for-PKCS5-and-PKCS8-PEM-private-key-formats-differences
	return block.Type == "ENCRYPTED PRIVATE KEY" || block.Type == "PRIVATE KEY"
}

// Encrypted determines if the private key is encrypted. The Proc-Type header
// contains 'ENCRYPTED' if the key is encrypted. See RFC 1421 Section 4.6.1.1.
func (s *storedKey) Encrypted() bool {
	block, _ := pem.Decode([]byte(s.PEMPrivateKey))
	if block == nil {
		// Attempt to handle this gracefully and guess that it isn't
		// encrypted.  If the key is not properly formatted, we'll
		// complain anyways when it is loaded.
		return false
	}

	// Type used for PKCS#8 keys.
	// https://github.com/kjur/jsrsasign/wiki/Tutorial-for-PKCS5-and-PKCS8-PEM-private-key-formats-differences
	if block.Type == "ENCRYPTED PRIVATE KEY" {
		return true
	}

	// OpenSSH keys don't have a type or header indicating if they are
	// encrypted. We could parse the key to determine that, but that would
	// reimplement the underlying crypto libraries. Instead, just attempt to
	// decrypt it without a passphrase.
	if block.Type == "OPENSSH PRIVATE KEY" {
		_, err := ssh.ParseRawPrivateKey([]byte(s.PEMPrivateKey))
		return err != nil
	}

	return strings.Contains(block.Headers["Proc-Type"], "ENCRYPTED")
}

const (
	// keyPrefix is the prefix for keys stored in persistent storage.
	// The full key is of the form 'key.<id>'.
	keyPrefix = "key."
	// commentPrefix is the prefix for the comment included when a
	// configured key is loaded into the agent. The full comment is of the
	// form 'chrome-ssh-agent:<id>'.
	commentPrefix = "chrome-ssh-agent:"
)

// readKeys returns all the stored keys from persistent storage. callback is
// invoked with the returned keys.
func (m *manager) readKeys(callback func(keys []*storedKey, err error)) {
	m.storage.Get(func(data map[string]js.Value, err error) {
		if err != nil {
			callback(nil, fmt.Errorf("failed to read from storage: %w", err))
			return
		}

		var keys []*storedKey
		for k, v := range data {
			if !strings.HasPrefix(k, keyPrefix) {
				continue
			}

			var sk storedKey
			if err := vert.ValueOf(v).AssignTo(&sk); err != nil {
				dom.LogError(fmt.Sprintf("failed to parse key %s; dropping", k))
				continue
			}

			keys = append(keys, &sk)
		}
		callback(keys, nil)
	})
}

// readKey returns the key of the specified ID from persistent storage. callback
// is invoked with the returned key.
func (m *manager) readKey(id ID, callback func(key *storedKey, err error)) {
	m.readKeys(func(keys []*storedKey, err error) {
		if err != nil {
			callback(nil, fmt.Errorf("failed to read keys: %w", err))
			return
		}

		for _, k := range keys {
			if ID(k.ID) == id {
				callback(k, nil)
				return
			}
		}

		callback(nil, nil)
	})
}

// writeKey writes a new key to persistent storage.  callback is invoked when
// complete.
func (m *manager) writeKey(name string, pemPrivateKey string, callback func(err error)) {
	i, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		callback(fmt.Errorf("failed to generate new ID: %w", err))
		return
	}
	id := ID(i.String())
	storageKey := fmt.Sprintf("%s%s", keyPrefix, id)
	sk := storedKey{
		ID:            string(id),
		Name:          name,
		PEMPrivateKey: pemPrivateKey,
	}
	data := map[string]js.Value{
		storageKey: vert.ValueOf(sk).JSValue(),
	}
	m.storage.Set(data, func(err error) {
		callback(err)
	})
}

// removeKey removes the key with the specified ID from persistent storage.
// callback is invoked on completion.
func (m *manager) removeKey(id ID, callback func(err error)) {
	m.readKeys(func(keys []*storedKey, err error) {
		if err != nil {
			callback(fmt.Errorf("failed to enumerate keys: %w", err))
			return
		}

		var storageKeys []string
		for _, k := range keys {
			if ID(k.ID) == id {
				storageKeys = append(storageKeys, fmt.Sprintf("%s%s", keyPrefix, k.ID))
			}
		}

		m.storage.Delete(storageKeys, func(err error) {
			if err != nil {
				callback(fmt.Errorf("failed to delete keys: %w", err))
				return
			}
			callback(nil)
		})
	})
}

// Configured implements Manager.Configured.
func (m *manager) Configured(callback func(keys []*ConfiguredKey, err error)) {
	m.readKeys(func(keys []*storedKey, err error) {
		if err != nil {
			callback(nil, fmt.Errorf("failed to read keys: %w", err))
			return
		}

		var result []*ConfiguredKey
		for _, k := range keys {
			c := ConfiguredKey{
				ID:        k.ID,
				Name:      k.Name,
				Encrypted: k.Encrypted(),
			}
			result = append(result, &c)
		}
		callback(result, nil)
	})
}

var (
	errInvalidName = errors.New("invalid name")
)

// Add implements Manager.Add.
func (m *manager) Add(name string, pemPrivateKey string, callback func(err error)) {
	if name == "" {
		callback(fmt.Errorf("%w: name must not be empty", errInvalidName))
		return
	}

	m.writeKey(name, pemPrivateKey, func(err error) {
		callback(err)
	})
}

// Remove implements Manager.Remove.
func (m *manager) Remove(id ID, callback func(err error)) {
	m.removeKey(id, func(err error) {
		callback(err)
	})
}

// Loaded implements Manager.Loaded.
func (m *manager) Loaded(callback func(keys []*LoadedKey, err error)) {
	loaded, err := m.agent.List()
	if err != nil {
		callback(nil, fmt.Errorf("failed to list loaded keys: %w", err))
		return
	}

	var result []*LoadedKey
	for _, l := range loaded {
		k := LoadedKey{
			Type:    l.Type(),
			Comment: l.Comment,
		}
		k.SetBlob(l.Marshal())
		result = append(result, &k)
	}

	callback(result, nil)
}

var (
	errKeyNotFound  = errors.New("key not found")
	errDecodeFailed = errors.New("key decode failed")
	errParseFailed  = errors.New("key parse failed")
)

// Load implements Manager.Load.
func (m *manager) Load(id ID, passphrase string, callback func(err error)) {
	m.readKey(id, func(key *storedKey, err error) {
		if err != nil {
			callback(fmt.Errorf("failed to read key: %w", err))
			return
		}

		if key == nil {
			callback(fmt.Errorf("%w: failed to find key with ID %s", errKeyNotFound, id))
			return
		}

		var priv interface{}
		if key.OpenSSH() && passphrase != "" {
			// Crypto libraries don't yet support encrypted OpenSSH keys:
			//   https://github.com/golang/go/issues/18692
			priv, err = sshkeys.ParseEncryptedRawPrivateKey([]byte(key.PEMPrivateKey), []byte(passphrase))
		} else if key.PKCS8() {
			// Crypto libraries don't yet support encrypted PKCS#8 keys:
			//   https://github.com/golang/go/issues/8860
			var block *pem.Block
			block, _ = pem.Decode([]byte(key.PEMPrivateKey))
			if block == nil {
				callback(fmt.Errorf("%w: failed to decode encrypted private key", errDecodeFailed))
				return
			}
			if passphrase != "" {
				priv, err = pkcs8.ParsePKCS8PrivateKey(block.Bytes, []byte(passphrase))
			} else {
				priv, err = pkcs8.ParsePKCS8PrivateKey(block.Bytes, nil)
			}
		} else if key.Encrypted() {
			priv, err = ssh.ParseRawPrivateKeyWithPassphrase([]byte(key.PEMPrivateKey), []byte(passphrase))
		} else {
			priv, err = ssh.ParseRawPrivateKey([]byte(key.PEMPrivateKey))
		}
		// Forward incorrect password errors on directly.
		if err != nil && errors.Is(err, x509.IncorrectPasswordError) {
			callback(fmt.Errorf("failed to parse private key: %w", err))
			return
		}
		// Wrap all other non-specific errors.
		if err != nil {
			callback(fmt.Errorf("%w: %v", errParseFailed, err))
			return
		}

		err = m.agent.Add(agent.AddedKey{
			PrivateKey: priv,
			Comment:    fmt.Sprintf("%s%s", commentPrefix, id),
		})
		if err != nil {
			callback(fmt.Errorf("failed to add key to agent: %w", err))
			return
		}
		callback(nil)
	})
}

var (
	errUnloadFailed = errors.New("key unload failed")
)

// Unload implements Manager.Unload.
func (m *manager) Unload(key *LoadedKey, callback func(err error)) {
	pub := &agent.Key{
		Format: key.Type,
		Blob:   key.Blob(),
	}
	if err := m.agent.Remove(pub); err != nil {
		callback(fmt.Errorf("%w: %v", errUnloadFailed, err))
		return
	}
	callback(nil)
}
