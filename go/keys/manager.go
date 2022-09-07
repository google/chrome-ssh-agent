//go:build js

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
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/google/chrome-ssh-agent/go/jsutil"
	"github.com/google/chrome-ssh-agent/go/storage"
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
		jsutil.LogError("failed to decode key blob: %v", err)
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
	// Configured returns the full set of keys that are configured.
	Configured(ctx jsutil.AsyncContext) ([]*ConfiguredKey, error)

	// Add configures a new key.  name is a human-readable name describing
	// the key, and pemPrivateKey is the PEM-encoded private key.
	Add(ctx jsutil.AsyncContext, name string, pemPrivateKey string) error

	// Remove removes the key with the specified ID.
	//
	// Note that it might be nice to return an error here, but
	// the underlying Chrome APIs don't make it trivial to determine
	// if the requested key was removed, or ignored because it didn't
	// exist.  This could be improved, but it doesn't seem worth it at
	// the moment.
	Remove(ctx jsutil.AsyncContext, id ID) error

	// Loaded returns the full set of keys loaded into the agent.
	Loaded(ctx jsutil.AsyncContext) ([]*LoadedKey, error)

	// Load loads a new key into to the agent, using the passphrase to
	// decrypt the private key.
	//
	// NOTE: Unencrypted private keys are not currently supported.
	Load(ctx jsutil.AsyncContext, id ID, passphrase string) error

	// Unload unloads a key from the agent.
	Unload(ctx jsutil.AsyncContext, id ID) error
}

// NewManager returns a Manager implementation that can manage keys in the
// supplied agent, and store configured keys in the supplied storage.
func NewManager(agt agent.Agent, syncStorage, sessionStorage storage.Area) *DefaultManager {
	return &DefaultManager{
		agent:          agt,
		syncStorage:    syncStorage,
		sessionStorage: sessionStorage,
		storedKeys:     storage.NewTyped[storedKey](syncStorage, storedKeyPrefixes),
		sessionKeys:    storage.NewTyped[sessionKey](sessionStorage, sessionKeyPrefixes),
	}
}

// DefaultManager is an implementation of Manager.
type DefaultManager struct {
	agent          agent.Agent
	syncStorage    storage.Area
	sessionStorage storage.Area
	storedKeys     *storage.Typed[storedKey]
	sessionKeys    *storage.Typed[sessionKey]
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
//
// Just as the key is stored in-memory in an SSH agent, we store it decrypted
// here.  We may be suspended/unloaded at arbitrary points by the browser, and
// we need to resume without re-prompting the user for their passphrase each
// time.
type sessionKey struct {
	ID         string `js:"id"`
	PrivateKey string `js:"privateKey"`
}

var (
	// storedKeyPrefix is the prefix for keys stored in persistent storage.
	storedKeyPrefixes = []string{"key"}
	// sessionKeyPrefix is the prefix for key material stored in-memory
	// for our current session.
	sessionKeyPrefixes = []string{"key"}

	// oldStoredKeyPrefixes are the prefixes for stored keys that we
	// previously used which are safe to delete from storage.
	//
	// WARNING: Only add a prefix to this list *after* the following
	// sequence of events:
	// (a) A replacement prefix has been added to the storedKeyPrefixes.
	// (b) Release including (a) has been deployed for 3 months.
	//     NOTE: At this point, we should be writing data to the new prefix
	//     and should be assured that data for both prefixes is equivalent.
	// (c) The old prefix has been removed from the above list.
	// (d) Release including (b) has been deployed for at least 3 weeks
	//     without any reported issues of data loss.
	//     NOTE: At this point, it is safe to add the prefix back to
	//     storedKeyPrefixes; the data is present, but we just aren't
	//     reading it.
	//
	// This sequence of events is important to support rollbacks without
	// incorrectly deleting data.
	//
	// For tracking, comments should track the progression of these events
	// for each prefix slated for deletion.
	oldStoredKeyPrefixes = []string{}

	// oldSessionKeyPrefixes are the prefixes for session key material
	// that are safe to delete from storage.
	//
	// WARNING: See warning for oldStoredKeyPrefixes above; the same applies
	// here.
	oldSessionKeyPrefixes = []string{}
)

const (
	// commentPrefix is the prefix for the comment included when a
	// configured key is loaded into the agent. The full comment is of the
	// form 'chrome-ssh-agent:<id>'.
	commentPrefix = "chrome-ssh-agent:"
)

// Configured implements Manager.Configured.
func (m *DefaultManager) Configured(ctx jsutil.AsyncContext) ([]*ConfiguredKey, error) {
	keys, err := m.storedKeys.ReadAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read keys: %w", err)
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
	return result, nil
}

var (
	errInvalidName = errors.New("invalid name")
)

// Add implements Manager.Add.
func (m *DefaultManager) Add(ctx jsutil.AsyncContext, name string, pemPrivateKey string) error {
	if name == "" {
		return fmt.Errorf("%w: name must not be empty", errInvalidName)
	}

	i, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return fmt.Errorf("failed to generate new ID: %w", err)
	}

	sk := &storedKey{
		ID:            i.String(),
		Name:          name,
		PEMPrivateKey: pemPrivateKey,
	}
	return m.storedKeys.Write(ctx, sk)
}

