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
	"errors"
	"fmt"
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/agentport"
	"github.com/google/chrome-ssh-agent/go/jsutil"
	"github.com/google/chrome-ssh-agent/go/keys"
	"github.com/google/chrome-ssh-agent/go/storage"
	"golang.org/x/crypto/ssh/agent"
)

var (
	// Create a keyring with loaded keys.
	agt = agent.NewKeyring()

	// Create a wrapper that can update the loaded keys. Exposed the
	// wrapper so it can be used by other pages in the extension.
	mgr = keys.NewManager(agt, storage.DefaultSync(), storage.DefaultSession())
	svr = keys.NewServer(mgr)

	// Keep a set of ports that are open for communicating between
	// clients and agents.
	ports = agentport.AgentPorts{}
)

func onMessage(ctx jsutil.AsyncContext, this js.Value, args []js.Value) (js.Value, error) {
	var message, sender, sendResponse js.Value
	jsutil.ExpandArgs(args, &message, &sender, &sendResponse)
	rsp := svr.OnMessage(ctx, message, sender)
	sendResponse.Invoke(rsp)
	return js.Undefined(), nil
}

func onConnectExternal(ctx jsutil.AsyncContext, this js.Value, args []js.Value) (js.Value, error) {
	port := jsutil.SingleArg(args)
	if ports.Lookup(port) != nil {
		err := errors.New("onConnectExternal: port already in use")
		jsutil.LogError(err.Error())
		return js.Undefined(), err
	}

	jsutil.LogDebug("onConnectExternal: connecting new port")
	ap := agentport.New(port)
	ports.Add(port, ap)

	jsutil.LogDebug("onConnectExternal: serving in background")
	go func() {
		jsutil.LogDebug("ServeAgent: starting for new port")
		defer jsutil.LogDebug("ServeAgent: finished")
		if err := agent.ServeAgent(agt, ap); err != nil {
			jsutil.LogDebug("ServeAgent: finished with error: %v", err)
		}
	}()
	return js.Undefined(), nil
}

func onConnectionMessage(ctx jsutil.AsyncContext, this js.Value, args []js.Value) (js.Value, error) {
	var port, msg js.Value
	jsutil.ExpandArgs(args, &port, &msg)

	ap := ports.Lookup(port)
	if ap == nil {
		err := errors.New("onConnectionMessage: connection for port not found")
		jsutil.LogError(err.Error())
		return js.Undefined(), err
	}

	jsutil.LogDebug("onConnectionMessage: forwarding message")
	ap.OnMessage(msg)
	return js.Undefined(), nil
}

func onConnectionDisconnect(ctx jsutil.AsyncContext, this js.Value, args []js.Value) (js.Value, error) {
	port := jsutil.SingleArg(args)

	ap := ports.Lookup(port)
	if ap == nil {
		err := errors.New("onConnectionDisconnect: connection for port not found")
		jsutil.LogError(err.Error())
		return js.Undefined(), err
	}

	jsutil.LogDebug("onConnectionDisconnect: disconnecting")
	ap.OnDisconnect()
	ports.Delete(port)
	return js.Undefined(), nil
}

func main() {
	jsutil.Log("Starting background worker")
	defer jsutil.Log("Exiting background worker")

	var cleanup jsutil.CleanupFuncs
	defer cleanup.Do()

	// Reload any keys for the session into the agent.
	jsutil.Async(func(ctx jsutil.AsyncContext) (js.Value, error) {
		if err := mgr.LoadFromSession(ctx); err != nil {
			// Log error
			jsutil.LogError("failed to load keys into agent: %v", err)
		}
		return js.Undefined(), nil
	}).Then(
		// Upon completion, attach our event handlers.
		func(value js.Value) {
			cleanup.Add(jsutil.DefineAsyncFunc(js.Global(), "handleOnMessage", onMessage))
			cleanup.Add(jsutil.DefineAsyncFunc(js.Global(), "handleOnConnectExternal", onConnectExternal))
			cleanup.Add(jsutil.DefineAsyncFunc(js.Global(), "handleConnectionMessage", onConnectionMessage))
			cleanup.Add(jsutil.DefineAsyncFunc(js.Global(), "handleConnectionDisconnect", onConnectionDisconnect))
		},
		func(err error) { panic(fmt.Errorf("unexpected error: %v", err)) },
	)

	done := make(chan struct{}, 0)
	<-done // Do not terminate immediately.
}
