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

	"github.com/google/chrome-ssh-agent/go/agentport"
	"github.com/google/chrome-ssh-agent/go/chrome"
	"github.com/google/chrome-ssh-agent/go/dom"
	"github.com/google/chrome-ssh-agent/go/keys"
	"golang.org/x/crypto/ssh/agent"
)

var (
	// Create a keyring with loaded keys.
	agt = agent.NewKeyring()

	// Create a wrapper that can update the loaded keys. Exposed the
	// wrapper so it can be used by other pages in the extension.
	chr = chrome.New(js.Null())
	mgr = keys.NewManager(agt, chr.SyncStorage(), chr.SessionStorage())
	svr = keys.NewServer(mgr)

	// Reload any keys for the session into the agent.
	_ = func() bool {
		mgr.LoadFromSession(func(err error) {
			if err != nil {
				dom.LogError("failed to load keys into agent: %v", err)
			}
		})
		return true // Dummy value.
	}()
)

func onMessage(this js.Value, args []js.Value) interface{} {
	var message, sender, sendResponse js.Value
	dom.ExpandArgs(args, &message, &sender, &sendResponse)
	svr.OnMessage(message, sender, func(rsp js.Value) {
		sendResponse.Invoke(rsp)
	})
	return nil
}

func onConnectExternal(this js.Value, args []js.Value) interface{} {
	port := dom.SingleArg(args)
	dom.Log("Starting agent for new port")
	go agent.ServeAgent(agt, agentport.New(port))
	return nil
}

func main() {
	dom.Log("Starting background worker")
	defer dom.Log("Exiting background worker")

	js.Global().Set("handleOnMessage", js.FuncOf(onMessage))
	js.Global().Set("handleOnConnectExternal", js.FuncOf(onConnectExternal))

	done := make(chan struct{}, 0)
	<-done // Do not terminate.
}
