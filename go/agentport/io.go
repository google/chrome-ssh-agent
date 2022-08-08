//go:build js && wasm

// The MIT License
//
// Copyright (c) 2015- Stripe, Inc. (https://stripe.com)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Package agentport supports serving the SSH Agent protocol to Chrome's
// Secure Shell Extension.
package agentport

import (
	"encoding/binary"
	"io"
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/dom"
	"github.com/norunners/vert"
)

type AgentPort struct {
	p         js.Value
	inReader  *io.PipeReader // client -> agent pipe: agent read from incoming messages
	inWriter  *io.PipeWriter // client -> agent pipe: write to agent
	outReader *io.PipeReader // agent -> client pipe: read from agent
	outWriter *io.PipeWriter // agent -> client pipe: agent write to outgoing messages
}

// New returns a io.ReaderWriter that converts from the Chrome Secure Shell
// Extension's SSH Agent protocol to the standard SSH Agent protocol.
//
// p is a Chrome Port object to which the Chrome Secure Shell Extension
// has connected.
func New(p js.Value) *AgentPort {
	dom.LogDebug("AgentPort.New")
	ir, iw := io.Pipe()
	or, ow := io.Pipe()
	ap := &AgentPort{
		p:         p,
		inReader:  ir,
		inWriter:  iw,
		outReader: or,
		outWriter: ow,
	}

	dom.LogDebug("AgentPort.New: Initiating SendMessages loop")
	go ap.SendMessages()

	return ap
}

func (ap *AgentPort) OnDisconnect() {
	dom.LogDebug("AgentPort.OnDisconnect: closing input writer")
	ap.inWriter.Close()
	dom.LogDebug("AgentPort.OnDisconnect: closing output writer")
	ap.outWriter.Close()
}

type message struct {
	Data []int `js:"data"`
}

func (ap *AgentPort) OnMessage(msg js.Value) {
	dom.LogDebug("AgentPort.OnMessage: parsing message from client to agent")
	var parsed message
	if err := vert.ValueOf(msg).AssignTo(&parsed); err != nil {
		dom.LogError("Failed to parse message to agent: %v; message=%s", err, msg)
		ap.p.Call("disconnect")
		return
	}

	dom.LogDebug("AgentPort.OnMessage: converting to bytestream")
	framed := make([]byte, 4+len(parsed.Data))
	binary.BigEndian.PutUint32(framed, uint32(len(parsed.Data)))
	for i, raw := range parsed.Data {
		framed[i+4] = byte(raw)
	}

	dom.LogDebug("AgentPort.OnMessage: writing to agent")
	_, err := ap.inWriter.Write(framed)
	if err != nil {
		dom.LogError("Error writing to pipe: %v", err)
		ap.p.Call("disconnect")
	}
}

func (ap *AgentPort) Read(p []byte) (n int, err error) {
	dom.LogDebug("AgentPort.Read: agent reading from client")
	defer dom.LogDebug("AgentPort.Read: read finished")
	return ap.inReader.Read(p)
}

var (
	array = js.Global().Get("Array")
)

func (ap *AgentPort) SendMessages() {
	dom.LogDebug("AgentPort.SendMessages: starting loop")
	defer dom.LogDebug("AgentPort.SendMessages: finished loop")
	for {
		dom.LogDebug("AgentPort.SendMessages: reading message length from agent to client")
		l := make([]byte, 4)
		_, err := io.ReadFull(ap.outReader, l)
		if err != nil {
			dom.Log("AgentPort.SendMessages: Error reading from pipe: %v", err)
			ap.outReader.Close()
			return
		}
		length := binary.BigEndian.Uint32(l)

		dom.LogDebug("AgentPort.SendMessages: reading message from agent to client")
		data := make([]byte, length)
		_, err = io.ReadFull(ap.outReader, data)
		if err != nil {
			dom.Log("AgentPort.SendMessages: Error reading from pipe: %v", err)
			ap.outReader.Close()
			return
		}

		dom.LogDebug("AgentPort.SendMessages: encoding message from agent to client")
		var encoded message
		encoded.Data = make([]int, len(data))
		for i, b := range data {
			encoded.Data[i] = int(b)
		}

		dom.LogDebug("AgentPort.SendMessages: sending message to client")
		ap.p.Call("postMessage", vert.ValueOf(encoded).JSValue())
	}
}

func (ap *AgentPort) Write(p []byte) (n int, err error) {
	dom.LogDebug("AgentPort.Write: agent writing to client")
	defer dom.LogDebug("AgentPort.Write: write finished")
	return ap.outWriter.Write(p)
}

type portRef struct {
	p js.Value
}

// AgentPorts is a mapping of chrome.runtime.Port objects to the corresponding
// connection (AgentPort) with our Agent.
type AgentPorts map[*portRef]*AgentPort

// Lookup returns the AgentPort corresponding to the supplied Port value. A Port
// value is considered equal if it refers to the exact same Port as was
// originally supplied. This works because the Chrome runtime appears to
// maintain a unique Port value for each port, and just pass around a reference
// to it.  Thus, we use js.Value.Equal() to compare ports; two references to the
// same object are equal iff they are equal in the '===' sense in Javascript.
func (a AgentPorts) Lookup(port js.Value) *AgentPort {
	for p, ap := range a {
		if p.p.Equal(port) {
			return ap
		}
	}
	return nil
}

// Delete removes the AgentPort corresponding to the supplied Port.
func (a AgentPorts) Delete(port js.Value) {
	for p := range a {
		if p.p.Equal(port) {
			delete(a, p)
			return
		}
	}
}

// Add adds an AgentPort corresponding to the supplied Port.
func (a AgentPorts) Add(port js.Value, ap *AgentPort) {
	a[&portRef{p: port}] = ap
}
