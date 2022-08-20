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

package storage

import (
	"fmt"
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/chrome"
	"github.com/google/chrome-ssh-agent/go/jsutil"
	"github.com/norunners/vert"
)

// Raw supports storing and retrieving data using Chrome's Storage API.
//
// Raw implements the Area interface.
type Raw struct {
	o js.Value
}

// NewRaw returns a Raw for storing and retrieving data.  The specified area
// must point to an object implmenting the StorageArea API.
func NewRaw(area js.Value) *Raw {
	return &Raw{
		o: area,
	}
}

func dataToValue(data map[string]js.Value) js.Value {
	res := jsutil.NewObject()
	for k, v := range data {
		res.Set(k, v)
	}
	return res
}

func valueToData(val js.Value) (map[string]js.Value, error) {
	keys, err := jsutil.ObjectKeys(val)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %v", err)
	}

	data := map[string]js.Value{}
	for _, k := range keys {
		data[k] = val.Get(k)
	}
	return data, nil
}

// Set implements Area.Set().
func (r *Raw) Set(data map[string]js.Value, callback func(err error)) {
	r.o.Call(
		"set", dataToValue(data),
		jsutil.OneTimeFuncOf(func(this js.Value, args []js.Value) interface{} {
			if err := chrome.LastError(); err != nil {
				callback(fmt.Errorf("failed to set data: %v", err))
				return nil
			}
			callback(nil)
			return nil
		}))
}

// Get implements Area.Get().
func (r *Raw) Get(callback func(data map[string]js.Value, err error)) {
	r.o.Call(
		"get", js.Null(),
		jsutil.OneTimeFuncOf(func(this js.Value, args []js.Value) interface{} {
			if err := chrome.LastError(); err != nil {
				callback(nil, fmt.Errorf("failed to get data: %v", err))
				return nil
			}

			data, err := valueToData(jsutil.SingleArg(args))
			if err != nil {
				callback(nil, fmt.Errorf("failed to parse data: %v", err))
				return nil
			}

			callback(data, nil)
			return nil
		}))
}

// Delete implements Area.Delete().
func (r *Raw) Delete(keys []string, callback func(err error)) {
	if len(keys) <= 0 {
		callback(nil) // Nothing to do.
		return
	}

	r.o.Call(
		"remove", vert.ValueOf(keys).JSValue(),
		jsutil.OneTimeFuncOf(func(this js.Value, args []js.Value) interface{} {
			if err := chrome.LastError(); err != nil {
				callback(fmt.Errorf("failed to delete data: %v", err))
				return nil
			}
			callback(nil)
			return nil
		}))
}
