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
	"syscall/js"
	"testing"
	"time"

	dt "github.com/google/chrome-ssh-agent/go/dom/testing"
	"github.com/google/chrome-ssh-agent/go/jsutil"
	"github.com/google/go-cmp/cmp"
)

func TestTextContent(t *testing.T) {
	d := New(dt.NewDocForTesting(`
		<div id="list"><div>first</div><div>second</div></div>
	`))
	if diff := cmp.Diff(TextContent(d.GetElement("list")), "firstsecond"); diff != "" {
		t.Errorf("incorrect text content; -got +want: %s", diff)
	}
}

func TestRemoveChildren(t *testing.T) {
	d := New(dt.NewDocForTesting(`
		<div id="list"><div>first</div><div>second</div></div>
	`))
	RemoveChildren(d.GetElement("list"))
	if diff := cmp.Diff(TextContent(d.GetElement("list")), ""); diff != "" {
		t.Errorf("incorrect text content; -got +want: %s", diff)
	}
}

func TestNewElement(t *testing.T) {
	d := New(dt.NewDocForTesting(`
		<div id="list"></div>
	`))
	AppendChild(d.GetElement("list"), d.NewElement("div"), func(child js.Value) {
		child.Set("id", "first")
		AppendChild(child, d.NewText("first"), nil)
	})
	AppendChild(d.GetElement("list"), d.NewElement("div"), func(child js.Value) {
		child.Set("id", "second")
		AppendChild(child, d.NewText("second"), nil)
	})
	if diff := cmp.Diff(TextContent(d.GetElement("list")), "firstsecond"); diff != "" {
		t.Errorf("incorrect text content; -got +want: %s", diff)
	}
	if diff := cmp.Diff(TextContent(d.GetElement("first")), "first"); diff != "" {
		t.Errorf("incorrect text content; -got +want: %s", diff)
	}
	if diff := cmp.Diff(TextContent(d.GetElement("second")), "second"); diff != "" {
		t.Errorf("incorrect text content; -got +want: %s", diff)
	}
}

func TestNewText(t *testing.T) {
	d := New(dt.NewDocForTesting(`
		<div id="list"><div>first</div><div>second</div></div>
	`))
	AppendChild(d.GetElement("list"), d.NewText("third"), nil)
	if diff := cmp.Diff(TextContent(d.GetElement("list")), "firstsecondthird"); diff != "" {
		t.Errorf("incorrect text content; -got +want: %s", diff)
	}
}

func TestClick(t *testing.T) {
	d := New(dt.NewDocForTesting(`
		<button id="btn"/>
	`))

	clicked := make(chan struct{})
	cleanup := OnClick(d.GetElement("btn"), func(ctx jsutil.AsyncContext, evt Event) { close(clicked) })
	defer cleanup()

	DoClick(d.GetElement("btn"))
	select {
	case <-clicked:
		return
	case <-time.After(5 * time.Second):
		t.Errorf("clicked callback not invoked")
	}
}

func TestDOMContentLoaded(t *testing.T) {
	d := New(dt.NewDocForTesting(`
		<p>Some Text</p>
	`))

	loaded := make(chan struct{}, 1)
	cleanup := d.OnDOMContentLoaded(func(ctx jsutil.AsyncContext) { loaded <- struct{}{} })
	defer cleanup()

	select {
	case <-loaded:
		return
	case <-time.After(5 * time.Second):
		t.Errorf("OnDOMContentLoaded not invoked")
	}
}

func TestValue(t *testing.T) {
	d := New(dt.NewDocForTesting(`
		<input id="ipt" type="text" value="Hello">
	`))

	if diff := cmp.Diff(Value(d.GetElement("ipt")), "Hello"); diff != "" {
		t.Errorf("incorrect value; -got +want: %s", diff)
	}

	SetValue(d.GetElement("ipt"), "World")
	if diff := cmp.Diff(Value(d.GetElement("ipt")), "World"); diff != "" {
		t.Errorf("incorrect value; -got +want: %s", diff)
	}
}

func joinTextContent(objs []js.Value) string {
	var result string
	for _, o := range objs {
		result = result + TextContent(o)
	}
	return result
}

func TestGetElementsByTag(t *testing.T) {
	d := New(dt.NewDocForTesting(`
		<div>foo</div>
		<pre>bar</pre>
	`))
	if diff := cmp.Diff(joinTextContent(d.GetElementsByTag("div")), "foo"); diff != "" {
		t.Errorf("incorrect text content; -got +want: %s", diff)
	}
	if diff := cmp.Diff(joinTextContent(d.GetElementsByTag("pre")), "bar"); diff != "" {
		t.Errorf("incorrect text content; -got +want: %s", diff)
	}
}
