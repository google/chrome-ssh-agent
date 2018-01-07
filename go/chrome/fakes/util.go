// Copyright 2018 Google LLC
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

package fakes

import (
	"github.com/gopherjs/gopherjs/js"
)

var (
	dummys = js.Global.Call("eval", `({
		return: function(x) {
			return x;
		},
	})`)
)

// toJSObject returns the js.Object corresponding to the supplied value.
//
// This is useful when implementing fake implementations that need to simulate
// how an object will be transformed by GopherJS to be consumed by external
// Javascript code (e.g., Chrome's extension APIs).
func toJSObject(v interface{}) *js.Object {
	return dummys.Call("return", v)
}
