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
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/jsutil"
)

// Area implementations provide access to underlying storage. The interface is
// a simplified subset of the StorageArea API:
//
//	https://developer.chrome.com/docs/extensions/reference/storage/#type-StorageArea
type Area interface {
	// Set stores new data in storage. data is a map of key-value pairs to
	// be stored. If a key already exists, it will be overwritten.
	Set(ctx jsutil.AsyncContext, data map[string]js.Value) error

	// Get reads all the data items currently stored. The data returned is a
	// map of key-value pairs, with each representing a distinct item from
	// storage.
	Get(ctx jsutil.AsyncContext) (map[string]js.Value, error)

	// Delete removes the items from storage with the specified keys. If a
	// key is not found in storage, it will be silently ignored (i.e., no
	// error will be returned).
	Delete(ctx jsutil.AsyncContext, keys []string) error
}
