//go:build js

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

	"github.com/google/chrome-ssh-agent/go/app"
	"github.com/google/chrome-ssh-agent/go/dom"
	"github.com/google/chrome-ssh-agent/go/jsutil"
	"github.com/google/chrome-ssh-agent/go/keys"
	"github.com/google/chrome-ssh-agent/go/message"
	"github.com/google/chrome-ssh-agent/go/optionsui"
	"github.com/google/chrome-ssh-agent/go/testing"
)

type options struct {
	manager keys.Manager
	doc     *dom.Doc
}

func newOptions() *options {
	mgr := keys.NewClient(message.NewLocalSender())
	doc := dom.New(js.Null())

	return &options{
		manager: mgr,
		doc:     doc,
	}
}

func (a *options) Name() string {
	return "OptionsUI"
}

func (a *options) Init(ctx jsutil.AsyncContext, cleanup *jsutil.CleanupFuncs) error {
	ui := optionsui.New(a.manager, a.doc)
	cleanup.Add(ui.Release)

	qs := dom.NewURLSearchParams(dom.DefaultQueryString())
	if qs.Has("test") {
		testing.WriteResults(a.doc, ui.EndToEndTest(ctx))
	}

	return nil
}

func main() {
	a := app.New(newOptions())
	defer a.Release()
	a.Run()
}
