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
	"bytes"
	"strings"
	"syscall/js"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/norunners/vert"
)

func objKeys(v js.Value) map[string]bool {
	res := map[string]bool{}
	keys := object.Call("keys", v)
	for i := 0; i < keys.Length(); i++ {
		res[keys.Index(i).String()] = true
	}
	return res
}

func valToString(v js.Value, level int) string {
	var buf bytes.Buffer
	switch v.Type() {
	case js.TypeObject:
		buf.WriteRune('{')
		buf.WriteRune('\n')
		for k := range objKeys(v) {
			buf.WriteString(strings.Repeat(" ", (level+1)*2))
			buf.WriteString(k)
			buf.WriteString(": ")
			buf.WriteString(valToString(v.Get(k), level+1))
			buf.WriteRune('\n')
		}
		buf.WriteString(strings.Repeat(" ", level*2))
		buf.WriteRune('}')
	case js.TypeString:
		buf.WriteRune('"')
		buf.WriteString(v.String())
		buf.WriteRune('"')
	default:
		buf.WriteString(v.String())
	}
	return buf.String()
}

func dataToString(d map[string]js.Value) string {
	var buf bytes.Buffer
	buf.WriteRune('{')
	buf.WriteRune('\n')
	for k, v := range d {
		buf.WriteString("  ")
		buf.WriteString(k)
		buf.WriteString(": ")
		buf.WriteString(valToString(v, 1))
		buf.WriteRune('\n')
	}
	buf.WriteRune('}')
	buf.WriteRune('\n')
	return buf.String()
}

func cmpJSValue(a, b js.Value) bool {
	if a.Type() != b.Type() {
		return false
	}
	switch a.Type() {
	case js.TypeObject:
		aKeys := objKeys(a)
		bKeys := objKeys(b)
		if len(aKeys) != len(bKeys) {
			return false
		}
		for k := range aKeys {
			if _, present := bKeys[k]; !present {
				return false
			}
			if !cmpJSValue(a.Get(k), b.Get(k)) {
				return false
			}
		}
		return true
	default:
		return a.Equal(b)
	}
}

type myStruct struct {
	Field int `js:"field"`
}

func TestDataEncodeAndDecode(t *testing.T) {
	testcases := []struct {
		description string
		data        map[string]js.Value
	}{
		{
			description: "empty data",
			data:        map[string]js.Value{},
		},
		{
			description: "simple entry",
			data: map[string]js.Value{
				"key": js.ValueOf(2),
			},
		},
		{
			description: "map entry",
			data: map[string]js.Value{
				"key": vert.ValueOf(map[string]int{
					"field": 2,
				}).JSValue(),
			},
		},
		{
			description: "object entry",
			data: map[string]js.Value{
				"key": vert.ValueOf(&myStruct{
					Field: 2,
				}).JSValue(),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			t.Logf("Original: %s", dataToString(tc.data))
			val := dataToValue(tc.data)
			t.Logf("Encoded: %s", valToString(val, 0))
			got, err := valueToData(val)
			t.Logf("Decoded: %s", dataToString(got))
			if err != nil {
				t.Fatalf("parsing failed: %v", err)
			}

			if diff := cmp.Diff(dataToString(got), dataToString(tc.data)); diff != "" {
				t.Errorf("incorrect data; -got +want: %s", diff)
			}
		})
	}
}
