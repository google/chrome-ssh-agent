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

package lock

import (
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/jsutil"
)

var (
	locks = func() js.Value {
		// Prefer Web Locks API is defined under 'navigator.locks'
		if navigator := js.Global().Get("navigator"); !navigator.IsUndefined() {
			return navigator.Get("locks")
		}
		// Fallback to node.js's web-locks implementation (used in tests).
		return js.Global().Call("eval", `{
			const { locks } = require("web-locks");
			locks;
		}`)
	}()
)

// Async runs a routine asynchronously once access to the resource has been
// granted. Access to the resource is released when the routine returns.
func Async(resource string, f func(ctx jsutil.AsyncContext)) *jsutil.Promise {
	return jsutil.AsPromise(locks.Call(
		"request",
		// Return a promise encapsulating the supplied function.
		jsutil.OneTimeFuncOf(func(this js.Value, args []js.Value) interface{} {
			return jsutil.Async(func(ctx jsutil.AsyncContext) (js.Value, error) {
				f(ctx)
				return js.Undefined(), nil
			}).JSValue()
		}),
	))
}
