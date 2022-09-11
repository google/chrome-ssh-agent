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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/norunners/vert"
)

func TestValueGet(t *testing.T) {
	testcases := []struct {
		description string
		init        map[string]js.Value
		want        myStruct
		wantErr     error
	}{
		{
			description: "no value present",
			want:        myStruct{},
		},
		{
			description: "parse values",
			init: map[string]js.Value{
				testKeyPrefix + "." + valueKey: vert.ValueOf(&myStruct{IntField: 42}).JSValue(),
			},
			want: myStruct{IntField: 42},
		},
		{
			description: "error on unparseable value",
			init: map[string]js.Value{
				testKeyPrefix + "." + valueKey: js.ValueOf(42),
			},
			wantErr: errParse,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			jut.DoSync(func(ctx jsutil.AsyncContext) {
				store := NewRaw(st.NewMemArea())
				if err := store.Set(ctx, tc.init); err != nil {
					t.Fatalf("Set failed: %v", err)
				}

				vs := NewValue[myStruct](store, testKeyPrefixes)

				got, err := vs.Get(ctx)
				if diff := cmp.Diff(got, tc.want); diff != "" {
					t.Errorf("incorrect result: -got +want: %s", diff)
				}
				if diff := cmp.Diff(err, tc.wantErr, cmpopts.EquateErrors()); diff != "" {
					t.Errorf("incorrect error: -got +want: %s", diff)
				}
			})
		})
	}
}

func TestValueSet(t *testing.T) {
	testcases := []struct {
		description string
		init        map[string]js.Value
		write       myStruct
		want        myStruct
	}{
		{
			description: "write initial value",
			write:       myStruct{IntField: 100},
			want:        myStruct{IntField: 100},
		},
		{
			description: "overwrite previous value",
			init: map[string]js.Value{
				testKeyPrefix + "." + valueKey: vert.ValueOf(&myStruct{StringField: "foo"}).JSValue(),
			},
			write: myStruct{IntField: 42},
			want:  myStruct{IntField: 42},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			jut.DoSync(func(ctx jsutil.AsyncContext) {
				store := NewRaw(st.NewMemArea())
				if err := store.Set(ctx, tc.init); err != nil {
					t.Fatalf("Set failed: %v", err)
				}

				vs := NewValue[myStruct](store, testKeyPrefixes)

				if err := vs.Set(ctx, tc.write); err != nil {
					t.Fatalf("Value.Set failed: %v", err)
				}

				got, err := vs.Get(ctx)
				if err != nil {
					t.Fatalf("Value.Get failed: %v", err)
				}
				if diff := cmp.Diff(got, tc.want); diff != "" {
					t.Errorf("incorrect result: -got +want: %s", diff)
				}
			})
		})
	}
}
