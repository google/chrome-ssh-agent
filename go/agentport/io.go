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
	"log"

	"github.com/gopherjs/gopherjs/js"
)

type agentPort struct {
	p         *js.Object
	inReader  *io.PipeReader
	inWriter  *io.PipeWriter
	outReader *io.PipeReader
	outWriter *io.PipeWriter
}

// New returns a io.ReaderWriter that converts from the Chrome Secure Shell
// Extension's SSH Agent protocol to the standard SSH Agent protocol.
//
// p is a Chrome Port object to which the Chrome Secure Shell Extension
// has connected.
func New(p *js.Object) io.ReadWriter {
	ir, iw := io.Pipe()
	or, ow := io.Pipe()
	ap := &agentPort{
		p:         p,
		inReader:  ir,
		inWriter:  iw,
		outReader: or,
		outWriter: ow,
	}
	ap.p.Get("onDisconnect").Call("addListener", func() {
		go ap.OnDisconnect()
	})
	ap.p.Get("onMessage").Call("addListener", func(msg js.M) {
		go ap.OnMessage(msg)
	})

	go ap.SendMessages()

	return ap
}

func (ap *agentPort) OnDisconnect() {
	ap.inWriter.Close()
}

func (ap *agentPort) OnMessage(msg js.M) {
	d, ok := msg["data"].([]interface{})
	if !ok {
		log.Printf("Message did not contain Array data field: %v", msg)
		ap.p.Call("disconnect")
		return
	}

	framed := make([]byte, 4+len(d))
	binary.BigEndian.PutUint32(framed, uint32(len(d)))

	for i, raw := range d {
		n, ok := raw.(float64)
		if !ok {
			log.Printf("Message contained non-numeric data: %v", msg)
			ap.p.Call("disconnect")
			return
		}

		framed[i+4] = byte(n)
	}

	_, err := ap.inWriter.Write(framed)
	if err != nil {
		log.Printf("Error writing to pipe: %v", err)
		ap.p.Call("disconnect")
	}
}

func (ap *agentPort) Read(p []byte) (n int, err error) {
	return ap.inReader.Read(p)
}

func (ap *agentPort) SendMessages() {
	for {
		l := make([]byte, 4)
		_, err := io.ReadFull(ap.outReader, l)
		if err != nil {
			log.Printf("Error reading from pipe: %v", err)
			ap.outReader.Close()
			return
		}
		length := binary.BigEndian.Uint32(l)

		data := make([]byte, length)
		_, err = io.ReadFull(ap.outReader, data)
		if err != nil {
			log.Printf("Error reading from pipe: %v", err)
			ap.outReader.Close()
			return
		}

		encoded := make(js.S, length)
		for i, b := range data {
			encoded[i] = float64(b)
		}

		ap.p.Call("postMessage", js.M{
			"data": encoded,
		})
	}
}

func (ap *agentPort) Write(p []byte) (n int, err error) {
	return ap.outWriter.Write(p)
}
