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
	"errors"
	"syscall/js"
)

var jsError = js.Global().Get("Error")

// JSError represents javascript's Error type.
type JSError struct {
	js.Value
}

// AsJSValue returns the error as a Javascript Value.
func (e JSError) AsJSValue() js.Value {
	return e.Value
}

// Name returns the Error's name.
func (e JSError) Name() string {
	return e.Value.Get("name").String()
}

// Message returns the Error's message.
func (e JSError) Message() string {
	return e.Value.Get("message").String()
}

// Error implements Go's error interface.
func (e JSError) Error() string {
	return e.Value.Call("toString").String()
}

const (
	goErrorName = "GoError"
)

// NewError returns a Javascript error corresponding to a Go error.
func NewError(err error) JSError {
	if err == nil {
		panic("Cannot construct nil JSError")
	}

	var je JSError
	if errors.As(err, &je) {
		return je
	}

	e := jsError.New(err.Error())
	e.Set("name", goErrorName)
	return JSError{Value: e}
}

// NewErrorFromVal returns a Javascript error corresponding to an
// arbitrary value. We do our best to preserve the original value,
// but we do assume that it reflects an error and is not some
// arbitrary value.
func NewErrorFromVal(val js.Value) JSError {
	switch {
	case val.IsUndefined():
		panic("Cannot construct JSError from undefined")
	case val.IsNull():
		panic("Cannot construct JSError from null")
	case val.InstanceOf(jsError):
		return JSError{Value: val}
	default:
		return JSError{Value: jsError.New(val.String())}
	}
}
