//go:build js

// Copyright 2018 Google LLC
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

// Package fakes implements fake implementations of Chrome's extension APIs to
// ease unit testing.
package fakes

import (
	"syscall/js"
)

// Errs contains errors that should be returned by the fake implementation.
type Errs struct {
	// Get is the error that should be returned by Storage.Get().
	Get error
	// Set is the error that should be returned by Storage.Set().
	Set error
	// Delete is the error that should be returned by Storage.Delete().
	Delete error
}

// MemStorage is a fake implementation of Chrome's storage API.
type MemStorage struct {
	data map[string]js.Value
	err  Errs
}

// NewMemStorage returns a fake implementation of Chrome's storage API.
func NewMemStorage() *MemStorage {
	return &MemStorage{
		data: make(map[string]js.Value),
	}
}

// SetError specifies the errors that should be returned from various
// operations.  Forcing the fake implementation to return errors is
// useful to test error conditions in unit tests.
func (m *MemStorage) SetError(err Errs) {
	m.err = err
}

// Set is a fake implmentation of chrome.Storage.Set().
func (m *MemStorage) Set(data map[string]js.Value, callback func(err error)) {
	if m.err.Set != nil {
		callback(m.err.Set)
		return
	}

	for k, v := range data {
		m.data[k] = v
	}
	callback(nil)
}

// Get is a fake implmentation of chrome.Storage.Get().
func (m *MemStorage) Get(callback func(data map[string]js.Value, err error)) {
	if m.err.Get != nil {
		callback(nil, m.err.Get)
		return
	}

	// TODO(ralimi) Make a copy.
	callback(m.data, nil)
}

// Delete is a fake implmentation of chrome.Storage.Delete().
func (m *MemStorage) Delete(keys []string, callback func(err error)) {
	if m.err.Delete != nil {
		callback(m.err.Delete)
		return
	}

	for _, k := range keys {
		delete(m.data, k)
	}
	callback(nil)
}
