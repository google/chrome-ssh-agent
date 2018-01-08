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
	"testing"

	"github.com/kr/pretty"
)

func TestFunctions(t *testing.T) {
	testcases := []struct {
		description string
		sequence    func(m *MemStorage, errc chan<- error)
		errs        Errs
		want        map[string]interface{}
		wantErrs    []error
	}{
		{
			description: "set keys",
			sequence: func(m *MemStorage, errc chan<- error) {
				m.Set(map[string]interface{}{"key1": 42}, func(err error) {
					errc <- err
				})
				m.Set(map[string]interface{}{"key2": "bar"}, func(err error) {
					errc <- err
				})
			},
			want: map[string]interface{}{
				"key1": 42.0,
				"key2": "bar",
			},
		},
		{
			description: "overwrite key",
			sequence: func(m *MemStorage, errc chan<- error) {
				m.Set(map[string]interface{}{"key1": 42}, func(err error) {
					errc <- err
				})
				m.Set(map[string]interface{}{"key1": 32, "key2": "bar"}, func(err error) {
					errc <- err
				})
			},
			want: map[string]interface{}{
				"key1": 32.0,
				"key2": "bar",
			},
		},
		{
			description: "delete key",
			sequence: func(m *MemStorage, errc chan<- error) {
				m.Set(map[string]interface{}{"key1": 42, "key2": "bar"}, func(err error) {
					errc <- err
				})
				m.Delete([]string{"key1"}, func(err error) {
					errc <- err
				})
			},
			want: map[string]interface{}{
				"key2": "bar",
			},
		},
		{
			description: "delete non-existent key returns no error",
			sequence: func(m *MemStorage, errc chan<- error) {
				m.Set(map[string]interface{}{"key1": 42}, func(err error) {
					errc <- err
				})
				m.Delete([]string{"key2"}, func(err error) {
					errc <- err
				})
			},
			want: map[string]interface{}{
				"key1": 42.0,
			},
		},
		{
			description: "return errors",
			sequence: func(m *MemStorage, errc chan<- error) {
				m.Set(map[string]interface{}{"key1": 42}, func(err error) {
					errc <- err
				})
				m.Get(func(data map[string]interface{}, err error) {
					errc <- err
				})
				m.Delete([]string{"key1"}, func(err error) {
					errc <- err
				})
			},
			errs: Errs{
				Get:    errors.New("storage.Get failed"),
				Set:    errors.New("storage.Set failed"),
				Delete: errors.New("storage.Delete failed"),
			},
			want: map[string]interface{}{},
			wantErrs: []error{
				errors.New("storage.Set failed"),
				errors.New("storage.Get failed"),
				errors.New("storage.Delete failed"),
			},
		},
	}

	for _, tc := range testcases {
		m := NewMemStorage()
		errc := make(chan error, 10)

		// Execute the test case, applying any configured errors.
		func() {
			m.SetError(tc.errs)
			defer m.SetError(Errs{})

			tc.sequence(m, errc)
		}()

		// Get final state of storage.
		var final map[string]interface{}
		m.Get(func(data map[string]interface{}, err error) {
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

		if diff := pretty.Diff(errs, tc.wantErrs); diff != nil {
			t.Errorf("%s: incorrect errors; -got +want: %s", tc.description, diff)
		}
		if diff := pretty.Diff(final, tc.want); diff != nil {
			t.Errorf("%s: incorrect final data; -got +want: %s", tc.description, diff)
		}
	}
}
