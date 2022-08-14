//go:build js && wasm

// Copyright 2022 Google LLC
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
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"strings"
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/jsutil"
	"github.com/norunners/vert"
)

// TypedStore reads and writes typed values. They are serialized upon writing,
// and deserialized upon reading.  If deserialization fails for a given value,
// it is ignored.
type TypedStore[V any] struct {
	store     PersistentStore
	keyPrefix string
}

// NewTypedStore returns a new TypedStore using the underlying persistent store.
// keyPrefix is the prefix used to distinguish values from others in the same
// underlying store.
func NewTypedStore[V any](store PersistentStore, keyPrefix string) *TypedStore[V] {
	return &TypedStore[V]{
		store:     store,
		keyPrefix: keyPrefix,
	}
}

// readAllItems returns all the stored values, along with their keys. callback
// is invoked with the result.
func (t *TypedStore[V]) readAllItems(callback func(data map[string]*V, err error)) {
	t.store.Get(func(data map[string]js.Value, err error) {
		if err != nil {
			callback(nil, err)
			return
		}

		values := map[string]*V{}
		for k, v := range data {
			if !strings.HasPrefix(k, t.keyPrefix) {
				continue
			}

			var tv V
			if err := vert.ValueOf(v).AssignTo(&tv); err != nil {
				jsutil.LogError("failed to parse value %s; dropping", k)
				continue
			}

			values[k] = &tv
		}
		callback(values, nil)
	})
}

// ReadAll returns all the stored values. callback is invoked when complete.
func (t *TypedStore[V]) ReadAll(callback func(values []*V, err error)) {
	t.readAllItems(func(data map[string]*V, err error) {
		if err != nil {
			callback(nil, err)
			return
		}

		var values []*V
		for _, v := range data {
			values = append(values, v)
		}
		callback(values, nil)
	})
}

// Read returns a single value that matches the supplied test function. If
// multiple values match, only the first is returned.  callback is invoked with
// the returned value. If the value is not found, then the callback is invoked
// with a nil value.
func (t *TypedStore[V]) Read(test func(v *V) bool, callback func(value *V, err error)) {
	t.ReadAll(func(values []*V, err error) {
		if err != nil {
			callback(nil, err)
			return
		}

		for _, v := range values {
			if test(v) {
				callback(v, nil)
				return
			}
		}

		callback(nil, nil)
	})
}

// Write writes a new value to storage. callback is invoked when complete.
func (t *TypedStore[V]) Write(value *V, callback func(err error)) {
	// Generate a unique key under which value will be stored.
	i, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		callback(fmt.Errorf("failed to generate new ID: %w", err))
		return
	}
	key := fmt.Sprintf("%s%s", t.keyPrefix, i)

	data := map[string]js.Value{
		key: vert.ValueOf(value).JSValue(),
	}
	t.store.Set(data, callback)
}

// Delete removes the value that matches the supplied test function. If multiple
// values match, all matching values are removed. callback is invoked upon
// completion.
func (t *TypedStore[V]) Delete(test func(v *V) bool, callback func(err error)) {
	t.readAllItems(func(data map[string]*V, err error) {
		if err != nil {
			callback(fmt.Errorf("failed to enumerate values: %w", err))
			return
		}

		var keys []string
		for k, v := range data {
			if test(v) {
				keys = append(keys, k)
			}
		}

		t.store.Delete(keys, callback)
	})
}