// Remove implements Manager.Remove.
func (m *DefaultManager) Remove(ctx jsutil.AsyncContext, id ID) error {
	return m.storedKeys.Delete(ctx, func(sk *storedKey) bool { return ID(sk.ID) == id })
}

// Loaded implements Manager.Loaded.
func (m *DefaultManager) Loaded(ctx jsutil.AsyncContext) ([]*LoadedKey, error) {
	loaded, err := m.agent.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list loaded keys: %w", err)
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

	return result, nil
}

var (
	errKeyNotFound   = errors.New("key not found")
	errDecodeFailed  = errors.New("key decode failed")
	errParseFailed   = errors.New("key parse failed")
	errMarshalFailed = errors.New("key marshalling failed")
)

// CleanupOldData removes storage data that is no longer required.
func (m *DefaultManager) CleanupOldData(ctx jsutil.AsyncContext) {
	jsutil.LogDebug("DefaultManager.CleanupOldData: Cleaning up stored keys")

	areas := []storage.Area{
		m.syncStorage,
		m.sessionStorage,
	}
	prefixesLists := [][]string{
		oldStoredKeyPrefixes,
		oldSessionKeyPrefixes,
	}
	for _, area := range areas {
		for _, prefixes := range prefixesLists {
			if err := storage.DeleteViewPrefixes(ctx, prefixes, area); err != nil {
				jsutil.LogError("failed to delete old prefixes '%s': %v", oldStoredKeyPrefixes, err)
			}
		}
	}
}

// LoadFromSession loads all keys for the current session into the agent.
func (m *DefaultManager) LoadFromSession(ctx jsutil.AsyncContext) error {
	// Read session keys. We'll load these into the agent.
	jsutil.LogDebug("DefaultManager.LoadFromSession: Read session keys")
	sessionKeys, err := m.sessionKeys.ReadAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to read session keys: %w", err)
	}

	// Attempt to load each into the agent.
	jsutil.LogDebug("DefaultManager.LoadFromSession: Load session keys")
	for _, k := range sessionKeys {
		if err := m.addToAgent(ID(k.ID), decryptedKey(k.PrivateKey)); err != nil {
			jsutil.LogError("failed to load session key ID %s into agent: %v; skipping", k.ID, err)
		}
	}
	return nil
}

type decryptedKey string

const (
	pkcs8BlockType = "PRIVATE KEY"
)

