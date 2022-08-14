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
	"time"
)

// OneTimeFuncOf returns a js.Func that can be invoked once. It is automatically
// released when invoked.
func OneTimeFuncOf(f func(this js.Value, args []js.Value) interface{}) js.Func {
	var cb js.Func
	cb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer cb.Release()
		return f(this, args)
	})
	return cb
}

// CleanupFunc is a function to be invoked to cleanup resources.
type CleanupFunc func()

// CleanupFuncs is a set of cleanup functions.
type CleanupFuncs struct {
	cf []CleanupFunc
}

// Add adds a cleanup func.
func (c *CleanupFuncs) Add(f CleanupFunc) {
	c.cf = append(c.cf, f)
}

// Do invokes all of the cleanup functions.
func (c *CleanupFuncs) Do() {
	for _, f := range c.cf {
		f()
	}
}

// DefineFunc defines a new function and attaches it to the specified object.
// The returned cleanup function must be invoked to detach the function and
// release it.
func DefineFunc(o js.Value, name string, f func(this js.Value, args []js.Value) interface{}) CleanupFunc {
	fo := js.FuncOf(f)
	o.Set(name, fo)
	return func() {
		o.Set(name, js.Undefined())
		fo.Release()
	}
}

// SetTimeout registers a callback to be invoked when the timeout has expired.
func SetTimeout(timeout time.Duration, callback func()) {
	cb := OneTimeFuncOf(func(this js.Value, args []js.Value) interface{} {
		callback()
		return nil
	})
	js.Global().Call("setTimeout", cb, timeout.Milliseconds())
}

// ExpandArgs unpacks function arguments to target values.
func ExpandArgs(args []js.Value, target ...*js.Value) {
	// Assign args to target.
	for i := 0; i < len(args) && i < len(target); i++ {
		*(target[i]) = args[i]
	}
	// Any excessive targets are set to undefined.
	for i := len(args); i < len(target); i++ {
		*(target[i]) = js.Undefined()
	}
}

// SingleArg unpacks a single function argument and returns it.
func SingleArg(args []js.Value) js.Value {
	var val js.Value
	ExpandArgs(args, &val)
	return val
}
