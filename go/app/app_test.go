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
	"errors"
	"syscall/js"
	"testing"
	"time"

	"github.com/google/chrome-ssh-agent/go/jsutil"
	jut "github.com/google/chrome-ssh-agent/go/jsutil/testing"
)

func resolveValue(name string) js.Value {
	for {
		if v := js.Global().Get(name); !v.IsUndefined() {
			return v
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func initWait(ctx jsutil.AsyncContext) (js.Value, error) {
	f := resolveValue(initWaitFunc)
	return jsutil.AsPromise(f.Invoke()).Await(ctx)
}

func terminate(ctx jsutil.AsyncContext) (js.Value, error) {
	f := resolveValue(terminateFunc)
	return jsutil.AsPromise(f.Invoke()).Await(ctx)
}

type testApp struct {
	initErr error
	initted bool
}

func (t *testApp) Name() string { return "TestApp" }

func (t *testApp) Init(_ jsutil.AsyncContext, _ *jsutil.CleanupFuncs) error {
	t.initted = true
	return t.initErr
}

func TestAppInitAndTerminate(t *testing.T) {
	t.Parallel()

	func() {
		a := &testApp{}
		ac := New(a)
		defer ac.Release()

		t.Log("Start app")
		done := make(chan struct{})
		go func() {
			defer close(done)
			ac.Run()
		}()

		jut.DoSync(func(ctx jsutil.AsyncContext) {
			t.Log("Ensure Init() was called")
			if _, err := initWait(ctx); err != nil {
				t.Errorf("Init() failed: %v", err)
			}
			if !a.initted {
				t.Errorf("Init() not invoked")
			}

			t.Log("Terminate app")
			if _, err := terminate(ctx); err != nil {
				t.Errorf("Terminate() failed: %v", err)
			}
		})

		t.Log("Wait for app terminate")
		<-done
	}()

	t.Log("Validate function cleanup")
	if f := js.Global().Get(initWaitFunc); !f.IsUndefined() {
		t.Errorf("initWait function not cleaned up")
	}
	if f := js.Global().Get(terminateFunc); !f.IsUndefined() {
		t.Errorf("terminate function not cleaned up")
	}
}

func TestAppInitErr(t *testing.T) {
	t.Parallel()

	a := &testApp{
		initErr: errors.New("init failed"),
	}
	ac := New(a)
	defer ac.Release()

	t.Log("Start app")
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected Run() to panic on Init() error")
			}
		}()
		ac.Run()
	}()

	jut.DoSync(func(ctx jsutil.AsyncContext) {
		t.Log("Ensure Init() returned error")
		if _, err := initWait(ctx); err == nil {
			t.Errorf("Init() did not return error")
		}
		if !a.initted {
			t.Errorf("Init() not invoked")
		}
	})

	t.Log("Wait for app terminate")
	<-done
}
