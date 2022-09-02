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

	"github.com/google/chrome-ssh-agent/go/jsutil"
	jut "github.com/google/chrome-ssh-agent/go/jsutil/testing"
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
		sequence    func(ctx jsutil.AsyncContext, m *Mem, errc chan<- error)
		errs        Errs
		want        map[string]js.Value
		wantErrs    []error
	}{
		{
			description: "set keys",
			sequence: func(ctx jsutil.AsyncContext, m *Mem, errc chan<- error) {
				if err := m.Set(ctx, map[string]js.Value{"key1": js.ValueOf(42)}); err != nil {
					errc <- err
				}
				if err := m.Set(ctx, map[string]js.Value{"key2": js.ValueOf("bar")}); err != nil {
					errc <- err
				}
			},
			want: map[string]js.Value{
				"key1": js.ValueOf(42.0),
				"key2": js.ValueOf("bar"),
			},
		},
		{
			description: "overwrite key",
			sequence: func(ctx jsutil.AsyncContext, m *Mem, errc chan<- error) {
				if err := m.Set(ctx, map[string]js.Value{"key1": js.ValueOf(42)}); err != nil {
					errc <- err
				}
				if err := m.Set(ctx, map[string]js.Value{"key1": js.ValueOf(32), "key2": js.ValueOf("bar")}); err != nil {
					errc <- err
				}
			},
			want: map[string]js.Value{
				"key1": js.ValueOf(32.0),
				"key2": js.ValueOf("bar"),
			},
		},
		{
			description: "delete key",
			sequence: func(ctx jsutil.AsyncContext, m *Mem, errc chan<- error) {
				if err := m.Set(ctx, map[string]js.Value{"key1": js.ValueOf(42), "key2": js.ValueOf("bar")}); err != nil {
					errc <- err
				}
				if err := m.Delete(ctx, []string{"key1"}); err != nil {
					errc <- err
				}
			},
			want: map[string]js.Value{
				"key2": js.ValueOf("bar"),
			},
		},
		{
			description: "delete non-existent key returns no error",
			sequence: func(ctx jsutil.AsyncContext, m *Mem, errc chan<- error) {
				if err := m.Set(ctx, map[string]js.Value{"key1": js.ValueOf(42)}); err != nil {
					errc <- err
				}
				if err := m.Delete(ctx, []string{"key2"}); err != nil {
					errc <- err
				}
			},
			want: map[string]js.Value{
				"key1": js.ValueOf(42.0),
			},
		},
		{
			description: "return errors",
			sequence: func(ctx jsutil.AsyncContext, m *Mem, errc chan<- error) {
				if err := m.Set(ctx, map[string]js.Value{"key1": js.ValueOf(42)}); err != nil {
					errc <- err
				}
				if _, err := m.Get(ctx); err != nil {
					errc <- err
				}
				if err := m.Delete(ctx, []string{"key1"}); err != nil {
					errc <- err
				}
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

			m := NewMem()
			errc := make(chan error, 10)

			// Execute the test case, applying any configured errors.
			// Then, get the final state of storage.
			var final map[string]js.Value
			jut.DoSync(func(ctx jsutil.AsyncContext) {
				defer close(errc)

				func() {
					m.SetError(tc.errs)
					defer m.SetError(Errs{})

					tc.sequence(ctx, m, errc)
				}()

				var err error
				if final, err = m.Get(ctx); err != nil {
					errc <- err
				}
			})

			var errs []error
			for err := range errc {
				errs = append(errs, err)
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
