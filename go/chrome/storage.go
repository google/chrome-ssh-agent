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

package chrome

import (
	"fmt"
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/dom"
	"github.com/norunners/vert"
)

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

// Storage supports storing and retrieving data using Chrome's Storage API.
//
// Storage implements the PersistentStore interface.
type Storage struct {
	chrome *C
	o      js.Value
}

func dataToValue(data map[string]js.Value) js.Value {
	res := dom.Object.New()
	for k, v := range data {
		res.Set(k, v)
	}
	return res
}

func valueToData(val js.Value) (map[string]js.Value, error) {
	keys, err := dom.ObjectKeys(val)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %v", err)
	}

	data := map[string]js.Value{}
	for _, k := range keys {
		data[k] = val.Get(k)
	}
	return data, nil
}

// Set stores new data in storage. data is a map of key-value pairs to be
// stored. If a key already exists, it will be overwritten.  Callback will
// be invoked when complete.
//
// See set() in https://developer.chrome.com/apps/storage#type-StorageArea.
func (s *Storage) Set(data map[string]js.Value, callback func(err error)) {
	var cb js.Func
	cb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer cb.Release() // One-time callback; release when done.
		if err := s.chrome.Error(); err != nil {
			callback(fmt.Errorf("failed to set data: %v", err))
			return nil
		}
		callback(nil)
		return nil
	})
	s.o.Call("set", dataToValue(data), cb)
}

// Get reads all the data items currently stored.  The callback will be
// invoked when complete, suppliing the items read and indicating any errors.
// The data suppiled with the callback is a map of key-value pairs, with
// each representing a distinct item from storage.
//
// See get() in https://developer.chrome.com/apps/storage#type-StorageArea.
func (s *Storage) Get(callback func(data map[string]js.Value, err error)) {
	var cb js.Func
	cb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer cb.Release() // One-time callback; release when done.
		if err := s.chrome.Error(); err != nil {
			callback(nil, fmt.Errorf("failed to get data: %v", err))
			return nil
		}

		data, err := valueToData(dom.SingleArg(args))
		if err != nil {
			callback(nil, fmt.Errorf("failed to parse data: %v", err))
			return nil
		}

		callback(data, nil)
		return nil
	})
	s.o.Call("get", js.Null(), cb)
}

// Delete removes the items from storage with the specified keys. If a key is
// not found in storage, it will be silently ignored (i.e., no error will be
// returned). Callback is invoked when complete.
//
// See remove() in https://developer.chrome.com/apps/storage#type-StorageArea.
func (s *Storage) Delete(keys []string, callback func(err error)) {
	var cb js.Func
	cb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer cb.Release() // One-time callback; release when done.
		if err := s.chrome.Error(); err != nil {
			callback(fmt.Errorf("failed to delete data: %v", err))
			return nil
		}
		callback(nil)
		return nil
	})
	s.o.Call("remove", vert.ValueOf(keys).JSValue(), cb)
}
