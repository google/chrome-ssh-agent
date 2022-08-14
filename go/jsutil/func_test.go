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
	"syscall/js"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestOneTimeFuncOf(t *testing.T) {
	got := make(chan struct{}, 1)
	f := OneTimeFuncOf(func(this js.Value, args []js.Value) interface{} {
		got <- struct{}{}
		return nil
	})

	f.Invoke()
	select {
	case <-got: // nothing to do.
	case <-time.After(5 * time.Second):
		t.Errorf("function not invoked")
	}
}

func TestDefineFunc(t *testing.T) {
	const funcName = "myFunc"

	o := NewObject()

	got := make(chan struct{}, 1)
	cleanup := DefineFunc(o, funcName, func(this js.Value, args []js.Value) interface{} {
		got <- struct{}{}
		return nil
	})

	// Ensure function has been defined.
	if typ := o.Get(funcName).Type(); typ != js.TypeFunction {
		t.Errorf("defined value is not a function; got %s", typ)
	}

	// Call the function once defined. Clean it up once called.
	func() {
		defer cleanup()

		o.Call(funcName)
		select {
		case <-got: // nothing to do.
		case <-time.After(5 * time.Second):
			t.Errorf("function not invoked")
		}
	}()

	// After cleanup, function should not be defined.
	if !o.Get(funcName).IsUndefined() {
		t.Errorf("function still defined after cleanup")
	}
}

func TestSetTimeout(t *testing.T) {
	got := make(chan struct{}, 1)
	SetTimeout(1*time.Millisecond, func() {
		got <- struct{}{}
	})

	select {
	case <-got: // nothing to do.
	case <-time.After(5 * time.Second):
		t.Errorf("function not invoked")
	}
}

func TestExpandArgs(t *testing.T) {
	testcases := []struct {
		description string
		args        []js.Value
		targets     int
		want        []js.Value
	}{
		{
			description: "all args matched",
			args: []js.Value{
				js.ValueOf(2),
				js.ValueOf("foo"),
			},
			targets: 2,
			want: []js.Value{
				js.ValueOf(2),
				js.ValueOf("foo"),
			},
		},
		{
			description: "fewer targets than args",
			args: []js.Value{
				js.ValueOf(2),
				js.ValueOf("foo"),
			},
			targets: 1,
			want: []js.Value{
				js.ValueOf(2),
			},
		},
		{
			description: "more targets than args",
			args: []js.Value{
				js.ValueOf(2),
				js.ValueOf("foo"),
			},
			targets: 3,
			want: []js.Value{
				js.ValueOf(2),
				js.ValueOf("foo"),
				js.Undefined(),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			var got []*js.Value
			for i := 0; i < tc.targets; i++ {
				got = append(got, &js.Value{})
			}
			ExpandArgs(tc.args, got...)

			// Convert to JSON for comparison
			var gotJSON, wantJSON []string
			for _, j := range got {
				gotJSON = append(gotJSON, ToJSON(*j))
			}
			for _, j := range tc.want {
				wantJSON = append(wantJSON, ToJSON(j))
			}
			if diff := cmp.Diff(gotJSON, wantJSON); diff != "" {
				t.Errorf("incorrect result; -got +want: %s", diff)
			}
		})
	}
}

func TestSingleArgs(t *testing.T) {
	testcases := []struct {
		description string
		args        []js.Value
		want        js.Value
	}{
		{
			description: "no args",
			args:        []js.Value{},
			want:        js.Undefined(),
		},
		{
			description: "single arg",
			args: []js.Value{
				js.ValueOf(2),
			},
			want: js.ValueOf(2),
		},
		{
			description: "multiple args",
			args: []js.Value{
				js.ValueOf(2),
				js.ValueOf("foo"),
			},
			want: js.ValueOf(2),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			got := SingleArg(tc.args)
			if diff := cmp.Diff(ToJSON(got), ToJSON(tc.want)); diff != "" {
				t.Errorf("incorrect result; -got +want: %s", diff)
			}
		})
	}
}
