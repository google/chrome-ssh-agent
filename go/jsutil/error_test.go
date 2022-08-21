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
	"errors"
	"syscall/js"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewError(t *testing.T) {
	testcases := []struct {
		description string
		err         error
		want        string
	}{
		{
			description: "Go error",
			err:         errors.New("some error"),
			want:        "GoError: some error",
		},
		{
			description: "Preserve JSError",
			err:         NewErrorFromVal(js.ValueOf("some error")),
			want:        "Error: some error",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			got := NewError(tc.err)
			if diff := cmp.Diff(got.Error(), tc.want); diff != "" {
				t.Errorf("incorrect result; -got +want: %s", diff)
			}
		})
	}
}

func TestNewErrorFromVal(t *testing.T) {
	testcases := []struct {
		description string
		val         js.Value
		want        string
	}{
		{
			description: "From string",
			val:         js.ValueOf("some error"),
			want:        "Error: some error",
		},
		{
			description: "preserve Error value",
			val:         js.Global().Get("Error").New("generic error"),
			want:        "Error: generic error",
		},
		{
			description: "preserve value derived from Error",
			val:         js.Global().Get("RangeError").New("my range error"),
			want:        "RangeError: my range error",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			got := NewErrorFromVal(tc.val)
			if diff := cmp.Diff(got.Error(), tc.want); diff != "" {
				t.Errorf("incorrect result; -got +want: %s", diff)
			}
		})
	}
}

func TestErrorProperties(t *testing.T) {
	e := NewErrorFromVal(js.Global().Get("RangeError").New("my range error"))
	if diff := cmp.Diff(e.Name(), "RangeError"); diff != "" {
		t.Errorf("incorrect name: -got +want: %s", diff)
	}
	if diff := cmp.Diff(e.Message(), "my range error"); diff != "" {
		t.Errorf("incorrect message: -got +want: %s", diff)
	}
}
