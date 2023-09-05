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
	"github.com/norunners/vert"
)

func dataToJSON(data map[string]js.Value) map[string]string {
	json := map[string]string{}
	for k, v := range data {
		json[k] = jsutil.ToJSON(v)
	}
	return json
}

type myStruct struct {
	IntField    int    `js:"intField"`
	StringField string `js:"stringField"`
}

func myStructLess(a *myStruct, b *myStruct) bool {
	if a.IntField != b.IntField {
		return a.IntField < b.IntField
	}
	return a.StringField < b.StringField
}

func TestRawSetAndGet(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		data        map[string]js.Value
	}{
		{
			description: "empty data",
			data:        map[string]js.Value{},
		},
		{
			description: "simple entry",
			data: map[string]js.Value{
				"key": js.ValueOf(2),
			},
		},
		{
			description: "map entry",
			data: map[string]js.Value{
				"key": vert.ValueOf(map[string]int{
					"field": 2,
				}).JSValue(),
			},
		},
		{
			description: "object entry",
			data: map[string]js.Value{
				"key": vert.ValueOf(&myStruct{
					IntField: 2,
				}).JSValue(),
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			jut.DoSync(func(ctx jsutil.AsyncContext) {
				s := NewRaw(st.NewMemArea())
				if err := s.Set(ctx, tc.data); err != nil {
					t.Fatalf("Set failed: %v", err)
				}
				got, err := s.Get(ctx)
				if err != nil {
					t.Fatalf("Get failed: %v", err)
				}
				if diff := cmp.Diff(dataToJSON(got), dataToJSON(tc.data)); diff != "" {
					t.Errorf("incorrect data; -got +want: %s", diff)
				}
			})
		})
	}
}

func TestRawDelete(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		init        map[string]js.Value
		del         []string
		want        map[string]js.Value
	}{
		{
			description: "delete single entry",
			init: map[string]js.Value{
				"key1": js.ValueOf(1),
				"key2": js.ValueOf(2),
				"key3": js.ValueOf(3),
			},
			del: []string{"key2"},
			want: map[string]js.Value{
				"key1": js.ValueOf(1),
				"key3": js.ValueOf(3),
			},
		},
		{
			description: "delete multiple entries",
			init: map[string]js.Value{
				"key1": js.ValueOf(1),
				"key2": js.ValueOf(2),
				"key3": js.ValueOf(3),
			},
			del: []string{"key2", "key3"},
			want: map[string]js.Value{
				"key1": js.ValueOf(1),
			},
		},
		{
			description: "delete missing entry",
			init: map[string]js.Value{
				"key": js.ValueOf(2),
			},
			del: []string{"missing"},
			want: map[string]js.Value{
				"key": js.ValueOf(2),
			},
		},
		{
			description: "delete no entry",
			init: map[string]js.Value{
				"key": js.ValueOf(2),
			},
			del: []string{},
			want: map[string]js.Value{
				"key": js.ValueOf(2),
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			jut.DoSync(func(ctx jsutil.AsyncContext) {
				s := NewRaw(st.NewMemArea())
				if err := s.Set(ctx, tc.init); err != nil {
					t.Fatalf("Set failed: %v", err)
				}
				if err := s.Delete(ctx, tc.del); err != nil {
					t.Fatalf("Delete failed: %v", err)
				}

				got, err := s.Get(ctx)
				if err != nil {
					t.Fatalf("Get failed: %v", err)
				}
				if diff := cmp.Diff(dataToJSON(got), dataToJSON(tc.want)); diff != "" {
					t.Errorf("incorrect data; -got +want: %s", diff)
				}
			})
		})
	}
}
