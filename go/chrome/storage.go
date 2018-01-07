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

package chrome

import (
	"fmt"

	"github.com/gopherjs/gopherjs/js"
)

// Storage supports storing and retrieving data using Chrome's Storage API.
type Storage struct {
	chrome *C
	o      *js.Object
}

// Set stores new data in storage. data is a map of key-value pairs to be
// stored. If a key already exists, it will be overwritten.  Callback will
// be invoked when complete.
//
// See set() in https://developer.chrome.com/apps/storage#type-StorageArea.
func (s *Storage) Set(data map[string]interface{}, callback func(err error)) {
	s.o.Call("set", data, func() {
		if err := s.chrome.Error(); err != nil {
			callback(fmt.Errorf("failed to set data: %v", err))
			return
		}
		callback(nil)
	})
}

// Get reads all the data items currently stored.  The callback will be
// invoked when complete, suppliing the items read and indicating any errors.
// The data suppiled with the callback is a map of key-value pairs, with
// each representing a distinct item from storage.
//
// See get() in https://developer.chrome.com/apps/storage#type-StorageArea.
func (s *Storage) Get(callback func(data map[string]interface{}, err error)) {
	s.o.Call("get", nil, func(vals interface{}) {
		if err := s.chrome.Error(); err != nil {
			callback(nil, fmt.Errorf("failed to get data: %v", err))
			return
		}

		callback(vals.(map[string]interface{}), nil)
	})
}

// Delete removes the items from storage with the specified keys. If a key is
// not found in storage, it will be silently ignored (i.e., no error will be
// returned). Callback is invoked when complete.
//
// See remove() in https://developer.chrome.com/apps/storage#type-StorageArea.
func (s *Storage) Delete(keys []string, callback func(err error)) {
	s.o.Call("remove", keys, func() {
		if err := s.chrome.Error(); err != nil {
			callback(fmt.Errorf("failed to delete data: %v", err))
			return
		}

		callback(nil)
	})
}
