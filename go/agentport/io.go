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

package agentport

import (
	"encoding/binary"
	"errors"
	"io"
	"log"

	"github.com/gopherjs/gopherjs/js"
)

var ErrInvalidMsg = errors.New("invalid message frame")

type agentPort struct {
	p         *js.Object
	inReader  *io.PipeReader
	inWriter  *io.PipeWriter
	outReader *io.PipeReader
	outWriter *io.PipeWriter
}

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
