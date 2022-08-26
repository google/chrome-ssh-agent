// Copyright 2022 Google LLC
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

package storage

import (
	"syscall/js"
	"testing"

	"github.com/google/chrome-ssh-agent/go/jsutil"
	jut "github.com/google/chrome-ssh-agent/go/jsutil/testing"
	"github.com/google/chrome-ssh-agent/go/storage/fakes"
	"github.com/google/go-cmp/cmp"
)

func TestViewSet(t *testing.T) {
	testcases := []struct {
		description string
		prefix      string
		initRaw     map[string]js.Value
		set         map[string]js.Value
		wantRaw     map[string]string
	}{
		{
			description: "simple set",
			prefix:      "foo",
			set: map[string]js.Value{
				"my-key": js.ValueOf(2),
			},
			wantRaw: map[string]string{
				"foo.my-key": "2",
			},
		},
		{
			description: "multiple values",
			prefix:      "foo",
			set: map[string]js.Value{
				"my-key":    js.ValueOf(2),
				"other-key": js.ValueOf("some-val"),
			},
			wantRaw: map[string]string{
				"foo.my-key":    "2",
				"foo.other-key": `"some-val"`,
			},
		},
		{
			description: "overwrite same prefix",
			prefix:      "foo",
			initRaw: map[string]js.Value{
				"foo.my-key":    js.ValueOf(3),
				"foo.other-key": js.ValueOf("other-val"),
			},
			set: map[string]js.Value{
				"my-key":    js.ValueOf(2),
				"other-key": js.ValueOf("some-val"),
			},
			wantRaw: map[string]string{
				"foo.my-key":    "2",
				"foo.other-key": `"some-val"`,
			},
		},
		{
			description: "don't overwrite other prefixes",
			prefix:      "foo",
			initRaw: map[string]js.Value{
				"bar.my-key":    js.ValueOf(3),
				"bar.other-key": js.ValueOf("other-val"),
			},
			set: map[string]js.Value{
				"my-key":    js.ValueOf(2),
				"other-key": js.ValueOf("some-val"),
			},
			wantRaw: map[string]string{
				"bar.my-key":    "3",
				"bar.other-key": `"other-val"`,
				"foo.my-key":    "2",
				"foo.other-key": `"some-val"`,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			jut.DoSync(func(ctx jsutil.AsyncContext) {
				raw := fakes.NewMem()
				if err := raw.Set(ctx, tc.initRaw); err != nil {
					t.Fatalf("initial Set failed: %v", err)
				}

				view := NewView(tc.prefix, raw)
				if err := view.Set(ctx, tc.set); err != nil {
					t.Fatalf("View.Set failed: %v", err)
				}

				got, err := getJSON(ctx, raw)
				if err != nil {
					t.Fatalf("Get failed: %v", err)
				}

				if diff := cmp.Diff(got, tc.wantRaw); diff != "" {
					t.Errorf("incorrect result; -got +want: %s", diff)
				}
			})
		})
	}
}

func TestViewGet(t *testing.T) {
	testcases := []struct {
		description string
		prefix      string
		initRaw     map[string]js.Value
		want        map[string]string
	}{
		{
			description: "simple get",
			prefix:      "foo",
			initRaw: map[string]js.Value{
				"foo.my-key": js.ValueOf(2),
			},
			want: map[string]string{
				"my-key": "2",
			},
		},
		{
			description: "multiple values",
			prefix:      "foo",
			initRaw: map[string]js.Value{
				"foo.my-key":    js.ValueOf(2),
				"foo.other-key": js.ValueOf("some-val"),
			},
			want: map[string]string{
				"my-key":    "2",
				"other-key": `"some-val"`,
			},
		},
		{
			description: "ignore other prefixes",
			prefix:      "foo",
			initRaw: map[string]js.Value{
				"my-key":        js.ValueOf("my-val"), // No prefix
				"bar.my-key":    js.ValueOf(3),        // Different prefix
				"foo.other-key": js.ValueOf("other-val"),
			},
			want: map[string]string{
				"other-key": `"other-val"`,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			jut.DoSync(func(ctx jsutil.AsyncContext) {
				raw := fakes.NewMem()
				if err := raw.Set(ctx, tc.initRaw); err != nil {
					t.Fatalf("initial Set failed: %v", err)
				}

				view := NewView(tc.prefix, raw)
				got, err := getJSON(ctx, view)
				if err != nil {
					t.Fatalf("View.Get failed: %v", err)
				}

				if diff := cmp.Diff(got, tc.want); diff != "" {
					t.Errorf("incorrect result; -got +want: %s", diff)
				}
			})
		})
	}
}

