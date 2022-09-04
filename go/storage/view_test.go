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
	st "github.com/google/chrome-ssh-agent/go/storage/testing"
	"github.com/google/go-cmp/cmp"
)

func TestViewSet(t *testing.T) {
	testcases := []struct {
		description string
		prefixes    []string
		initRaw     map[string]js.Value
		set         map[string]js.Value
		wantRaw     map[string]string
	}{
		{
			description: "simple set",
			prefixes:    []string{"foo"},
			set: map[string]js.Value{
				"my-key": js.ValueOf(2),
			},
			wantRaw: map[string]string{
				"foo.my-key": "2",
			},
		},
		{
			description: "multiple values",
			prefixes:    []string{"foo"},
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
			description: "multiple prefixes",
			prefixes:    []string{"foo-new", "foo-old"},
			set: map[string]js.Value{
				"my-key":    js.ValueOf(2),
				"other-key": js.ValueOf("some-val"),
			},
			wantRaw: map[string]string{
				"foo-old.my-key":    "2",
				"foo-old.other-key": `"some-val"`,
				"foo-new.my-key":    "2",
				"foo-new.other-key": `"some-val"`,
			},
		},
		{
			description: "overwrite same prefix",
			prefixes:    []string{"foo"},
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
			prefixes:    []string{"foo"},
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
				raw := NewRaw(st.NewMemArea())
				if err := raw.Set(ctx, tc.initRaw); err != nil {
					t.Fatalf("initial Set failed: %v", err)
				}

				view := NewView(tc.prefixes, raw)
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
		prefixes    []string
		initRaw     map[string]js.Value
		want        map[string]string
	}{
		{
			description: "simple get",
			prefixes:    []string{"foo"},
			initRaw: map[string]js.Value{
				"foo.my-key": js.ValueOf(2),
			},
			want: map[string]string{
				"my-key": "2",
			},
		},
		{
			description: "multiple values",
			prefixes:    []string{"foo"},
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
			description: "multiple prefixes",
			prefixes:    []string{"foo-new", "foo-old"},
			initRaw: map[string]js.Value{
				"foo-new.my-key":    js.ValueOf(2),
				"foo-old.other-key": js.ValueOf("some-val"),
			},
			want: map[string]string{
				"my-key":    "2",
				"other-key": `"some-val"`,
			},
		},
		{
			description: "earlier prefix takes precedence",
			prefixes:    []string{"foo-new", "foo-old"},
			initRaw: map[string]js.Value{
				"foo-new.my-key":    js.ValueOf(4),
				"foo-old.my-key":    js.ValueOf(2),
				"foo-new.other-key": js.ValueOf("some-val"),
			},
			want: map[string]string{
				"my-key":    "4",
				"other-key": `"some-val"`,
			},
		},
		{
			description: "ignore other prefixes",
			prefixes:    []string{"foo"},
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
				raw := NewRaw(st.NewMemArea())
				if err := raw.Set(ctx, tc.initRaw); err != nil {
					t.Fatalf("initial Set failed: %v", err)
				}

				view := NewView(tc.prefixes, raw)
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
		prefixes    []string
		initRaw     map[string]js.Value
		del         []string
		wantRaw     map[string]string
	}{
		{
			description: "simple delete",
			prefixes:    []string{"foo"},
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
			prefixes:    []string{"foo"},
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
			description: "multiple prefixes",
			prefixes:    []string{"foo-new", "foo-old"},
			initRaw: map[string]js.Value{
				"foo-new.my-key":          js.ValueOf(2),
				"foo-old.other-key":       js.ValueOf("some-val"),
				"foo-new.yet-another-key": js.ValueOf("some-other-val"),
			},
			del: []string{
				"my-key",
				"other-key",
			},
			wantRaw: map[string]string{
				"foo-new.yet-another-key": `"some-other-val"`,
			},
		},
		{
			description: "ignore other prefixes",
			prefixes:    []string{"foo"},
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
				raw := NewRaw(st.NewMemArea())
				if err := raw.Set(ctx, tc.initRaw); err != nil {
					t.Fatalf("initial Set failed: %v", err)
				}

				view := NewView(tc.prefixes, raw)
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
		raw := NewRaw(st.NewMemArea())
		v1 := NewView([]string{"foo"}, raw)
		v2 := NewView([]string{"bar"}, raw)

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

func TestDeleteViewPrefixes(t *testing.T) {
	testcases := []struct {
		description string
		prefixes    []string
		initRaw     map[string]js.Value
		wantRaw     map[string]string
	}{
		{
			description: "simple delete",
			prefixes:    []string{"foo"},
			initRaw: map[string]js.Value{
				"foo.my-key":    js.ValueOf(2),
				"foo.other-key": js.ValueOf("some-val"),
				"bar.some-key":  js.ValueOf(4),
			},
			wantRaw: map[string]string{
				"bar.some-key": "4",
			},
		},
		{
			description: "multiple prefixes",
			prefixes:    []string{"foo", "bar"},
			initRaw: map[string]js.Value{
				"foo.my-key":    js.ValueOf(2),
				"foo.other-key": js.ValueOf("some-val"),
				"bar.some-key":  js.ValueOf(4),
			},
			wantRaw: map[string]string{},
		},
		{
			description: "no prefixes",
			prefixes:    []string{},
			initRaw: map[string]js.Value{
				"foo.my-key":    js.ValueOf(2),
				"foo.other-key": js.ValueOf("some-val"),
				"bar.some-key":  js.ValueOf(4),
			},
			wantRaw: map[string]string{
				"foo.my-key":    "2",
				"foo.other-key": `"some-val"`,
				"bar.some-key":  "4",
			},
		},

	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			jut.DoSync(func(ctx jsutil.AsyncContext) {
				raw := NewRaw(st.NewMemArea())
				if err := raw.Set(ctx, tc.initRaw); err != nil {
					t.Fatalf("initial Set failed: %v", err)
				}
				if err := DeleteViewPrefixes(ctx, tc.prefixes, raw); err != nil {
					t.Fatalf("DeleteViewPrefixes failed: %v", err)
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
