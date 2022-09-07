//go:build js

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

package storage

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/jsutil"
	"github.com/norunners/vert"
)

// Typed reads and writes typed values. They are serialized upon writing,
// and deserialized upon reading.  If deserialization fails for a given value,
// it is ignored.
type Typed[V any] struct {
	store Area
}

// NewTyped returns a new Typed using the underlying persistent store.
// keyPrefix is the prefix used to distinguish values from others in the same
// underlying store; multiple may be supplied to support migration scenarios.
func NewTyped[V any](store Area, keyPrefix []string) *Typed[V] {
	return &Typed[V]{
		store: NewView(keyPrefix, store),
	}
}

// readAllItems returns all the stored values, along with their keys.
func (t *Typed[V]) readAllItems(ctx jsutil.AsyncContext) (map[string]*V, error) {
	data, err := t.store.Get(ctx)
	if err != nil {
		return nil, err
	}

	values := map[string]*V{}
	for k, v := range data {
		var tv V
		if err := vert.ValueOf(v).AssignTo(&tv); err != nil {
			jsutil.LogError("failed to parse value %s; dropping", k)
			continue
		}

		values[k] = &tv
	}
	return values, nil
}

// ReadAll returns all the stored values.
func (t *Typed[V]) ReadAll(ctx jsutil.AsyncContext) ([]*V, error) {
	data, err := t.readAllItems(ctx)
	if err != nil {
		return nil, err
	}

	var values []*V
	for _, v := range data {
		values = append(values, v)
	}
	return values, nil
}

// Read returns a single value that matches the supplied test function. If
// multiple values match, only the first is returned. If the value is not found,
// a nil value is returned.
func (t *Typed[V]) Read(ctx jsutil.AsyncContext, test func(v *V) bool) (*V, error) {
	values, err := t.ReadAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, v := range values {
		if test(v) {
			return v, nil
		}
	}

	return nil, nil
}

// Write writes a new value to storage.
func (t *Typed[V]) Write(ctx jsutil.AsyncContext, value *V) error {
	// Generate a unique key under which value will be stored.
	key, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return fmt.Errorf("failed to generate new ID: %w", err)
	}
	data := map[string]js.Value{
		key.String(): vert.ValueOf(value).JSValue(),
	}
	return t.store.Set(ctx, data)
}

// Delete removes the value that matches the supplied test function. If multiple
// values match, all matching values are removed.
func (t *Typed[V]) Delete(ctx jsutil.AsyncContext, test func(v *V) bool) error {
	data, err := t.readAllItems(ctx)
	if err != nil {
		return fmt.Errorf("failed to enumerate values: %w", err)
	}

	var keys []string
	for k, v := range data {
		if test(v) {
			keys = append(keys, k)
		}
	}

	return t.store.Delete(ctx, keys)
}
