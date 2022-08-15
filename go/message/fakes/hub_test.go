//go:build js && wasm

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

	"github.com/google/go-cmp/cmp"
	"github.com/norunners/vert"
)

type intReceiver struct{}

func (i *intReceiver) OnMessage(header js.Value, sender js.Value, sendResponse func(js.Value)) {
	if header.Type() == js.TypeNumber && header.Int() == 42 {
		sendResponse(js.ValueOf("int"))
	}
}

type stringReceiver struct{}

func (s *stringReceiver) OnMessage(header js.Value, sender js.Value, sendResponse func(js.Value)) {
	if header.Type() == js.TypeString && header.String() == "foo" {
		sendResponse(js.ValueOf("string"))
	}
}

type mapReceiver struct{}

func (m *mapReceiver) OnMessage(header js.Value, sender js.Value, sendResponse func(js.Value)) {
	if header.Type() == js.TypeObject && !header.Get("some-key").IsUndefined() {
		sendResponse(js.ValueOf("map"))
	}
}

func TestMessagePassing(t *testing.T) {
	hub := NewHub()

	// Add handlers that respond to different values.
	hub.AddReceiver(&intReceiver{})
	hub.AddReceiver(&stringReceiver{})
	hub.AddReceiver(&mapReceiver{})

	// Send messages of the various types.
	var intRsp, strRsp, mapRsp js.Value
	hub.Send(js.ValueOf(42), func(rsp js.Value, err error) {
		if err != nil {
			t.Errorf("SendMessage failed: %v", err)
			return
		}
		intRsp = rsp
	})
	hub.Send(js.ValueOf("foo"), func(rsp js.Value, err error) {
		if err != nil {
			t.Errorf("SendMessage failed: %v", err)
			return
		}
		strRsp = rsp
	})
	hub.Send(vert.ValueOf(map[string]int{"some-key": 7}).JSValue(), func(rsp js.Value, err error) {
		if err != nil {
			t.Errorf("SendMessage failed: %v", err)
			return
		}
		mapRsp = rsp
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
