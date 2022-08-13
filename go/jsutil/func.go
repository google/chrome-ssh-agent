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

// AddEventListener adds a function that will be invoked on the specified event
// for an object.  The returned cleanup function must be invoked to cleanup the
// function.
func AddEventListener(o js.Value, event string, f func(this js.Value, args []js.Value) interface{}) CleanupFunc {
	fo := js.FuncOf(f)
	o.Call("addEventListener", event, fo)
	return func() {
		o.Call("removeEventListener", event, fo)
		fo.Release()
	}
}