func decryptKey(key *storedKey, passphrase string) (decryptedKey, error) {
	// Decode and decrypt the key.
	var err error
	var priv interface{}
	if key.EncryptedPKCS8() {
		// Crypto libraries don't yet support encrypted PKCS#8 keys:
		//   https://github.com/golang/go/issues/8860
		var block *pem.Block
		block, _ = pem.Decode([]byte(key.PEMPrivateKey))
		if block == nil {
			return "", fmt.Errorf("%w: failed to decode encrypted private key", errDecodeFailed)
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
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}
	// Wrap all other non-specific errors.
	if err != nil {
		return "", fmt.Errorf("%w: %v", errParseFailed, err)
	}

	// Workaround for https://github.com/google/chrome-ssh-agent/issues/28.
	// In the case of ed25519 keys, ssh.ParseRawPublicKey() will return a
	// *ed25519.PrivateKey (pointer), but x509.MarshalPKCS8PrivateKey()
	// expects a ed25519.PrivateKey (non-pointer).
	if k, ok := priv.(*ed25519.PrivateKey); ok {
		priv = *k
	}

	// Marshal to PKCS#8 format.
	buf, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", fmt.Errorf("%w: %v", errMarshalFailed, err)
	}

	return decryptedKey(pem.EncodeToMemory(&pem.Block{
		Type:  pkcs8BlockType,
		Bytes: buf,
	})), nil
}

func parseDecryptedKey(pemPrivateKey decryptedKey) (interface{}, error) {
	return ssh.ParseRawPrivateKey([]byte(pemPrivateKey))
}

func (m *DefaultManager) addToAgent(id ID, key decryptedKey) error {
	priv, err := parseDecryptedKey(key)
	if err != nil {
		return err
	}

	err = m.agent.Add(agent.AddedKey{
		PrivateKey: priv,
		Comment:    fmt.Sprintf("%s%s", commentPrefix, id),
	})
	if err != nil {
		return fmt.Errorf("failed to add key to agent: %w", err)
	}
	return nil
}

// Load implements Manager.Load.
func (m *DefaultManager) Load(ctx jsutil.AsyncContext, id ID, passphrase string) error {
	key, err := m.storedKeys.Read(ctx, func(key *storedKey) bool { return ID(key.ID) == id })
	if err != nil {
		return fmt.Errorf("failed to read key: %w", err)
	}

	if key == nil {
		return fmt.Errorf("%w: failed to find key with ID %s", errKeyNotFound, id)
	}

	decrypted, err := decryptKey(key, passphrase)
	if err != nil {
		return fmt.Errorf("failed to decrypt key: %w", err)
	}

	if err := m.addToAgent(id, decrypted); err != nil {
		return err
	}

	sk := &sessionKey{
		ID:         string(id),
		PrivateKey: string(decrypted),
	}
	if err := m.sessionKeys.Write(ctx, sk); err != nil {
		return fmt.Errorf("failed to store loaded key to session: %w", err)
	}
	return nil
}

var (
	errAgentUnloadFailed   = errors.New("key unload from agent failed")
	errStorageUnloadFailed = errors.New("key removal from session storage failed")
)

// Unload implements Manager.Unload.
func (m *DefaultManager) Unload(ctx jsutil.AsyncContext, id ID) error {
	if id == InvalidID {
		return fmt.Errorf("%w: invalid id", errAgentUnloadFailed)
	}

	loaded, err := m.Loaded(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to enumerate loaded keys: %v", errAgentUnloadFailed, id)
	}

	var lk *LoadedKey
	for _, l := range loaded {
		if l.ID() == id {
			lk = l
			break
		}
	}
	if lk == nil {
		return fmt.Errorf("%w: invalid id: %s", errAgentUnloadFailed, id)
	}

	pub := &agent.Key{
		Format: lk.Type,
		Blob:   lk.Blob(),
	}
	if err := m.agent.Remove(pub); err != nil {
		return fmt.Errorf("%w: %v", errAgentUnloadFailed, err)
	}

	if err := m.sessionKeys.Delete(ctx, func(sk *sessionKey) bool { return ID(sk.ID) == id }); err != nil {
		return fmt.Errorf("%w: %v", errStorageUnloadFailed, err)
	}

	return nil
}
