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
	"errors"
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/jsutil"
	"github.com/norunners/vert"
)

const (
	// valueKey is the key under which the singular value is stored.
	valueKey = "current"
)

var (
	errParse = errors.New("parse failed")
)

// Value reads and writes a singular value.
type Value[V any] struct {
	store Area
}

// NewValue returns a new Value using the underlying persistent store.
// keyPrefix is the prefix used to distinguish values from others in the same
// underlying store; multiple may be supplied to support migration scenarios.
func NewValue[V any](store Area, keyPrefix []string) *Value[V] {
	return &Value[V]{
		store: NewView(keyPrefix, store),
	}
}

// Get reads the current value from storage.  If it doesn't exist, the zero
// value is returned.
func (v *Value[V]) Get(ctx jsutil.AsyncContext) (V, error) {
	var zero V

	data, err := v.store.Get(ctx)
	if err != nil {
		return zero, err
	}

	val, present := data[valueKey]
	if !present {
		return zero, nil
	}

	var tv V
	if err := vert.ValueOf(val).AssignTo(&tv); err != nil {
		return zero, errParse
	}

	return tv, nil
}

// Set writes a new value to storage.
func (v *Value[V]) Set(ctx jsutil.AsyncContext, val V) error {
	data := map[string]js.Value{
		valueKey: vert.ValueOf(val).JSValue(),
	}
	return v.store.Set(ctx, data)
}
