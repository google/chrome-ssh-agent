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

// RepeatableFunc is a special type of function that is expected to be
// repeatedly invoked.  Thus, invoking Release() is the caller's responsibility.
type RepeatableFunc js.Func


// RepeatableFuncOf returns a RepeatablFunc that can be invoked multiple times.
// The caller is responsible for invoking Release() on the resulting func when
// it is no longer needed.
func RepeatableFuncOf(f func(this js.Value, args []js.Value) interface{}) RepeatableFunc {
	return RepeatableFunc(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return f(this, args)
	}))
}

// AsJSFunc returns the corresponding js.Func object.
func (r RepeatableFunc) AsJSFunc() js.Func {
	return js.Func(r)
}
