//go:build js && wasm

// Copyright 2017 Google LLC
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

// Package chrome exposes Go versions of Chrome's extension APIs.
package chrome

import (
	"errors"
	"syscall/js"
)

var (
	// chrome is a reference to the 'chrome' object.
	chrome = js.Global().Get("chrome")
	// runtime is a reference to the'chrome.runtime' object.
	runtime = func() js.Value {
		if chrome.IsUndefined() {
			return js.Undefined() // Not running under chrome.
		}
		return chrome.Get("runtime")
	}()
	// extensionID is the unique ID allocated to our extension.
	extensionID = func() string {
		if runtime.IsUndefined() {
			return "" // Not running under chrome.
		}
		return runtime.Get("id").String()
	}()
)

// Runtime returns a reference to 'chrome.runtime'.
func Runtime() js.Value {
	return runtime
}

// ExtensionID returns the unique ID allocated to this extension.
func ExtensionID() string {
	return extensionID
}

// LastError returns the error (if any) from the last call. Returns nil if there
// was no error.
//
// See https://developer.chrome.com/apps/runtime#property-lastError.
func LastError() error {
	if err := runtime.Get("lastError"); !err.IsNull() && !err.IsUndefined() {
		return errors.New(err.Get("message").String())
	}
	return nil
}
