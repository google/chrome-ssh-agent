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

package jsutil

import (
	"fmt"
	"syscall/js"
)

var (
	promise = js.Global().Get("Promise")
)

// Promise encapsulates a javascript Promise type.
type Promise struct {
	v js.Value
}

// NewPromise runs a function in the background.
func NewPromise(f func(resolve func(value js.Value), reject func(reason js.Value))) *Promise {
	v := promise.New(OneTimeFuncOf(func(this js.Value, args []js.Value) interface{} {
		// Gather the resolve and reject functions to be invoked.
		var doResolve, doReject js.Value
		ExpandArgs(args, &doResolve, &doReject)

		// Define stub functions to invoke resolve and reject.
		invokeResolve := func(value js.Value) {
			doResolve.Invoke(value)
		}
		invokeReject := func(reason js.Value) {
			doReject.Invoke(reason)
		}

		// Run function in the background. The function invokes resolve
		// or reject as appropriate, which forwards them on to the
		// appropriate functions.
		go func() {
			f(invokeResolve, invokeReject)
		}()

		return nil
	}))
	return &Promise{v: v}
}

// AsPromise returns a Promise for javascript value. The value is assumed to
// be of the Promise type.
func AsPromise(v js.Value) *Promise {
	if v.Type() != js.TypeObject || v.Get("then").Type() != js.TypeFunction {
		panic(fmt.Errorf("Value %s is not a Promise", v))
	}

	return &Promise{v: v}
}

// JSValue returns the javascript value representing the promise.
func (p *Promise) JSValue() js.Value {
	return p.v
}

// Then implements Promise.then(). Resolve is invoked when the promise is
// resolved, and reject is invoked when it is rejected.
func (p *Promise) Then(resolve func(value js.Value), reject func(reason js.Value)) {
	var onReject, onResolve js.Func
	onResolve = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Only one of resolve or reject will be called, but both must
		// be released.
		defer onResolve.Release()
		defer onReject.Release()
		resolve(SingleArg(args))
		return nil
	})
	onReject = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer onResolve.Release()
		defer onReject.Release()
		reject(SingleArg(args))
		return nil
	})
	p.v.Call("then", onResolve, onReject)
}
