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
	"fmt"
	"syscall/js"
)

var (
	// object refers to Javascript's Object class.
	object = js.Global().Get("Object")
)

func NewObject() js.Value {
	return object.New()
}

// ObjectKeys returns the keys for a given object.
func ObjectKeys(val js.Value) ([]string, error) {
	if val.Type() != js.TypeObject {
		return nil, fmt.Errorf("Object required; got type %s", val.Type())
	}

	var res []string
	keys := object.Call("keys", val)
	for i := 0; i < keys.Length(); i++ {
		res = append(res, keys.Index(i).String())
	}
	return res, nil
}
