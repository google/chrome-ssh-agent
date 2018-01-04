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

package dom

import (
	"github.com/gopherjs/gopherjs/js"
)

var (
	Doc = js.Global.Get("document")
)

type DOM struct {
	doc *js.Object
}

func New(doc *js.Object) *DOM {
	return &DOM{doc: doc}
}

func (d *DOM) RemoveChildren(p *js.Object) {
	for p.Call("hasChildNodes").Bool() {
		p.Call("removeChild", p.Get("firstChild"))
	}
}

func (d *DOM) NewElement(kind string) *js.Object {
	return d.doc.Call("createElement", kind)
}

func (d *DOM) NewText(text string) *js.Object {
	return d.doc.Call("createTextNode", text)
}

func (d *DOM) OnClick(o *js.Object, callback func()) {
	o.Call("addEventListener", "click", callback)
}

func (d *DOM) OnDOMContentLoaded(callback func()) {
	d.doc.Call("addEventListener", "DOMContentLoaded", callback)
}

func (d *DOM) AppendChild(parent, child *js.Object, populate func(child *js.Object)) {
	if populate != nil {
		populate(child)
	}
	parent.Call("appendChild", child)
}

func (d *DOM) GetElement(id string) *js.Object {
	return d.doc.Call("getElementById", id)
}

func (d *DOM) ShowModal(o *js.Object) {
	o.Call("showModal")
}

func (d *DOM) Close(o *js.Object) {
	o.Call("close")
}
