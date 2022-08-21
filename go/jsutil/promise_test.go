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
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestAsPromise(t *testing.T) {
	orig := NewPromise(func(ctx AsyncContext, resolve ResolveFunc, reject RejectFunc) {
		resolve(js.Null())
	})
	p := AsPromise(orig.JSValue())

	done := make(chan struct{})
	p.Then(
		func(val js.Value) { close(done) },
		func(err error) { close(done) },
	)
	<-done
}

func TestPromiseResolve(t *testing.T) {
	p := NewPromise(func(ctx AsyncContext, resolve ResolveFunc, reject RejectFunc) {
		time.Sleep(10 * time.Millisecond) // some blocking function
		resolve(js.ValueOf(2))
	})

	resolved := make(chan struct{})
	p.Then(
		// Resolve
		func(value js.Value) {
			if diff := cmp.Diff(value.Int(), 2); diff != "" {
				t.Errorf("incorrect number: -got +want: %s", diff)
			}
			close(resolved)
		},
		// Reject
		func(err error) {
			t.Errorf("Reject invoked incorrectly with error %v", err)
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
	orig := NewPromise(func(ctx AsyncContext, resolve ResolveFunc, reject RejectFunc) {
		time.Sleep(10 * time.Millisecond) // some blocking function
		reject(errors.New("my error"))
	})

	rejected := make(chan struct{})
	p := AsPromise(orig.JSValue())
	p.Then(
		// Resolve
		func(value js.Value) {
			t.Errorf("Resolve invoked incorrectly with value: %s", value)
		},
		// Reject
		func(err error) {
			if diff := cmp.Diff(err.Error(), "GoError: my error"); diff != "" {
				t.Errorf("incorrect error: -got +want: %s", diff)
			}
			close(rejected)
		},
	)

	select {
	case <-rejected:
		// Done.
	case <-time.After(5 * time.Second):
		t.Errorf("Reject not invoked")
	}
}

func TestAsyncSuccess(t *testing.T) {
	p := Async(func(ctx AsyncContext) (js.Value, error) {
		return js.ValueOf(2), nil
	})

	resolved := make(chan struct{}, 1)
	p.Then(
		// Resolve
		func(value js.Value) {
			if diff := cmp.Diff(value.Int(), 2); diff != "" {
				t.Errorf("incorrect number: -got +want: %s", diff)
			}
			close(resolved)
		},
		// Reject
		func(err error) {
			t.Errorf("Reject invoked incorrectly with error %v", err)
		},
	)

	select {
	case <-resolved:
		// Done.
	case <-time.After(5 * time.Second):
		t.Errorf("Resolve not invoked")
	}
}

func TestAsyncError(t *testing.T) {
	p := Async(func(ctx AsyncContext) (js.Value, error) {
		return js.Null(), errors.New("my error")
	})

	rejected := make(chan struct{})
	p.Then(
		// Resolve
		func(value js.Value) {
			t.Errorf("Resolve invoked incorrectly with value: %s", value)
		},
		// Reject
		func(err error) {
			if diff := cmp.Diff(err.Error(), "GoError: my error"); diff != "" {
				t.Errorf("incorrect error: -got +want: %s", diff)
			}
			close(rejected)
		},
	)

	select {
	case <-rejected:
		// Done.
	case <-time.After(5 * time.Second):
		t.Errorf("Reject not invoked")
	}
}

func TestAwait(t *testing.T) {
	p := Async(func(ctx AsyncContext) (js.Value, error) {
		// Function that returns success
		val, err := Async(func(ctx AsyncContext) (js.Value, error) {
			return js.ValueOf(2), nil
		}).Await(ctx)
		if diff := cmp.Diff(val.Int(), 2); diff != "" {
			t.Errorf("incorrect result: -got +want: %s", diff)
		}
		if err != nil {
			t.Errorf("incorrect error; got %v", err)
		}

		// Function that returns an error
		val, err = Async(func(ctx AsyncContext) (js.Value, error) {
			return js.ValueOf(2), errors.New("my error")
		}).Await(ctx)
		if !val.IsUndefined() {
			t.Errorf("incorrect result: got %s", val)
		}
		if diff := cmp.Diff(err.Error(), "GoError: my error"); diff != "" {
			t.Errorf("incorrect error; -got +want: %s", diff)
		}

		return js.ValueOf("done!"), nil
	})

	done := make(chan struct{})
	p.Then(
		func(value js.Value) {
			if diff := cmp.Diff(value.String(), "done!"); diff != "" {
				t.Errorf("incorrect result: -got +want: %s", diff)
			}
			close(done)
		},
		func(err error) {
			t.Errorf("Reject invoked with error: %v", err)
			close(done)
		},
	)
	<-done
}
