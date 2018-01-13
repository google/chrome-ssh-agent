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
	"testing"

	dt "github.com/google/chrome-ssh-agent/go/dom/testing"
	"github.com/gopherjs/gopherjs/js"
	"github.com/kr/pretty"
)

func TestTextContent(t *testing.T) {
	d := New(dt.NewDocForTesting(`
		<div id="list"><div>first</div><div>second</div></div>
	`))
	if diff := pretty.Diff(d.TextContent(d.GetElement("list")), "firstsecond"); diff != nil {
		t.Errorf("incorrect text content; -got +want: %s", diff)
	}
}

func TestRemoveChildren(t *testing.T) {
	d := New(dt.NewDocForTesting(`
		<div id="list"><div>first</div><div>second</div></div>
	`))
	d.RemoveChildren(d.GetElement("list"))
	if diff := pretty.Diff(d.TextContent(d.GetElement("list")), ""); diff != nil {
		t.Errorf("incorrect text content; -got +want: %s", diff)
	}
}

func TestNewElement(t *testing.T) {
	d := New(dt.NewDocForTesting(`
		<div id="list"></div>
	`))
	d.AppendChild(d.GetElement("list"), d.NewElement("div"), func(child *js.Object) {
		child.Set("id", "first")
		d.AppendChild(child, d.NewText("first"), nil)
	})
	d.AppendChild(d.GetElement("list"), d.NewElement("div"), func(child *js.Object) {
		child.Set("id", "second")
		d.AppendChild(child, d.NewText("second"), nil)
	})
	if diff := pretty.Diff(d.TextContent(d.GetElement("list")), "firstsecond"); diff != nil {
		t.Errorf("incorrect text content; -got +want: %s", diff)
	}
	if diff := pretty.Diff(d.TextContent(d.GetElement("first")), "first"); diff != nil {
		t.Errorf("incorrect text content; -got +want: %s", diff)
	}
	if diff := pretty.Diff(d.TextContent(d.GetElement("second")), "second"); diff != nil {
		t.Errorf("incorrect text content; -got +want: %s", diff)
	}
}

func TestNewText(t *testing.T) {
	d := New(dt.NewDocForTesting(`
		<div id="list"><div>first</div><div>second</div></div>
	`))
	d.AppendChild(d.GetElement("list"), d.NewText("third"), nil)
	if diff := pretty.Diff(d.TextContent(d.GetElement("list")), "firstsecondthird"); diff != nil {
		t.Errorf("incorrect text content; -got +want: %s", diff)
	}
}

func TestClick(t *testing.T) {
	d := New(dt.NewDocForTesting(`
		<button id="btn"/>
	`))
	var clicked bool
	d.OnClick(d.GetElement("btn"), func() { clicked = true })
	d.DoClick(d.GetElement("btn"))
	if !clicked {
		t.Errorf("clicked callback not invoked")
	}
}

func TestValue(t *testing.T) {
	d := New(dt.NewDocForTesting(`
		<input id="ipt" type="text" value="Hello">
	`))

	if diff := pretty.Diff(d.Value(d.GetElement("ipt")), "Hello"); diff != nil {
		t.Errorf("incorrect value; -got +want: %s", diff)
	}

	d.SetValue(d.GetElement("ipt"), "World")
	if diff := pretty.Diff(d.Value(d.GetElement("ipt")), "World"); diff != nil {
		t.Errorf("incorrect value; -got +want: %s", diff)
	}
}

func TestRemoveEventListeners(t *testing.T) {
	d := New(dt.NewDocForTesting(`
		<button id="btn"/>
	`))

	// Add a handler, and ensure it works.
	var clicked bool
	d.OnClick(d.GetElement("btn"), func() { clicked = true })
	d.DoClick(d.GetElement("btn"))
	if !clicked {
		t.Errorf("clicked callback not invoked")
	}

	// Remove the handler, and ensure the handler does not fire.
	clicked = false
	btn := d.RemoveEventListeners(d.GetElement("btn"))
	d.DoClick(d.GetElement("btn")) // Lookup by ID.
	d.DoClick(btn)                 // Use cloned element we got back.
	if clicked {
		t.Errorf("clicked callback invoked after handlers removed")
	}

	// Add the handler to the button we got back. Ensure it works.
	clicked = false
	d.OnClick(btn, func() { clicked = true })
	d.DoClick(btn)
	if !clicked {
		t.Errorf("clicked callback not invoked")
	}
}
