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
	"errors"
	"syscall/js"
	"testing"

	"github.com/google/chrome-ssh-agent/go/storage/fakes"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/norunners/vert"
)

var (
	getError    = errors.New("Storage.Get failed")
	setError    = errors.New("Storage.Set failed")
	deleteError = errors.New("Storage.Delete failed")
)

func syncSet(t *testing.T, store Area, data map[string]js.Value) {
	errc := make(chan error, 1)
	store.Set(data, func(err error) {
		errc <- err
	})
	if err := <-errc; err != nil {
		t.Fatalf("Set failed: %v", err)
	}
}

const testKeyPrefix = "key:"

func TestTypedReadAll(t *testing.T) {
	testcases := []struct {
		description string
		init        map[string]js.Value
		errs        fakes.Errs
		want        []*myStruct
		wantErr     error
	}{
		{
			description: "parse values",
			init: map[string]js.Value{
				testKeyPrefix + "1": vert.ValueOf(&myStruct{IntField: 42}).JSValue(),
				testKeyPrefix + "2": vert.ValueOf(&myStruct{StringField: "foo"}).JSValue(),
			},
			want: []*myStruct{
				&myStruct{IntField: 42},
				&myStruct{StringField: "foo"},
			},
		},
		{
			description: "skip unparseable values",
			init: map[string]js.Value{
				testKeyPrefix + "1": vert.ValueOf(&myStruct{IntField: 42}).JSValue(),
				testKeyPrefix + "2": js.ValueOf(42),
			},
			want: []*myStruct{
				&myStruct{IntField: 42},
			},
		},
		{
			description: "skip unparseable values",
			init: map[string]js.Value{
				testKeyPrefix + "1": vert.ValueOf(&myStruct{IntField: 42}).JSValue(),
				"wrong:2":           vert.ValueOf(&myStruct{StringField: "foo"}).JSValue(),
			},
			want: []*myStruct{
				&myStruct{IntField: 42},
			},
		},
		{
			description: "passes through errors",
			errs: fakes.Errs{
				Get: getError,
			},
			wantErr: getError,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			store := fakes.NewMem()
			syncSet(t, store, tc.init)

			ts := NewTyped[myStruct](store, testKeyPrefix)

			store.SetError(tc.errs)
			gotc := make(chan []*myStruct, 1)
			errc := make(chan error, 1)
			ts.ReadAll(func(values []*myStruct, err error) {
				gotc <- values
				errc <- err
			})
			got := <-gotc
			err := <-errc

			if diff := cmp.Diff(got, tc.want, cmpopts.SortSlices(myStructLess)); diff != "" {
				t.Errorf("incorrect result: -got +want: %s", diff)
			}
			if diff := cmp.Diff(err, tc.wantErr, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("incorrect error: -got +want: %s", diff)
			}
		})
	}
}

func TestTypedRead(t *testing.T) {
	const testKeyPrefix = "key:"
	testcases := []struct {
		description string
		init        map[string]js.Value
		test        func(v *myStruct) bool
		errs        fakes.Errs
		want        *myStruct
		wantErr     error
	}{
		{
			description: "value found",
			init: map[string]js.Value{
				testKeyPrefix + "1": vert.ValueOf(&myStruct{IntField: 42}).JSValue(),
				testKeyPrefix + "2": vert.ValueOf(&myStruct{StringField: "foo"}).JSValue(),
			},
			test: func(v *myStruct) bool { return v.IntField == 42 },
			want: &myStruct{IntField: 42},
		},
		{
			description: "value not found",
			init: map[string]js.Value{
				testKeyPrefix + "1": vert.ValueOf(&myStruct{IntField: 42}).JSValue(),
				testKeyPrefix + "2": vert.ValueOf(&myStruct{StringField: "foo"}).JSValue(),
			},
			test: func(v *myStruct) bool { return v.IntField == 1000 },
			want: nil,
		},
		{
			description: "passes through errors",
			errs: fakes.Errs{
				Get: getError,
			},
			test:    func(v *myStruct) bool { return v.IntField == 42 },
			wantErr: getError,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			store := fakes.NewMem()
			syncSet(t, store, tc.init)

			ts := NewTyped[myStruct](store, testKeyPrefix)

			store.SetError(tc.errs)
			gotc := make(chan *myStruct, 1)
			errc := make(chan error, 1)
			ts.Read(tc.test, func(value *myStruct, err error) {
				gotc <- value
				errc <- err
			})
			got := <-gotc
			err := <-errc

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("incorrect result: -got +want: %s", diff)
			}
			if diff := cmp.Diff(err, tc.wantErr, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("incorrect error: -got +want: %s", diff)
			}
		})
	}
}

