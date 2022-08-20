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
	"syscall/js"
)

var (
	// json refers to Javascript's JSON class.
	json = js.Global().Get("JSON")
)

// ToJSON converts the supplied value to a JSON string.
func ToJSON(val js.Value) string {
	return json.Call("stringify", val).String()
}

// FromJSON converts the supplied JSON string to a Javascript value.
func FromJSON(s string) js.Value {
	defer func() {
		if r := recover(); r != nil {
			LogError("Failed to parse JSON string; returning default. Error: %v", r)
		}
	}()

	return json.Call("parse", s)
}
