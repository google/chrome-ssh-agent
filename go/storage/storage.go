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

package storage

import (
	"fmt"

	"github.com/gopherjs/gopherjs/js"
	"github.com/ralimi/chrome-ssh-agent/go/chrome"
)

type Storage struct {
	o *js.Object
}

func (s *Storage) Set(data map[string]interface{}, callback func(err error)) {
	// Convert data to a JS object so it is ready for storage.
	o := js.Global.Get("Object").New()
	for k, v := range data {
		o.Set(k, v)
	}

	s.o.Call("set", o, func() {
		if err := chrome.LastError(); err != nil {
			callback(fmt.Errorf("failed to set data: %v", err))
			return
		}
		callback(nil)
	})
}

func (s *Storage) Get(callback func(data map[string]interface{}, err error)) {
	s.o.Call("get", nil, func(vals interface{}) {
		if err := chrome.LastError(); err != nil {
			callback(nil, fmt.Errorf("failed to get data: %v", err))
			return
		}

		callback(vals.(map[string]interface{}), nil)
	})
}

func (s *Storage) Delete(keys []string, callback func(err error)) {
	s.o.Call("remove", keys, func() {
		if err := chrome.LastError(); err != nil {
			callback(fmt.Errorf("failed to delete data: %v", err))
			return
		}

		callback(nil)
	})
}

func New() *Storage {
	return &Storage{
		o: chrome.SyncStorage,
	}
}
