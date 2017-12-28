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

package keys

import (
	"errors"
	"fmt"

	"github.com/gopherjs/gopherjs/js"
	"github.com/google/chrome-ssh-agent/go/chrome"
)

type Server struct {
	a Available
}

func NewServer(avail Available) *Server {
	result := &Server{
		a: avail,
	}
	chrome.Runtime.Get("onMessage").Call("addListener", result.onMessage)
	return result
}

const (
	msgTypeAvailable int = 1000 + iota
	msgTypeAvailableRsp
	msgTypeLoaded
	msgTypeLoadedRsp
	msgTypeAdd
	msgTypeAddRsp
	msgTypeRemove
	msgTypeRemoveRsp
	msgTypeLoad
	msgTypeLoadRsp
)

type msgHeader struct {
	*js.Object
	Type int `js:"type"`
}

type msgAvailable struct {
	*msgHeader
}

type rspAvailable struct {
	*msgHeader
	Keys []string `js:"keys"`
	Err  string   `js:"err"`
}

type msgLoaded struct {
	*msgHeader
}

type rspLoaded struct {
	*msgHeader
	Keys []string `js:"keys"`
	Err  string   `js:"err"`
}

type msgAdd struct {
	*msgHeader
	Name          string `js:"name"`
	PEMPrivateKey string `js:"pemPrivateKey"`
}

type rspAdd struct {
	*msgHeader
	Err string `js:"err"`
}

type msgRemove struct {
	*msgHeader
	Name string `js:"name"`
}

type rspRemove struct {
	*msgHeader
	Err string `js:"err"`
}

type msgLoad struct {
	*msgHeader
	Name       string `js:"name"`
	Passphrase string `js:"passphrase"`
}

type rspLoad struct {
	*msgHeader
	Err string `js:"err"`
}

func makeErr(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}

func makeErrStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func makeObject(i interface{}) *js.Object {
	if i == nil {
		return js.Undefined
	}

	switch i.(type) {
	case *js.Object:
		return i.(*js.Object)
	case map[string]interface{}:
		o := js.Global.Get("Object").New()
		for k, v := range i.(map[string]interface{}) {
			o.Set(k, v)
		}
		return o
	}
	panic(fmt.Sprintf("failed to read object of type %T", i))
}

func (s *Server) onMessage(message interface{}, sender *js.Object, sendResponse func(interface{})) bool {
	omsg := makeObject(message)
	msg := &msgHeader{Object: omsg}
	switch msg.Type {
	case msgTypeAvailable:
		s.a.Available(func(keys []string, err error) {
			rsp := &rspAvailable{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
			rsp.Type = msgTypeAvailableRsp
			rsp.Keys = keys
			rsp.Err = makeErrStr(err)
			sendResponse(rsp)
		})
	case msgTypeLoaded:
		s.a.Loaded(func(keys []string, err error) {
			rsp := &rspLoaded{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
			rsp.Type = msgTypeLoadedRsp
			rsp.Keys = keys
			rsp.Err = makeErrStr(err)
			sendResponse(rsp)
		})
	case msgTypeAdd:
		m := &msgAdd{msgHeader: &msgHeader{Object: omsg}}
		s.a.Add(m.Name, m.PEMPrivateKey, func(err error) {
			rsp := &rspAdd{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
			rsp.Type = msgTypeAddRsp
			rsp.Err = makeErrStr(err)
			sendResponse(rsp)
		})
	case msgTypeRemove:
		m := &msgRemove{msgHeader: &msgHeader{Object: omsg}}
		s.a.Remove(m.Name, func(err error) {
			rsp := &rspRemove{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
			rsp.Type = msgTypeRemoveRsp
			rsp.Err = makeErrStr(err)
			sendResponse(rsp)
		})
	case msgTypeLoad:
		m := &msgLoad{msgHeader: &msgHeader{Object: omsg}}
		s.a.Load(m.Name, m.Passphrase, func(err error) {
			rsp := &rspLoad{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
			rsp.Type = msgTypeLoadRsp
			rsp.Err = makeErrStr(err)
			sendResponse(rsp)
		})
	}
	return true
}

type client struct {
}

func NewClient() Available {
	return &client{}
}

func (c *client) send(msg interface{}, responseCallback func(rsp interface{}, runtimeErr error)) {
	chrome.Runtime.Call("sendMessage", chrome.ExtensionId, msg, nil, func(rsp interface{}) {
		if err := chrome.LastError(); err != nil {
			responseCallback(nil, fmt.Errorf("failed to send message: %v", err))
			return
		}
		if o := makeObject(rsp); o == js.Undefined {
			responseCallback(nil, fmt.Errorf("failed to receive response; response undefined"))
			return
		}

		responseCallback(rsp, nil)
	})
}

func (c *client) Available(callback func(keys []string, err error)) {
	msg := &msgAvailable{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
	msg.Type = msgTypeAvailable
	c.send(msg, func(rsp interface{}, runtimeErr error) {
		if runtimeErr != nil {
			callback(nil, runtimeErr)
			return
		}
		r := &rspAvailable{msgHeader: &msgHeader{Object: makeObject(rsp)}}
		callback(r.Keys, makeErr(r.Err))
	})
}

func (c *client) Loaded(callback func(keys []string, err error)) {
	msg := &msgLoaded{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
	msg.Type = msgTypeLoaded
	c.send(msg, func(rsp interface{}, runtimeErr error) {
		if runtimeErr != nil {
			callback(nil, runtimeErr)
			return
		}
		r := &rspLoaded{msgHeader: &msgHeader{Object: makeObject(rsp)}}
		callback(r.Keys, makeErr(r.Err))
	})
}

func (c *client) Add(name string, pemPrivateKey string, callback func(err error)) {
	msg := &msgAdd{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
	msg.Type = msgTypeAdd
	msg.Name = name
	msg.PEMPrivateKey = pemPrivateKey
	c.send(msg, func(rsp interface{}, runtimeErr error) {
		if runtimeErr != nil {
			callback(runtimeErr)
			return
		}
		r := &rspAdd{msgHeader: &msgHeader{Object: makeObject(rsp)}}
		callback(makeErr(r.Err))
	})
}

func (c *client) Remove(name string, callback func(err error)) {
	msg := &msgRemove{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
	msg.Type = msgTypeRemove
	msg.Name = name
	c.send(msg, func(rsp interface{}, runtimeErr error) {
		if runtimeErr != nil {
			callback(runtimeErr)
			return
		}
		r := &rspRemove{msgHeader: &msgHeader{Object: makeObject(rsp)}}
		callback(makeErr(r.Err))
	})
}

func (c *client) Load(name, passphrase string, callback func(err error)) {
	msg := &msgLoad{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
	msg.Type = msgTypeLoad
	msg.Name = name
	msg.Passphrase = passphrase
	c.send(msg, func(rsp interface{}, runtimeErr error) {
		if runtimeErr != nil {
			callback(runtimeErr)
			return
		}
		r := &rspLoad{msgHeader: &msgHeader{Object: makeObject(rsp)}}
		callback(makeErr(r.Err))
	})
}
