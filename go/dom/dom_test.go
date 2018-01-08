package dom

import (
	"testing"

	"github.com/gopherjs/gopherjs/js"
	"github.com/kr/pretty"
)

func TestTextContent(t *testing.T) {
	d := New(NewDocForTesting(`
		<div id="list"><div>first</div><div>second</div></div>
	`))
	if diff := pretty.Diff(d.TextContent(d.GetElement("list")), "firstsecond"); diff != nil {
		t.Errorf("incorrect text content; -got +want: %s", diff)
	}
}

func TestRemoveChildren(t *testing.T) {
	d := New(NewDocForTesting(`
		<div id="list"><div>first</div><div>second</div></div>
	`))
	d.RemoveChildren(d.GetElement("list"))
	if diff := pretty.Diff(d.TextContent(d.GetElement("list")), ""); diff != nil {
		t.Errorf("incorrect text content; -got +want: %s", diff)
	}
}

func TestNewElement(t *testing.T) {
	d := New(NewDocForTesting(`
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
	d := New(NewDocForTesting(`
		<div id="list"><div>first</div><div>second</div></div>
	`))
	d.AppendChild(d.GetElement("list"), d.NewText("third"), nil)
	if diff := pretty.Diff(d.TextContent(d.GetElement("list")), "firstsecondthird"); diff != nil {
		t.Errorf("incorrect text content; -got +want: %s", diff)
	}
}

func TestClick(t *testing.T) {
	d := New(NewDocForTesting(`
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
	d := New(NewDocForTesting(`
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
