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
	// prefix is the prefix prepended to each key. The prefix disambiguates
	// entries for this view from entries for a different view.
	prefix string

	// s is the underlying storage area.
	s Area
}

// NewView returns a view of a storage area with a given key prefix.
func NewView(prefix string, store Area) *View {
	return &View{
		prefix: prefix + ".",
		s:      store,
	}
}

// isViewKey detects if the key belongs to our view.
func (v *View) readKey(key string) (string, bool) {
	return strings.TrimPrefix(key, v.prefix), strings.HasPrefix(key, v.prefix)
}

func (v *View) makeKey(key string) string {
	return v.prefix + key
}

// Set implements Area.Set().
func (v *View) Set(ctx jsutil.AsyncContext, data map[string]js.Value) error {
	ndata := map[string]js.Value{}
	for k, val := range data {
		ndata[v.makeKey(k)] = val
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
		if sk, ok := v.readKey(k); ok {
			ndata[sk] = val
		}
	}
	return ndata, nil
}

// Delete implements Area.Delete().
func (v *View) Delete(ctx jsutil.AsyncContext, keys []string) error {
	var nkeys []string
	for _, k := range keys {
		nkeys = append(nkeys, v.makeKey(k))
	}
	return v.s.Delete(ctx, nkeys)
}