func TestTypedWrite(t *testing.T) {
	testcases := []struct {
		description string
		init        map[string]js.Value
		write       *myStruct
		errs        fakes.Errs
		want        []*myStruct
		wantErr     error
	}{
		{
			description: "write unique value",
			init: map[string]js.Value{
				testKeyPrefix + "1": vert.ValueOf(&myStruct{IntField: 42}).JSValue(),
				testKeyPrefix + "2": vert.ValueOf(&myStruct{StringField: "foo"}).JSValue(),
			},
			write: &myStruct{IntField: 100},
			want: []*myStruct{
				&myStruct{IntField: 42},
				&myStruct{IntField: 100},
				&myStruct{StringField: "foo"},
			},
		},
		{
			description: "write duplicate value",
			init: map[string]js.Value{
				testKeyPrefix + "1": vert.ValueOf(&myStruct{IntField: 42}).JSValue(),
				testKeyPrefix + "2": vert.ValueOf(&myStruct{StringField: "foo"}).JSValue(),
			},
			write: &myStruct{IntField: 42},
			want: []*myStruct{
				&myStruct{IntField: 42},
				&myStruct{IntField: 42},
				&myStruct{StringField: "foo"},
			},
		},
		{
			description: "passes through errors",
			write:       &myStruct{IntField: 100},
			errs: fakes.Errs{
				Set: setError,
			},
			wantErr: setError,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			store := fakes.NewMem()
			syncSet(t, store, tc.init)

			ts := NewTyped[myStruct](store, testKeyPrefix)

			store.SetError(tc.errs)
			errc := make(chan error, 1)
			ts.Write(tc.write, func(err error) {
				errc <- err
			})
			err := <-errc
			if diff := cmp.Diff(err, tc.wantErr, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("incorrect error: -got +want: %s", diff)
			}
			store.SetError(fakes.Errs{})

			gotc := make(chan []*myStruct, 1)
			errc = make(chan error, 1)
			ts.ReadAll(func(values []*myStruct, err error) {
				gotc <- values
				errc <- err
			})
			got := <-gotc
			if err := <-errc; err != nil {
				t.Fatalf("ReadAll failed: %v", err)
			}
			if diff := cmp.Diff(got, tc.want, cmpopts.SortSlices(myStructLess)); diff != "" {
				t.Errorf("incorrect result: -got +want: %s", diff)
			}
		})
	}
}

func TestTypedDelete(t *testing.T) {
	testcases := []struct {
		description string
		init        map[string]js.Value
		test        func(v *myStruct) bool
		errs        fakes.Errs
		want        []*myStruct
		wantErr     error
	}{
		{
			description: "delete single value",
			init: map[string]js.Value{
				testKeyPrefix + "1": vert.ValueOf(&myStruct{IntField: 42}).JSValue(),
				testKeyPrefix + "2": vert.ValueOf(&myStruct{StringField: "foo"}).JSValue(),
			},
			test: func(v *myStruct) bool { return v.IntField == 42 },
			want: []*myStruct{
				&myStruct{StringField: "foo"},
			},
		},
		{
			description: "delete multiple values",
			init: map[string]js.Value{
				testKeyPrefix + "1": vert.ValueOf(&myStruct{IntField: 42}).JSValue(),
				testKeyPrefix + "2": vert.ValueOf(&myStruct{IntField: 100}).JSValue(),
				testKeyPrefix + "3": vert.ValueOf(&myStruct{StringField: "foo"}).JSValue(),
			},
			test: func(v *myStruct) bool { return v.IntField > 0 },
			want: []*myStruct{
				&myStruct{StringField: "foo"},
			},
		},
		{
			description: "passes through errors",
			init: map[string]js.Value{
				testKeyPrefix + "1": vert.ValueOf(&myStruct{IntField: 42}).JSValue(),
				testKeyPrefix + "2": vert.ValueOf(&myStruct{StringField: "foo"}).JSValue(),
			},
			test: func(v *myStruct) bool { return v.IntField == 42 },
			errs: fakes.Errs{
				Delete: deleteError,
			},
			want: []*myStruct{
				&myStruct{IntField: 42},
				&myStruct{StringField: "foo"},
			},
			wantErr: deleteError,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			store := fakes.NewMem()
			syncSet(t, store, tc.init)

			ts := NewTyped[myStruct](store, testKeyPrefix)

			store.SetError(tc.errs)
			errc := make(chan error, 1)
			ts.Delete(tc.test, func(err error) {
				errc <- err
			})
			err := <-errc
			if diff := cmp.Diff(err, tc.wantErr, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("incorrect error: -got +want: %s", diff)
			}
			store.SetError(fakes.Errs{})

			gotc := make(chan []*myStruct, 1)
			errc = make(chan error, 1)
			ts.ReadAll(func(values []*myStruct, err error) {
				gotc <- values
				errc <- err
			})
			got := <-gotc
			if err := <-errc; err != nil {
				t.Fatalf("ReadAll failed: %v", err)
			}
			if diff := cmp.Diff(got, tc.want, cmpopts.SortSlices(myStructLess)); diff != "" {
				t.Errorf("incorrect result: -got +want: %s", diff)
			}
		})
	}
}
