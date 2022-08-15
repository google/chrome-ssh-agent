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

package storage

import (
	"syscall/js"
	"testing"

	"github.com/google/chrome-ssh-agent/go/jsutil"
	"github.com/google/go-cmp/cmp"
	"github.com/norunners/vert"
)

func dataToJSON(data map[string]js.Value) map[string]string {
	json := map[string]string{}
	for k, v := range data {
		json[k] = jsutil.ToJSON(v)
	}
	return json
}

type myStruct struct {
	IntField    int    `js:"intField"`
	StringField string `js:"stringField"`
}

func myStructLess(a *myStruct, b *myStruct) bool {
	if a.IntField != b.IntField {
		return a.IntField < b.IntField
	}
	return a.StringField < b.StringField
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
					IntField: 2,
				}).JSValue(),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			val := dataToValue(tc.data)
			got, err := valueToData(val)
			if err != nil {
				t.Fatalf("parsing failed: %v", err)
			}

			if diff := cmp.Diff(dataToJSON(got), dataToJSON(tc.data)); diff != "" {
				t.Errorf("incorrect data; -got +want: %s", diff)
			}
		})
	}
}