func TestViewDelete(t *testing.T) {
	testcases := []struct {
		description string
		prefix      string
		initRaw     map[string]js.Value
		del         []string
		wantRaw     map[string]string
	}{
		{
			description: "simple delete",
			prefix:      "foo",
			initRaw: map[string]js.Value{
				"foo.my-key":    js.ValueOf(2),
				"foo.other-key": js.ValueOf("some-val"),
			},
			del: []string{
				"my-key",
			},
			wantRaw: map[string]string{
				"foo.other-key": `"some-val"`,
			},
		},
		{
			description: "multiple values",
			prefix:      "foo",
			initRaw: map[string]js.Value{
				"foo.my-key":          js.ValueOf(2),
				"foo.other-key":       js.ValueOf("some-val"),
				"foo.yet-another-key": js.ValueOf("some-other-val"),
			},
			del: []string{
				"my-key",
				"other-key",
			},
			wantRaw: map[string]string{
				"foo.yet-another-key": `"some-other-val"`,
			},
		},
		{
			description: "ignore other prefixes",
			prefix:      "foo",
			initRaw: map[string]js.Value{
				"my-key":        js.ValueOf("my-val"), // No prefix
				"bar.my-key":    js.ValueOf(3),        // Different prefix
				"foo.other-key": js.ValueOf("other-val"),
			},
			del: []string{
				"other-key",
			},
			wantRaw: map[string]string{
				"my-key":     `"my-val"`,
				"bar.my-key": "3",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			jut.DoSync(func(ctx jsutil.AsyncContext) {
				raw := fakes.NewMem()
				if err := raw.Set(ctx, tc.initRaw); err != nil {
					t.Fatalf("initial Set failed: %v", err)
				}

				view := NewView(tc.prefix, raw)
				if err := view.Delete(ctx, tc.del); err != nil {
					t.Fatalf("View.Delete failed: %v", err)
				}

				got, err := getJSON(ctx, raw)
				if err != nil {
					t.Fatalf("Get failed: %v", err)
				}

				if diff := cmp.Diff(got, tc.wantRaw); diff != "" {
					t.Errorf("incorrect result; -got +want: %s", diff)
				}
			})
		})
	}
}

func TestMultipleViews(t *testing.T) {
	jut.DoSync(func(ctx jsutil.AsyncContext) {
		raw := fakes.NewMem()
		v1 := NewView("foo", raw)
		v2 := NewView("bar", raw)

		if err := v1.Set(ctx, map[string]js.Value{"my-key": js.ValueOf(2)}); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
		if err := v2.Set(ctx, map[string]js.Value{"my-key": js.ValueOf(3)}); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		gotV1, err := getJSON(ctx, v1)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		gotV2, err := getJSON(ctx, v2)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if diff := cmp.Diff(gotV1, map[string]string{"my-key": "2"}); diff != "" {
			t.Errorf("incorrect view1: -got +want: %s", diff)
		}
		if diff := cmp.Diff(gotV2, map[string]string{"my-key": "3"}); diff != "" {
			t.Errorf("incorrect view2: -got +want: %s", diff)
		}
	})
}
