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
	"syscall/js"
	"testing"

	"github.com/google/chrome-ssh-agent/go/jsutil"
	jut "github.com/google/chrome-ssh-agent/go/jsutil/testing"
	"github.com/google/go-cmp/cmp"
	"github.com/norunners/vert"
)

type intReceiver struct{}

func (i *intReceiver) OnMessage(_ jsutil.AsyncContext, header js.Value, _ js.Value) js.Value {
	if header.Type() == js.TypeNumber && header.Int() == 42 {
		return js.ValueOf("int")
	}
	return js.Undefined()
}

type stringReceiver struct{}

func (s *stringReceiver) OnMessage(_ jsutil.AsyncContext, header js.Value, _ js.Value) js.Value {
	if header.Type() == js.TypeString && header.String() == "foo" {
		return js.ValueOf("string")
	}
	return js.Undefined()
}

type mapReceiver struct{}

func (m *mapReceiver) OnMessage(_ jsutil.AsyncContext, header js.Value, _ js.Value) js.Value {
	if header.Type() == js.TypeObject && !header.Get("some-key").IsUndefined() {
		return js.ValueOf("map")
	}
	return js.Undefined()
}

func TestMessagePassing(t *testing.T) {
	t.Parallel()

	hub := NewHub()

	// Add handlers that respond to different values.
	hub.AddReceiver(&intReceiver{})
	hub.AddReceiver(&stringReceiver{})
	hub.AddReceiver(&mapReceiver{})

	// Send messages of the various types.
	var err error
	var intRsp, strRsp, mapRsp js.Value
	jut.DoSync(func(ctx jsutil.AsyncContext) {
		if intRsp, err = hub.Send(ctx, js.ValueOf(42)); err != nil {
			t.Errorf("SendMessage failed: %v", err)
			return
		}
		if strRsp, err = hub.Send(ctx, js.ValueOf("foo")); err != nil {
			t.Errorf("SendMessage failed: %v", err)
			return
		}
		if mapRsp, err = hub.Send(ctx, vert.ValueOf(map[string]int{"some-key": 7}).JSValue()); err != nil {
			t.Errorf("SendMessage failed: %v", err)
			return
		}
	})

	// Ensure we got the correct responses.
	if diff := cmp.Diff(intRsp.String(), "int"); diff != "" {
		t.Errorf("incorrect response for int; -got +want: %s", diff)
	}
	if diff := cmp.Diff(strRsp.String(), "string"); diff != "" {
		t.Errorf("incorrect response for string; -got +want: %s", diff)
	}
	if diff := cmp.Diff(mapRsp.String(), "map"); diff != "" {
		t.Errorf("incorrect response for map; -got +want: %s", diff)
	}
}
