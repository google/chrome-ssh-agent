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

package keys

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/gopherjs/gopherjs/js"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type ID string

const (
	InvalidID ID = ""
)

type ConfiguredKey struct {
	*js.Object
	Id   ID     `js:"id"`
	Name string `js:"name"`
}

type LoadedKey struct {
	*js.Object
	Type    string `js:"type"`
	Blob    string `js:"blob"`
	Comment string `js:"comment"`
}

func (k *LoadedKey) ID() ID {
	if !strings.HasPrefix(k.Comment, commentPrefix) {
		return InvalidID
	}

	return ID(strings.TrimPrefix(k.Comment, commentPrefix))
}

type Manager interface {
	Configured(callback func(keys []*ConfiguredKey, err error))
	Add(name string, pemPrivateKey string, callback func(err error))
	Remove(id ID, callback func(err error))
	Loaded(callback func(keys []*LoadedKey, err error))
	Load(id ID, passphrase string, callback func(err error))
}

type PersistentStore interface {
	Set(data map[string]interface{}, callback func(err error))
	Get(callback func(data map[string]interface{}, err error))
	Delete(keys []string, callback func(err error))
}

func NewManager(agt agent.Agent, storage PersistentStore) Manager {
	return &manager{
		agent:   agt,
		storage: storage,
	}
}

type manager struct {
	agent   agent.Agent
	storage PersistentStore
}

type storedKey struct {
	*js.Object
	Id            ID     `js:"id"`
	Name          string `js:"name"`
	PEMPrivateKey string `js:"pemPrivateKey"`
}

const (
	keyPrefix     = "key."
	commentPrefix = "chrome-ssh-agent:"
)

func newStoredKey(m map[string]interface{}) *storedKey {
	o := js.Global.Get("Object").New()
	for k, v := range m {
		o.Set(k, v)
	}
	return &storedKey{Object: o}
}

func (m *manager) readKeys(callback func(keys []*storedKey, err error)) {
	m.storage.Get(func(data map[string]interface{}, err error) {
		if err != nil {
			callback(nil, fmt.Errorf("failed to read from storage: %v", err))
			return
		}

		var keys []*storedKey
		for k, v := range data {
			if !strings.HasPrefix(k, keyPrefix) {
				continue
			}

			keys = append(keys, newStoredKey(v.(map[string]interface{})))
		}
		callback(keys, nil)
	})
}

func (m *manager) readKey(id ID, callback func(key *storedKey, err error)) {
	m.readKeys(func(keys []*storedKey, err error) {
		if err != nil {
			callback(nil, fmt.Errorf("failed to read keys: %v", err))
			return
		}

		for _, k := range keys {
			if k.Id == id {
				callback(k, nil)
				return
			}
		}

		callback(nil, nil)
	})
}

func (m *manager) writeKey(name string, pemPrivateKey string, callback func(err error)) {
	i, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		callback(fmt.Errorf("failed to generate new ID: %v", err))
		return
	}
	id := ID(i.String())
	storageKey := fmt.Sprintf("%s%s", keyPrefix, id)
	sk := &storedKey{Object: js.Global.Get("Object").New()}
	sk.Id = id
	sk.Name = name
	sk.PEMPrivateKey = pemPrivateKey
	data := map[string]interface{}{
		storageKey: sk,
	}
	m.storage.Set(data, func(err error) {
		callback(err)
	})
}

func (m *manager) removeKey(id ID, callback func(err error)) {
	m.readKeys(func(keys []*storedKey, err error) {
		if err != nil {
			callback(fmt.Errorf("failed to enumerate keys: %v", err))
			return
		}

		var storageKeys []string
		for _, k := range keys {
			if k.Id == id {
				storageKeys = append(storageKeys, fmt.Sprintf("%s%s", keyPrefix, k.Id))
			}
		}

		m.storage.Delete(storageKeys, func(err error) {
			if err != nil {
				callback(fmt.Errorf("failed to delete keys: %v", err))
				return
			}
			callback(nil)
		})
	})
}

func (m *manager) Configured(callback func(keys []*ConfiguredKey, err error)) {
	m.readKeys(func(keys []*storedKey, err error) {
		if err != nil {
			callback(nil, fmt.Errorf("failed to read keys: %v", err))
			return
		}

		var result []*ConfiguredKey
		for _, k := range keys {
			c := &ConfiguredKey{Object: js.Global.Get("Object").New()}
			c.Id = k.Id
			c.Name = k.Name
			result = append(result, c)
		}
		callback(result, nil)
	})
}

func (m *manager) Add(name string, pemPrivateKey string, callback func(err error)) {
	if name == "" {
		callback(errors.New("name must not be empty"))
		return
	}

	m.writeKey(name, pemPrivateKey, func(err error) {
		callback(err)
	})
}

func (m *manager) Remove(id ID, callback func(err error)) {
	m.removeKey(id, func(err error) {
		callback(err)
	})
}

func (m *manager) Loaded(callback func(keys []*LoadedKey, err error)) {
	loaded, err := m.agent.List()
	if err != nil {
		callback(nil, fmt.Errorf("failed to list loaded keys: %v", err))
		return
	}

	var result []*LoadedKey
	for _, l := range loaded {
		k := &LoadedKey{Object: js.Global.Get("Object").New()}
		k.Type = l.Type()
		k.Blob = string(l.Marshal())
		k.Comment = l.Comment
		result = append(result, k)
	}

	callback(result, nil)
}

func (m *manager) Load(id ID, passphrase string, callback func(err error)) {
	m.readKey(id, func(key *storedKey, err error) {
		if err != nil {
			callback(fmt.Errorf("failed to read key: %v", err))
			return
		}

		priv, err := ssh.ParseRawPrivateKeyWithPassphrase([]byte(key.PEMPrivateKey), []byte(passphrase))
		if err != nil {
			callback(fmt.Errorf("failed to parse private key: %v", err))
			return
		}

		err = m.agent.Add(agent.AddedKey{
			PrivateKey: priv,
			Comment:    fmt.Sprintf("%s%s", commentPrefix, id),
		})
		if err != nil {
			callback(fmt.Errorf("failed to add key to agent: %v", err))
			return
		}
		callback(nil)
	})
}
