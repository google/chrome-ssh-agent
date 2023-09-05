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

package app

import (
	"fmt"
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/jsutil"
)

const (
	initWaitFunc  = "appInitWaitImpl"
	terminateFunc = "appTerminateImpl"
)

// App defines a WASM application that can be managed with the Run routine
// (see below).
type App interface {
	// Name returns a descriptive name for the application, suitable for
	// display in logs.
	Name() string

	// Init performs any initialization work needed for the application.
	Init(ctx jsutil.AsyncContext, cleanup *jsutil.CleanupFuncs) error
}

type Context struct {
	app     App
	cleanup jsutil.CleanupFuncs
}

func New(app App) *Context {
	return &Context{
		app: app,
	}
}

func (a *Context) Release() {
	a.cleanup.Do()
}

// Run runs the application.  Run is expected to be called directly from the main
// function of the program, and thus is permitted to block.
//
// Run exports the following async functions to be available from Javascript:
//
//	initWaitFunc (see above): waits for app initialization to complete. If this
//	  function returns successfully, then the App.Init() function is guaranteed
//	  to have completed without error.
//
//	terminateFunc (see above): signals the application to terminate; Run() will
//	  terminate.
func (a *Context) Run() {
	jsutil.LogDebug("%s starting", a.app.Name())
	defer jsutil.LogDebug("%s finished", a.app.Name())

	var initErr error
	init := newSignal()
	done := newSignal()

	var cleanup jsutil.CleanupFuncs
	defer cleanup.Do()

	a.cleanup.Add(jsutil.DefineAsyncFunc(js.Global(), initWaitFunc, func(ctx jsutil.AsyncContext, this js.Value, args []js.Value) (js.Value, error) {
		init.Wait()
		return js.Undefined(), initErr
	}))
	a.cleanup.Add(jsutil.DefineAsyncFunc(js.Global(), terminateFunc, func(ctx jsutil.AsyncContext, this js.Value, args []js.Value) (js.Value, error) {
		done.Signal()
		return js.Undefined(), nil
	}))

	jsutil.Async(func(ctx jsutil.AsyncContext) (js.Value, error) {
		jsutil.LogDebug("Run: Initialize")
		defer jsutil.LogDebug("Run: Finished Initialize")
		initErr = a.app.Init(ctx, &cleanup)
		init.Signal()
		return js.Undefined(), nil
	})

	init.Wait()
	if initErr != nil {
		panic(fmt.Errorf("%s init failed: %w; terminating", a.app.Name(), initErr))
	}

	done.Wait()
}
