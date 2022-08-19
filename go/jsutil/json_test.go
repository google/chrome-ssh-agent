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
	"syscall/js"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestToJSON(t *testing.T) {
	testcases := []struct {
		description string
		val         js.Value
		want        string
	}{
		{
			description: "null",
			val:         js.Null(),
			want:        "null",
		},
		{
			description: "simple value: number",
			val:         js.ValueOf(2),
			want:        "2",
		},
		{
			description: "composite value: map",
			val: js.ValueOf(map[string]interface{}{
				"my-key": "my-val",
			}),
			want: `{"my-key":"my-val"}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			if diff := cmp.Diff(ToJSON(tc.val), tc.want); diff != "" {
				t.Errorf("incorrect result; -got +want: %s", diff)
			}
		})
	}
}

func TestFromJSON(t *testing.T) {
	testcases := []struct {
		description string
		val         string
		want        js.Value
	}{
		{
			description: "null",
			val:         "null",
			want:        js.Null(),
		},
		{
			description: "simple value: number",
			val:         "2",
			want:        js.ValueOf(2),
		},
		{
			description: "composite value: map",
			val:         `{"my-key":"my-val"}`,
			want: js.ValueOf(map[string]interface{}{
				"my-key": "my-val",
			}),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			// We assume that ToJSON works per the tests above. We can
			// compare values by their JSON representations.
			if diff := cmp.Diff(ToJSON(FromJSON(tc.val)), ToJSON(tc.want)); diff != "" {
				t.Errorf("incorrect result; -got +want: %s", diff)
			}
		})
	}
}
