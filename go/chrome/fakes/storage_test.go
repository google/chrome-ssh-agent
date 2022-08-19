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

package fakes

import (
	"errors"
	"syscall/js"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var (
	getError    = errors.New("Get() failed")
	setError    = errors.New("Set() failed")
	deleteError = errors.New("Delete() failed")
)

func TestFunctions(t *testing.T) {
	testcases := []struct {
		description string
		sequence    func(m *MemStorage, errc chan<- error)
		errs        Errs
		want        map[string]js.Value
		wantErrs    []error
	}{
		{
			description: "set keys",
			sequence: func(m *MemStorage, errc chan<- error) {
				m.Set(map[string]js.Value{"key1": js.ValueOf(42)}, func(err error) {
					errc <- err
				})
				m.Set(map[string]js.Value{"key2": js.ValueOf("bar")}, func(err error) {
					errc <- err
				})
			},
			want: map[string]js.Value{
				"key1": js.ValueOf(42.0),
				"key2": js.ValueOf("bar"),
			},
		},
		{
			description: "overwrite key",
			sequence: func(m *MemStorage, errc chan<- error) {
				m.Set(map[string]js.Value{"key1": js.ValueOf(42)}, func(err error) {
					errc <- err
				})
				m.Set(map[string]js.Value{"key1": js.ValueOf(32), "key2": js.ValueOf("bar")}, func(err error) {
					errc <- err
				})
			},
			want: map[string]js.Value{
				"key1": js.ValueOf(32.0),
				"key2": js.ValueOf("bar"),
			},
		},
		{
			description: "delete key",
			sequence: func(m *MemStorage, errc chan<- error) {
				m.Set(map[string]js.Value{"key1": js.ValueOf(42), "key2": js.ValueOf("bar")}, func(err error) {
					errc <- err
				})
				m.Delete([]string{"key1"}, func(err error) {
					errc <- err
				})
			},
			want: map[string]js.Value{
				"key2": js.ValueOf("bar"),
			},
		},
		{
			description: "delete non-existent key returns no error",
			sequence: func(m *MemStorage, errc chan<- error) {
				m.Set(map[string]js.Value{"key1": js.ValueOf(42)}, func(err error) {
					errc <- err
				})
				m.Delete([]string{"key2"}, func(err error) {
					errc <- err
				})
			},
			want: map[string]js.Value{
				"key1": js.ValueOf(42.0),
			},
		},
		{
			description: "return errors",
			sequence: func(m *MemStorage, errc chan<- error) {
				m.Set(map[string]js.Value{"key1": js.ValueOf(42)}, func(err error) {
					errc <- err
				})
				m.Get(func(data map[string]js.Value, err error) {
					errc <- err
				})
				m.Delete([]string{"key1"}, func(err error) {
					errc <- err
				})
			},
			errs: Errs{
				Get:    getError,
				Set:    setError,
				Delete: deleteError,
			},
			want: map[string]js.Value{},
			wantErrs: []error{
				setError,
				getError,
				deleteError,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {

			m := NewMemStorage()
			errc := make(chan error, 10)

			// Execute the test case, applying any configured errors.
			func() {
				m.SetError(tc.errs)
				defer m.SetError(Errs{})

				tc.sequence(m, errc)
			}()

			// Get final state of storage.
			var final map[string]js.Value
			m.Get(func(data map[string]js.Value, err error) {
				final = data
				errc <- err
			})

			close(errc)

			var errs []error
			for err := range errc {
				if err != nil {
					errs = append(errs, err)
				}
			}

			if diff := cmp.Diff(errs, tc.wantErrs, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("incorrect errors; -got +want: %s", diff)
			}
			if diff := cmp.Diff(final, tc.want); diff != "" {
				t.Errorf("incorrect final data; -got +want: %s", diff)
			}
		})
	}
}
