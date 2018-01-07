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

type Errs struct {
	Get    error
	Set    error
	Delete error
}

type MemStorage struct {
	data map[string]interface{}
	err  Errs
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		data: make(map[string]interface{}),
	}
}

func (m *MemStorage) SetError(err Errs) {
	m.err = err
}

func (m *MemStorage) Set(data map[string]interface{}, callback func(err error)) {
	if m.err.Set != nil {
		callback(m.err.Set)
		return
	}

	for k, v := range data {
		m.data[k] = toJSObject(v).Interface()
	}
	callback(nil)
}

func (m *MemStorage) Get(callback func(data map[string]interface{}, err error)) {
	if m.err.Get != nil {
		callback(nil, m.err.Get)
		return
	}

	// TODO(ralimi) Make a copy.
	callback(m.data, nil)
}

func (m *MemStorage) Delete(keys []string, callback func(err error)) {
	if m.err.Delete != nil {
		callback(m.err.Delete)
		return
	}

	for _, k := range keys {
		delete(m.data, k)
	}
	callback(nil)
}
