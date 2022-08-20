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

func TestPromiseResolve(t *testing.T) {
	orig := NewPromise(func(resolve, reject func(v js.Value)) {
		time.Sleep(10 * time.Millisecond) // some blocking function
		resolve(js.ValueOf(2))
	})

	resolved := make(chan struct{})
	defer close(resolved)

	p := AsPromise(orig.JSValue())
	p.Then(
		// Resolve
		func(value js.Value) {
			if diff := cmp.Diff(value.Int(), 2); diff != "" {
				t.Errorf("incorrect number: -got +want: %s", diff)
			}
			resolved <- struct{}{}
		},
		// Reject
		func(reason js.Value) {
			t.Errorf("Reject invoked incorrectly")
		},
	)

	select {
	case <-resolved:
		// Done.
	case <-time.After(5 * time.Second):
		t.Errorf("Resolve not invoked")
	}
}

func TestPromiseReject(t *testing.T) {
	orig := NewPromise(func(resolve, reject func(v js.Value)) {
		time.Sleep(10 * time.Millisecond) // some blocking function
		reject(js.ValueOf(2))
	})

	rejected := make(chan struct{})
	defer close(rejected)

	p := AsPromise(orig.JSValue())
	p.Then(
		// Resolve
		func(value js.Value) {
			t.Errorf("Resolve invoked incorrectly")
		},
		// Reject
		func(reason js.Value) {
			if diff := cmp.Diff(reason.Int(), 2); diff != "" {
				t.Errorf("incorrect number: -got +want: %s", diff)
			}
			rejected <- struct{}{}
		},
	)

	select {
	case <-rejected:
		// Done.
	case <-time.After(5 * time.Second):
		t.Errorf("Reject not invoked")
	}
	return nil
}
