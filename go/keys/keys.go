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
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/gopherjs/gopherjs/js"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type Available interface {
	Available(callback func(keys []string, err error))
	Add(name string, pemPrivateKey string, callback func(err error))
	Remove(name string, callback func(err error))
	Loaded(callback func(keys []string, err error))
	Load(name, passphrase string, callback func(err error))
}

func New(a agent.Agent) Available {
	return &available{
		a: a,
		s: NewStorage(),
	}
}

type available struct {
	a agent.Agent
	s *Storage
}

type availableKey struct {
	*js.Object
	ID            string `js:"id"`
	Name          string `js:"name"`
	PEMPrivateKey string `js:"pemPrivateKey"`
	AuthorizedKey string `js:"authorizedKey"`
}

const (
	keyPrefix = "key."
)

func newAvailableKey(m map[string]interface{}) *availableKey {
	o := js.Global.Get("Object").New()
	for k, v := range m {
		o.Set(k, v)
	}
	return &availableKey{Object: o}
}

func (a *available) readKeys(callback func(keys []*availableKey, err error)) {
	a.s.Get(func(data map[string]interface{}, err error) {
		if err != nil {
			callback(nil, fmt.Errorf("failed to read from storage: %v", err))
			return
		}

		var keys []*availableKey
		for k, v := range data {
			if !strings.HasPrefix(k, keyPrefix) {
				continue
			}

			keys = append(keys, newAvailableKey(v.(map[string]interface{})))
		}
		callback(keys, nil)
	})
}

func (a *available) readKey(name string, callback func(key *availableKey, err error)) {
	a.readKeys(func(keys []*availableKey, err error) {
		if err != nil {
			callback(nil, fmt.Errorf("failed to read keys: %v", err))
			return
		}

		for _, k := range keys {
			if k.Name == name {
				callback(k, nil)
				return
			}
		}

		callback(nil, nil)
	})
}

func (a *available) writeKey(name string, pemPrivateKey string, callback func(err error)) {
	id := strconv.FormatUint(rand.Uint64(), 16)
	storageKey := fmt.Sprintf("%s%s", keyPrefix, id)
	ak := &availableKey{Object: js.Global.Get("Object").New()}
	ak.ID = id
	ak.Name = name
	ak.PEMPrivateKey = pemPrivateKey
	data := map[string]interface{}{
		storageKey: ak,
	}
	a.s.Set(data, func(err error) {
		callback(err)
	})
}

func (a *available) removeKey(name string, callback func(err error)) {
	a.readKeys(func(keys []*availableKey, err error) {
		if err != nil {
			callback(fmt.Errorf("failed to enumerate keys: %v", err))
			return
		}

		var deleteKeys []string
		for _, k := range keys {
			if k.Name == name {
				deleteKeys = append(deleteKeys, fmt.Sprintf("%s%s", keyPrefix, k.ID))
			}
		}

		a.s.Delete(deleteKeys, func(err error) {
			if err != nil {
				callback(fmt.Errorf("failed to delete keys: %v", err))
				return
			}
			callback(nil)
		})
	})
}

func (a *available) Available(callback func(keys []string, err error)) {
	a.readKeys(func(keys []*availableKey, err error) {
		if err != nil {
			callback(nil, fmt.Errorf("failed to read keys: %v", err))
			return
		}

		var result []string
		for _, k := range keys {
			result = append(result, k.Name)
		}
		callback(result, nil)
	})
}

func (a *available) Add(name string, pemPrivateKey string, callback func(err error)) {
	if name == "" {
		callback(errors.New("name must not be empty"))
		return
	}

	a.writeKey(name, pemPrivateKey, func(err error) {
		callback(err)
	})
}

func (a *available) Remove(name string, callback func(err error)) {
	a.removeKey(name, func(err error) {
		callback(err)
	})
}

func (a *available) Loaded(callback func(keys []string, err error)) {
	keys, err := a.a.List()
	if err != nil {
		callback(nil, fmt.Errorf("failed to list loaded keys: %v", err))
		return
	}

	var result []string
	for _, k := range keys {
		result = append(result, k.Comment)
	}
	callback(result, nil)
}

func (a *available) Load(name, passphrase string, callback func(err error)) {
	a.readKey(name, func(key *availableKey, err error) {
		if err != nil {
			callback(fmt.Errorf("failed to read key %s: %v", name, err))
			return
		}

		priv, err := ssh.ParseRawPrivateKeyWithPassphrase([]byte(key.PEMPrivateKey), []byte(passphrase))
		if err != nil {
			callback(fmt.Errorf("failed to parse private key: %v", err))
			return
		}

		err = a.a.Add(agent.AddedKey{
			PrivateKey: priv,
			Comment:    name,
		})
		if err != nil {
			callback(fmt.Errorf("failed to add key to agent: %v", err))
			return
		}
		callback(nil)
	})
}
