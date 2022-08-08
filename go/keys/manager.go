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

	"github.com/google/chrome-ssh-agent/go/chrome"
	"github.com/google/chrome-ssh-agent/go/dom"
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

// NewManager returns a Manager implementation that can manage keys in the
// supplied agent, and store configured keys in the supplied storage.
func NewManager(agt agent.Agent, syncStorage, sessionStorage chrome.PersistentStore) *DefaultManager {
	return &DefaultManager{
		agent:       agt,
		storedKeys:  chrome.NewTypedStore[storedKey](syncStorage, keyPrefix),
		sessionKeys: chrome.NewTypedStore[sessionKey](sessionStorage, keyPrefix),
	}
}

// DefaultManager is an implementation of Manager.
type DefaultManager struct {
	agent       agent.Agent
	storedKeys  *chrome.TypedStore[storedKey]
	sessionKeys *chrome.TypedStore[sessionKey]
}

// storedKey is the raw object stored in persistent storage for a configured
// key.
type storedKey struct {
	ID            string `js:"id"`
	Name          string `js:"name"`
	PEMPrivateKey string `js:"pemPrivateKey"`
}

// EncryptedPKCS8 determines if the private key is an encrypted PKCS#8 formatted
// key.
func (s *storedKey) EncryptedPKCS8() bool {
	block, _ := pem.Decode([]byte(s.PEMPrivateKey))
	if block == nil {
		// Attempt to handle this gracefully and guess that it isn't
		// PKCS#8 formatted. If the key is not properly formatted,
		// we'll complain when it is loaded.
		return false
	}

	// Types used for PKCS#8 keys:
	// https://github.com/kjur/jsrsasign/wiki/Tutorial-for-PKCS5-and-PKCS8-PEM-private-key-formats-differences
	return block.Type == "ENCRYPTED PRIVATE KEY"
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

// sessionKey is the raw object stored in session storage for a key that has
// been loaded into the agent.
type sessionKey struct {
	ID         string `js:"id"`
	Passphrase string `js:"passphrase"`
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

// Configured implements Manager.Configured.
func (m *DefaultManager) Configured(callback func(keys []*ConfiguredKey, err error)) {
	m.storedKeys.ReadAll(func(keys []*storedKey, err error) {
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
func (m *DefaultManager) Add(name string, pemPrivateKey string, callback func(err error)) {
	if name == "" {
		callback(fmt.Errorf("%w: name must not be empty", errInvalidName))
		return
	}

	i, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		callback(fmt.Errorf("failed to generate new ID: %w", err))
		return
	}

	sk := &storedKey{
		ID:            i.String(),
		Name:          name,
		PEMPrivateKey: pemPrivateKey,
	}
	m.storedKeys.Write(sk, callback)
}

// Remove implements Manager.Remove.
func (m *DefaultManager) Remove(id ID, callback func(err error)) {
	m.storedKeys.Delete(func(sk *storedKey) bool { return ID(sk.ID) == id }, callback)
}

// Loaded implements Manager.Loaded.
func (m *DefaultManager) Loaded(callback func(keys []*LoadedKey, err error)) {
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

// LoadFromSession loads all keys for the current session into the agent.
func (m *DefaultManager) LoadFromSession(callback func(err error)) {
	// Read all the stored keys. We need these to load session keys
	// into the agent.
	m.storedKeys.ReadAll(func(storedKeys []*storedKey, err error) {
		if err != nil {
			callback(fmt.Errorf("failed to read stored keys: %w", err))
			return
		}

		// Index stored keys by ID for faster lookup.
		storedKeysByID := map[string]*storedKey{}
		for _, k := range storedKeys {
			storedKeysByID[k.ID] = k
		}

		// Read session keys. We'll load these into the agent.
		m.sessionKeys.ReadAll(func(sessionKeys []*sessionKey, err error) {
			if err != nil {
				callback(fmt.Errorf("failed to read session keys: %w", err))
				return
			}

			// Attempt to load each into the agent.
			for _, k := range sessionKeys {
				sk, present := storedKeysByID[k.ID]
				if !present {
					dom.LogError("failed to locate session key ID %s in persistent storage; skipping", k.ID)
					continue
				}

				m.addToAgent(ID(k.ID), k.Passphrase, sk, func(err error) {
					if err != nil {
						dom.LogError("failed to load session key ID %s into agent: %v; skipping", k.ID, err)
						return
					}
				})
			}
			callback(nil)
		})
	})
}

func (m *DefaultManager) addToAgent(id ID, passphrase string, key *storedKey, callback func(err error)) {
	var err error
	var priv interface{}
	if key.EncryptedPKCS8() {
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
}

// Load implements Manager.Load.
func (m *DefaultManager) Load(id ID, passphrase string, callback func(err error)) {
	m.storedKeys.Read(
		func(key *storedKey) bool { return ID(key.ID) == id },
		func(key *storedKey, err error) {
			if err != nil {
				callback(fmt.Errorf("failed to read key: %w", err))
				return
			}

			if key == nil {
				callback(fmt.Errorf("%w: failed to find key with ID %s", errKeyNotFound, id))
				return
			}

			m.addToAgent(id, passphrase, key, func(err error) {
				if err != nil {
					callback(err)
					return
				}

				sk := &sessionKey{
					ID:         string(id),
					Passphrase: passphrase,
				}
				m.sessionKeys.Write(sk, func(err error) {
					if err != nil {
						callback(fmt.Errorf("failed to store loaded key to session: %w", err))
						return
					}
					callback(nil)
				})
			})
		})
}

var (
	errAgentUnloadFailed   = errors.New("key unload from agent failed")
	errStorageUnloadFailed = errors.New("key removal from session storage failed")
)

// Unload implements Manager.Unload.
func (m *DefaultManager) Unload(key *LoadedKey, callback func(err error)) {
	pub := &agent.Key{
		Format: key.Type,
		Blob:   key.Blob(),
	}
	if err := m.agent.Remove(pub); err != nil {
		callback(fmt.Errorf("%w: %v", errAgentUnloadFailed, err))
		return
	}

	id := key.ID()
	if id == InvalidID {
		callback(fmt.Errorf("%w: invalid id", errStorageUnloadFailed))
		return
	}

	m.sessionKeys.Delete(
		func(sk *sessionKey) bool { return ID(sk.ID) == id },
		func(err error) {
			if err != nil {
				callback(fmt.Errorf("%w: %v", errStorageUnloadFailed, err))
				return
			}

			callback(nil)
		})
}
