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
	"log"

	"github.com/gopherjs/gopherjs/js"
)

var (
	// Doc is the default 'document' object.
	Doc = js.Global.Get("document")
)

// DOM provides an API for interacting with the DOM for a Document.
type DOM struct {
	doc *js.Object
}

// New returns a DOM instance for interacting with the specified
// Document object.
func New(doc *js.Object) *DOM {
	return &DOM{doc: doc}
}

// RemoveChildren removes all children of the specified node.
func (d *DOM) RemoveChildren(p *js.Object) {
	for p.Call("hasChildNodes").Bool() {
		p.Call("removeChild", p.Get("firstChild"))
	}
}

// NewElement returns a new element with the specified tag (e.g., 'tr', 'td').
func (d *DOM) NewElement(tag string) *js.Object {
	return d.doc.Call("createElement", kind)
}

// NewText returns a new text element with the specified text.
func (d *DOM) NewText(text string) *js.Object {
	return d.doc.Call("createTextNode", text)
}

// DoClick simulates a click. Any callback registered by OnClick() will be
// invoked.
func (d *DOM) DoClick(o *js.Object) {
	o.Call("click")
}

// OnClick registers a callback to be invoked when the specified object is
// clicked.
func (d *DOM) OnClick(o *js.Object, callback func()) {
	o.Call("addEventListener", "click", callback)
}

// DoDOMContentLoaded simulates the DOMContentLoaded event. Any callback
// registered by OnDOMContentLoaded() will be invoked.
func (d *DOM) DoDOMContentLoaded() {
	event := d.doc.Call("createEvent", "Event")
	event.Call("initEvent", "DOMContentLoaded", true, true)
	d.doc.Call("dispatchEvent", event)
}

// OnDOMContentLoaded registers a callback to be invoked when the DOM has
// finished loading.
func (d *DOM) OnDOMContentLoaded(callback func()) {
	d.doc.Call("addEventListener", "DOMContentLoaded", callback)
}

// Value returns the value of an object as a string.
func (d *DOM) Value(o *js.Object) string {
	return o.Get("value").String()
}

// SetValue sets the of the object.
func (d *DOM) SetValue(o *js.Object, value string) {
	o.Set("value", value)
}

// TextContent returns the text content of the specified object (and its
// children).
func (d *DOM) TextContent(o *js.Object) string {
	return o.Get("textContent").String()
}

// AppendChild adds the child object.  If non-nil, the populate() function is
// invoked on the child to initialize it.
func (d *DOM) AppendChild(parent, child *js.Object, populate func(child *js.Object)) {
	if populate != nil {
		populate(child)
	}
	parent.Call("appendChild", child)
}

// GetElement returns the element with the specified ID.
func (d *DOM) GetElement(id string) *js.Object {
	return d.doc.Call("getElementById", id)
}

// ShowModal shows the specified dialog as a modal dialog.
func (d *DOM) ShowModal(o *js.Object) {
	if o.Get("showModal") == js.Undefined {
		// jsdom (which is used in tests) does not support showModal.
		log.Printf("showModal() not found")
		return
	}
	o.Call("showModal")
}

// Close closes the specified dialog.
func (d *DOM) Close(o *js.Object) {
	if o.Get("close") == js.Undefined {
		// jsdom (which is used in tests) does not support showModal.
		log.Printf("close() not found")
		return
	}

	o.Call("close")
}
