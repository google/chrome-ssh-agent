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

// Package testing provides utilities for using the DOM in unit tests.
package testing

import (
	"github.com/gopherjs/gopherjs/js"
)

var (
	// funcs contains Javascript functions that can be invoked by Go library
	// functions in this package. The following are defined:
	// - newDoc: Uses jsdom to create a new Document object. For use in
	//     testing only. This requires node.js (which is used by
	//     'gopherjs test') with the jsdom package installed.
	funcs = js.Global.Call("eval", `({
		newDoc: function(html) {
			const jsdom = require("jsdom");
			const virtualConsole = new jsdom.VirtualConsole();
			virtualConsole.sendTo(console);
			const { JSDOM } = jsdom;
			const dom = new JSDOM(html);
			return dom.window.document;
		},
	})`)
)

// NewDocForTesting returns a Document object that can be used for testing.
// The DOM in the Document object is instantiated using the supplied HTML.
func NewDocForTesting(html string) *js.Object {
	return funcs.Call("newDoc", html)
}
