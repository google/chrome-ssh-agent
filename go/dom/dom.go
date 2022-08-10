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
	"time"
	
	"github.com/google/chrome-ssh-agent/go/jsutil"
)

var (
	// Doc is the default 'document' object.  This should be used for regular
	// code. See NewDocForTesting() for a Document object that can be used in
	// unit tests.
	Doc = js.Global().Get("document")

	// Console is the default 'console' object for the browser.
	Console = js.Global().Get("console")

	// Object refers to Javascript's Object class.
	Object = js.Global().Get("Object")

	// JSON refers to Javascript's JSON class.
	JSON = js.Global().Get("JSON")
)

// Event provides an API for interacting with events.
type Event struct {
	js.Value
}

// DOM provides an API for interacting with the DOM for a Document.
type DOM struct {
	doc js.Value
}

// New returns a DOM instance for interacting with the specified
// Document object.
func New(doc js.Value) *DOM {
	return &DOM{doc: doc}
}

// RemoveChildren removes all children of the specified node.
func (d *DOM) RemoveChildren(p js.Value) {
	for p.Call("hasChildNodes").Bool() {
		p.Call("removeChild", p.Get("firstChild"))
	}
}

// NewElement returns a new element with the specified tag (e.g., 'tr', 'td').
func (d *DOM) NewElement(tag string) js.Value {
	return d.doc.Call("createElement", tag)
}

// NewText returns a new text element with the specified text.
func (d *DOM) NewText(text string) js.Value {
	return d.doc.Call("createTextNode", text)
}

// DoClick simulates a click. Any callback registered by OnClick() will be
// invoked.
func (d *DOM) DoClick(o js.Value) {
	o.Call("click")
}

// OnClick registers a callback to be invoked when the specified object is
// clicked.
func (d *DOM) OnClick(o js.Value, callback func(evt Event)) {
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		callback(Event{Value: SingleArg(args)})
		return nil
	})
	o.Call("addEventListener", "click", cb)
}

// SetTimeout registers a callback to be invoked when the timeout has expired.
func SetTimeout(timeout time.Duration, callback func()) {
	cb := jsutil.OneTimeFuncOf(func(this js.Value, args []js.Value) interface{} {
		callback()
		return nil
	})
	js.Global().Call("setTimeout", cb, timeout.Milliseconds())
}

// OnDOMContentLoaded registers a callback to be invoked when the DOM has
// finished loading.
func (d *DOM) OnDOMContentLoaded(callback func()) {
	if d.doc.Get("readyState").String() != "loading" {
		SetTimeout(0, callback) // Event already fired. Invoke callback directly.
	}

	d.doc.Call(
		"addEventListener", "DOMContentLoaded",
		jsutil.OneTimeFuncOf(func(this js.Value, args []js.Value) interface{} {
			callback()
			return nil
		}))
}

// Value returns the value of an object as a string.
func (d *DOM) Value(o js.Value) string {
	return o.Get("value").String()
}

// SetValue sets the of the object.
func (d *DOM) SetValue(o js.Value, value string) {
	o.Set("value", value)
}

// TextContent returns the text content of the specified object (and its
// children).
func (d *DOM) TextContent(o js.Value) string {
	return o.Get("textContent").String()
}

// AppendChild adds the child object.  If non-nil, the populate() function is
// invoked on the child to initialize it.
func (d *DOM) AppendChild(parent, child js.Value, populate func(child js.Value)) {
	if populate != nil {
		populate(child)
	}
	parent.Call("appendChild", child)
}

// GetElement returns the element with the specified ID.
func (d *DOM) GetElement(id string) js.Value {
	return d.doc.Call("getElementById", id)
}

// GetElementsByTag returns the elements with the speciied tag.
func (d *DOM) GetElementsByTag(tag string) []js.Value {
	var result []js.Value
	elts := d.doc.Call("getElementsByTagName", tag)
	for i := 0; i < elts.Length(); i++ {
		result = append(result, elts.Index(i))
	}
	return result
}

// ShowModal shows the specified dialog as a modal dialog.
func (d *DOM) ShowModal(o js.Value) {
	if o.Get("showModal").IsUndefined() {
		// jsdom (which is used in tests) does not support showModal.
		Log("showModal() not found")
		return
	}
	o.Call("showModal")
}

// Close closes the specified dialog.
func (d *DOM) Close(o js.Value) {
	if o.Get("close").IsUndefined() {
		// jsdom (which is used in tests) does not support showModal.
		Log("close() not found")
		return
	}

	o.Call("close")
}

// RemoveEventListeners removes all event listeners from an object and its
// children.  This is accomplished by cloning the object, which has the side
// effect of *not* cloning the event listeners.   The newly-created object is
// returned.
func (d *DOM) RemoveEventListeners(o js.Value) js.Value {
	clone := o.Call("cloneNode", true)
	o.Get("parentNode").Call("replaceChild", clone, o)
	return clone
}

// Log logs general information to the Javascript Console.
func Log(format string, objs ...interface{}) {
	Console.Call("log", time.Now().Format(time.StampMilli), fmt.Sprintf(format, objs...))
}

// LogError logs an error to the Javascript Console.
func LogError(format string, objs ...interface{}) {
	Console.Call("error", time.Now().Format(time.StampMilli), fmt.Sprintf(format, objs...))
}

// LogDebug logs a debug message to the Javascript Console.
func LogDebug(format string, objs ...interface{}) {
	Console.Call("debug", time.Now().Format(time.StampMilli), fmt.Sprintf(format, objs...))
}

// ExpandArgs unpacks function arguments to target values.
func ExpandArgs(args []js.Value, target ...*js.Value) {
	// Assign args to target.
	for i := 0; i < len(args) && i < len(target); i++ {
		*(target[i]) = args[i]
	}
	// Any excessive targets are set to undefined.
	for i := len(args); i < len(target); i++ {
		*(target[i]) = js.Undefined()
	}
}

// SingleArg unpacks a single function argument and returns it.
func SingleArg(args []js.Value) js.Value {
	var val js.Value
	ExpandArgs(args, &val)
	return val
}

// ObjectKeys returns the keys for a given object.
func ObjectKeys(val js.Value) ([]string, error) {
	if val.Type() != js.TypeObject {
		return nil, fmt.Errorf("Object required; got type %s", val.Type())
	}

	var res []string
	keys := Object.Call("keys", val)
	for i := 0; i < keys.Length(); i++ {
		res = append(res, keys.Index(i).String())
	}
	return res, nil
}

// ToJSON converts the supplied value to a JSON string.
func ToJSON(val js.Value) string {
	return JSON.Call("stringify", val).String()
}

// FromJSON converts the supplied JSON string to a Javascript value.
func FromJSON(s string) js.Value {
	defer func() {
		if r := recover(); r != nil {
			LogError("Failed to parse JSON string; returning default. Error: %v", r)
		}
	}()

	return JSON.Call("parse", s)
}
