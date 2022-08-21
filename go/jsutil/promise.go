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

// ResolveFunc is a function type that can be used to resolve a promise.
type ResolveFunc func(value js.Value)

// RejectFunc is a function type that can be used to reject a promise.
type RejectFunc func(err error)

// AsyncContext is a type of context passed to a function executing
// asynchronously.
type AsyncContext interface{}

// NewPromise runs a function in the background.  The function must
// invoke resolve or reject when complete. The function is passed an
// AsyncContext which can then be used to invoke Await() on promises
// that it subsequently creates.
func NewPromise(f func(ctx AsyncContext, resolve ResolveFunc, reject RejectFunc)) *Promise {
	v := promise.New(OneTimeFuncOf(func(this js.Value, args []js.Value) interface{} {
		// Gather the resolve and reject functions to be invoked.
		var doResolve, doReject js.Value
		ExpandArgs(args, &doResolve, &doReject)

		// Define stub functions to invoke resolve and reject.
		invokeResolve := func(value js.Value) {
			doResolve.Invoke(value)
		}
		invokeReject := func(reason error) {
			doReject.Invoke(NewError(reason).AsJSValue())
		}

		// Run function in the background. The function invokes resolve
		// or reject as appropriate, which forwards them on to the
		// appropriate functions.
		go func() {
			f(asyncContext, invokeResolve, invokeReject)
		}()

		return nil
	}))
	return &Promise{v: v}
}

// AsPromise returns a Promise for javascript value. The value is assumed to
// be of the Promise type.
func AsPromise(v js.Value) *Promise {
	if !v.InstanceOf(promise) {
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
func (p *Promise) Then(resolve ResolveFunc, reject RejectFunc) {
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
		reject(NewErrorFromVal(SingleArg(args)))
		return nil
	})
	p.v.Call("then", onResolve, onReject)
}

var (
	// asyncContext is a token value that implements AsyncContext. This
	// token value is expected to supplied to all asynchronously executing
	// functions. This allows us to require blocking calls such as Await()
	// to supply the context, providing some safety that the caller was
	// actually invoking them from an asynchronously executing function. If
	// blocking calls were made from the the main thread, we would deadlock.
	asyncContext = &struct{}{}
)

// Async executes a function asynchronously.  A promise corresponding to the
// function is returned.
func Async(f func(ctx AsyncContext) (js.Value, error)) *Promise {
	p := NewPromise(func(ctx AsyncContext, resolve ResolveFunc, reject RejectFunc) {
		val, err := f(ctx)
		if err != nil {
			reject(err)
		} else {
			resolve(val)
		}
	})
	return p
}

// Await blocks until a Promise is either resolved or rejected. It must only be
// invoked from within an AsyncContext.
func (p *Promise) Await(ctx AsyncContext) (js.Value, error) {
	if ctx != asyncContext {
		panic("Invalid AsyncContext")
	}

	var v js.Value
	var e error
	done := make(chan struct{})
	resolve := func(value js.Value) {
		v = value
		close(done)
	}
	reject := func(err error) {
		e = err
		close(done)
	}
	p.Then(resolve, reject)
	<-done
	return v, e
}
