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

package main

import (
	"fmt"

	"github.com/google/chrome-ssh-agent/go/chrome"
	"github.com/google/chrome-ssh-agent/go/dom"
	"github.com/google/chrome-ssh-agent/go/keys"
	"github.com/google/chrome-ssh-agent/go/optionsui"
	"github.com/gopherjs/gopherjs/js"
)

func isTest() bool {
	win := js.Global.Get("window").Get("location").Get("search")
	params := js.Global.Get("URLSearchParams").New(win)
	return params.Call("has", "test").Bool()
}

func writeTestResults(d *dom.DOM, errs []error) {
	body := d.GetElement("body")
	// Clear the existing elements from the doc.
	d.RemoveChildren(body)
	// Indicate how many tests failed. Give the element an ID so
	// it can be read by automation.
	d.AppendChild(body, d.NewElement("div"), func(failureCount *js.Object) {
		failureCount.Set("id", "failureCount")
		d.AppendChild(failureCount, d.NewText(fmt.Sprintf("%d", len(errs))), nil)
	})
	// Enumerate the failures.
	for _, err := range errs {
		d.AppendChild(body, d.NewElement("div"), func(failure *js.Object) {
			d.AppendChild(failure, d.NewText(err.Error()), nil)
		})
	}
}

func main() {
	c := chrome.New(nil)
	mgr := keys.NewClient(c)
	d := dom.New(dom.Doc)
	ui := optionsui.New(mgr, d)

	if isTest() {
		writeTestResults(d, ui.EndToEndTest())
	}
}
