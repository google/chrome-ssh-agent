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
	"fmt"
	"strings"
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/jsutil"
)

// View supports storing and retrieving keys and values with particular key
// prefix.  This allows multiple usages of the same underlying datastore,
// without them trampling on each other.
//
// It is the caller's responsibility to supply unique key prefixes as required.
type View struct {
	// prefixes are the prefix prepended to each key. The prefix disambiguates
	// entries for this view from entries for a different view.
	//
	// When there is more than one prefix, we always apply an operation to
	// each prefix. This helps support migration of data between prefixes:
	//   Step 0: Read/write data under only the old prefix.
	//   Step 1: Read/write data under old and new prefixes. This supports
	//     older and newer versions concurrently.
	//   [let sufficient time pass for older version to go away]
	//   Step 2: Read/write data under new prefix.
	//   Step 3: Delete data under old prefix to free up space.
	//
	// Prefixes are in preference order. That is, when reading, if a key is
	// present under multiple prefixes, the first-appearing prefix in this
	// list takes precedence.
	prefixes []string

	// s is the underlying storage area.
	s Area
}

// NewView returns a view of a storage area with a given set of key prefixes.
func NewView(prefixes []string, store Area) *View {
	var prefixesAdj []string
	for _, p := range prefixes {
		prefixesAdj = append(prefixesAdj, p+".")
	}

	return &View{
		prefixes: prefixesAdj,
		s:        store,
	}
}

// readKey detects if the key belongs to our view.
func (v *View) readKey(prefix, key string) (string, bool) {
	return strings.TrimPrefix(key, prefix), strings.HasPrefix(key, prefix)
}

func (v *View) makeKey(prefix, key string) string {
	return prefix + key
}

// Set implements Area.Set().
func (v *View) Set(ctx jsutil.AsyncContext, data map[string]js.Value) error {
	ndata := map[string]js.Value{}
	for k, val := range data {
		for _, prefix := range v.prefixes {
			ndata[v.makeKey(prefix, k)] = val
		}
	}
	return v.s.Set(ctx, ndata)
}

// Get implements Area.Get().
func (v *View) Get(ctx jsutil.AsyncContext) (map[string]js.Value, error) {
	data, err := v.s.Get(ctx)
	if err != nil {
		return nil, err
	}

	ndata := map[string]js.Value{}
	for k, val := range data {
		for _, prefix := range v.prefixes {
			sk, ok := v.readKey(prefix, k)
			if !ok {
				continue
			}

			if _, present := ndata[sk]; !present {
				ndata[sk] = val
				continue // First takes precedence; stop.
			}
		}
	}
	return ndata, nil
}

// Delete implements Area.Delete().
func (v *View) Delete(ctx jsutil.AsyncContext, keys []string) error {
	var nkeys []string
	for _, k := range keys {
		for _, prefix := range v.prefixes {
			nkeys = append(nkeys, v.makeKey(prefix, k))
		}
	}
	return v.s.Delete(ctx, nkeys)
}

// DeleteViewPrefixes deletes all storage entries for views with the given prefixes.
func DeleteViewPrefixes(ctx jsutil.AsyncContext, prefixes []string, store Area) error {
	v := NewView(prefixes, store)

	// Gather all of the keys.
	data, err := v.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get keys: %v", err)
	}
	var keys []string
	for k := range data {
		keys = append(keys, k)
	}

	// Remove them all.
	if err := v.Delete(ctx, keys); err != nil {
		return fmt.Errorf("failed to delete keys: %v", err)
	}

	return nil
}
