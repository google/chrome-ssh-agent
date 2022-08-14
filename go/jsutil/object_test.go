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

package jsutil

import (
	"sort"
	"syscall/js"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewObject(t *testing.T) {
	o := NewObject()
	if typ := o.Type(); typ != js.TypeObject {
		t.Errorf("expecting TypeObject; got %s", typ)
	}
}

func TestObjectKeys(t *testing.T) {
	o := FromJSON(`{
		"foo": 2,
		"bar": "value"
	}`)
	got, err := ObjectKeys(o)
	if err != nil {
		t.Fatalf("ObjectKeys failed: %v", err)
	}
	sort.Strings(got)
	if diff := cmp.Diff(got, []string{"bar", "foo"}); diff != "" {
		t.Errorf("incorrect result; -got +want: %s", diff)
	}
}
