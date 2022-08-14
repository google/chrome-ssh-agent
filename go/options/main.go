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

package main

import (
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/chrome"
	"github.com/google/chrome-ssh-agent/go/dom"
	"github.com/google/chrome-ssh-agent/go/jsutil"
	"github.com/google/chrome-ssh-agent/go/keys"
	"github.com/google/chrome-ssh-agent/go/optionsui"
	"github.com/google/chrome-ssh-agent/go/testing"
)

func main() {
	jsutil.Log("Starting Options UI")
	defer jsutil.Log("Exiting Options UI")
	done := make(chan struct{}, 0)

	c := chrome.New(js.Null())
	mgr := keys.NewClient(c)
	d := dom.New(dom.Doc)
	ui := optionsui.New(mgr, d)
	defer ui.Release()

	qs := dom.NewURLSearchParams(dom.DefaultQueryString())
	if qs.Has("test") {
		testing.WriteResults(d, ui.EndToEndTest())
	}

	<-done // Do not terminate.
}
