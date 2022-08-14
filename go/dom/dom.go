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

// Package dom provides APIs for interacting with the DOM.
package dom

import (
	"fmt"
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/jsutil"
)

var (
	// document is the default 'document' object.  This should be used for
	// regular code. See NewDocForTesting() for a Document object that can
	// be used in unit tests.
	document = js.Global().Get("document")
)

// Event provides an API for interacting with events.
type Event struct {
	js.Value
}

// Doc provides an API for interacting with the DOM for a Document.
type Doc struct {
	doc js.Value
}

// New returns a Doc instance for interacting with the specified
// Document object.
func New(doc js.Value) *Doc {
	if doc.IsUndefined() || doc.IsNull() {
		doc = document
	}
	return &Doc{doc: doc}
}

// NewElement returns a new element with the specified tag (e.g., 'tr', 'td').
func (d *Doc) NewElement(tag string) js.Value {
	return d.doc.Call("createElement", tag)
}

// NewText returns a new text element with the specified text.
func (d *Doc) NewText(text string) js.Value {
	return d.doc.Call("createTextNode", text)
}

// OnDOMContentLoaded registers a callback to be invoked when the DOM has
// finished loading.
func (d *Doc) OnDOMContentLoaded(callback func()) jsutil.CleanupFunc {
	if d.doc.Get("readyState").String() != "loading" {
		jsutil.SetTimeout(0, callback) // Event already fired. Invoke callback directly.
		return func() {}
	}

	return addEventListener(
		d.doc, "DOMContentLoaded",
		func(this js.Value, args []js.Value) interface{} {
			callback()
			return nil
		})
}

// GetElement returns the element with the specified ID.
func (d *Doc) GetElement(id string) js.Value {
	return d.doc.Call("getElementById", id)
}

// GetElementsByTag returns the elements with the speciied tag.
func (d *Doc) GetElementsByTag(tag string) []js.Value {
	var result []js.Value
	elts := d.doc.Call("getElementsByTagName", tag)
	for i := 0; i < elts.Length(); i++ {
		result = append(result, elts.Index(i))
	}
	return result
}

// RemoveChildren removes all children of the specified node.
func RemoveChildren(p js.Value) {
	for p.Call("hasChildNodes").Bool() {
		p.Call("removeChild", p.Get("firstChild"))
	}
}

// DoClick simulates a click. Any callback registered by OnClick() will be
// invoked.
func DoClick(o js.Value) {
	o.Call("click")
}

// addEventListener adds a function that will be invoked on the specified event
// for an object.  The returned cleanup function must be invoked to cleanup the
// function.
func addEventListener(o js.Value, event string, f func(this js.Value, args []js.Value) interface{}) jsutil.CleanupFunc {
	fo := js.FuncOf(f)
	o.Call("addEventListener", event, fo)
	return func() {
		o.Call("removeEventListener", event, fo)
		fo.Release()
	}
}

// OnClick registers a callback to be invoked when the specified object is
// clicked.
func OnClick(o js.Value, callback func(evt Event)) jsutil.CleanupFunc {
	return addEventListener(
		o, "click",
		func(this js.Value, args []js.Value) interface{} {
			callback(Event{Value: jsutil.SingleArg(args)})
			return nil
		})
}

// OnSubmit registers a callback to be invoked when the specified form is
// submitted.
func OnSubmit(o js.Value, callback func(evt Event)) jsutil.CleanupFunc {
	return addEventListener(
		o, "submit",
		func(this js.Value, args []js.Value) interface{} {
			callback(Event{Value: jsutil.SingleArg(args)})
			return nil
		})
}

// ID returns the element ID of an object as a string.
func ID(o js.Value) string {
	return o.Get("id").String()
}

// Value returns the value of an object as a string.
func Value(o js.Value) string {
	return o.Get("value").String()
}

// SetValue sets the of the object.
func SetValue(o js.Value, value string) {
	o.Set("value", value)
}

// TextContent returns the text content of the specified object (and its
// children).
func TextContent(o js.Value) string {
	return o.Get("textContent").String()
}

// AppendChild adds the child object.  If non-nil, the populate() function is
// invoked on the child to initialize it.
func AppendChild(parent, child js.Value, populate func(child js.Value)) {
	if populate != nil {
		populate(child)
	}
	parent.Call("appendChild", child)
}

// Dialog represents an HTML dialog.
type Dialog struct {
	dialog js.Value

	simOnClose js.Func
}

// NewDialog returns a dialog wrapping the specified element.
func NewDialog(dialog js.Value) *Dialog {
	return &Dialog{
		dialog: dialog,
	}
}

// ShowModal shows the dialog as a modal dialog.
func (d *Dialog) ShowModal() {
	if d.dialog.Get("showModal").IsUndefined() {
		// jsdom (which is used in tests) does not support showModal.
		jsutil.Log("showModal() not found")
		return
	}
	d.dialog.Call("showModal")
}

// Close closes the dialog.
func (d *Dialog) Close() {
	if d.dialog.Get("close").IsUndefined() {
		// jsdom (which is used in tests) does not support close.
		jsutil.Log("close() not found")
		// Simulate 'close' event; we need to ensure OnClose is triggered.
		// Using Javascript's dispatchEvent(new Event('close')) doesn't
		// work; it appears to send node.js into an infinite loop.
		if !d.simOnClose.IsUndefined() {
			d.simOnClose.Invoke()
		}
		return
	}

	d.dialog.Call("close")
}

// OnClose registers the specified callback to be invoked when the dialog is
// closed. The returned function must be invoked to cleanup when it is no longer
// needed.
func (d *Dialog) OnClose(callback func(evt Event)) jsutil.CleanupFunc {
	if d.dialog.Get("close").IsUndefined() {
		// jsdom (which is used in tests) does not support close. Store
		// the OnClose event for a subsequent invocation of Close().
		if !d.simOnClose.IsUndefined() {
			panic(fmt.Errorf("Multiple simulated OnClose handlers not supported"))
		}
		d.simOnClose = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			callback(Event{Value: jsutil.SingleArg(args)})
			return nil
		})
		return d.simOnClose.Release
	}

	return addEventListener(
		d.dialog, "close",
		func(this js.Value, args []js.Value) interface{} {
			callback(Event{Value: jsutil.SingleArg(args)})
			return nil
		})
}
