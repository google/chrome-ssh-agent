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
	"github.com/google/chrome-ssh-agent/go/jsutil"
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

	// Keep a set of ports that are open for communicating between
	// clients and agents.
	ports = agentport.AgentPorts{}
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
	if ports.Lookup(port) != nil {
		dom.LogError("onConnectExternal: port already in use; ignoring")
		return nil
	}

	dom.LogDebug("onConnectExternal: connecting new port")
	ap := agentport.New(port)
	ports.Add(port, ap)

	dom.LogDebug("onConnectExternal: serving in background")
	go func() {
		dom.LogDebug("ServeAgent: starting for new port")
		defer dom.LogDebug("ServeAgent: finished")
		if err := agent.ServeAgent(agt, ap); err != nil {
			dom.LogDebug("ServeAgent: finished with error: %v", err)
		}
	}()
	return nil
}

func onConnectionMessage(this js.Value, args []js.Value) interface{} {
	var port, msg js.Value
	dom.ExpandArgs(args, &port, &msg)

	ap := ports.Lookup(port)
	if ap == nil {
		dom.LogError("onConnectionMessage: connection for port not found; ignoring")
		return nil
	}

	dom.LogDebug("onConnectionMessage: forwarding message")
	ap.OnMessage(msg)
	return nil
}

func onConnectionDisconnect(this js.Value, args []js.Value) interface{} {
	port := dom.SingleArg(args)

	ap := ports.Lookup(port)
	if ap == nil {
		dom.LogError("onConnectionDisconnect: connection for port not found; ignoring")
		return nil
	}

	dom.LogDebug("onConnectionDisconnect: disconnecting")
	ap.OnDisconnect()
	ports.Delete(port)
	return nil
}

func main() {
	dom.Log("Starting background worker")
	defer dom.Log("Exiting background worker")

	c1 := jsutil.DefineFunc(js.Global(), "handleOnMessage", onMessage)
	defer c1()
	c2 := jsutil.DefineFunc(js.Global(), "handleOnConnectExternal", onConnectExternal)
	defer c2()
	c3 := jsutil.DefineFunc(js.Global(), "handleConnectionMessage", onConnectionMessage)
	defer c3()
	c4 := jsutil.DefineFunc(js.Global(), "handleConnectionDisconnect", onConnectionDisconnect)
	defer c4()

	done := make(chan struct{}, 0)
	<-done // Do not terminate immediately.
}
