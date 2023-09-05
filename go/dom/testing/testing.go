//go:build js

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
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/jsutil"
)

// We use jsdom to create a new Document object. For use in testing
// only. This requires running under node.js with the jsdom package
// installed.
var jsdom = js.Global().Call("eval", `{
	const jsdom = require("jsdom");
	const { JSDOM } = jsdom;
	JSDOM;
}`)

// NewDocForTesting returns a Document object that can be used for testing.
// The DOM in the Document object is instantiated using the supplied HTML.
func NewDocForTesting(html string) js.Value {
	dom := jsdom.New(html, map[string]interface{}{
		"runScripts": "dangerously",
		"resources":  "usable",
	})

	// Create doc, but then wait until loading is complete and constructed
	// doc is returned. By default, jsdom loads doc asynchronously:
	//   https://oliverjam.es/blog/frontend-testing-node-jsdom/#waiting-for-external-resources
	c := make(chan js.Value)
	dom.Get("window").Call(
		"addEventListener", "load",
		jsutil.OneTimeFuncOf(func(this js.Value, args []js.Value) interface{} {
			c <- dom.Get("window").Get("document")
			return nil
		}))
	return <-c
}
