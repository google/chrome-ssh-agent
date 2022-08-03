//go:build js && wasm

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

package chrome

import (
	"fmt"
	"strings"
	"syscall/js"
	"testing"

	"github.com/google/chrome-ssh-agent/go/chrome/fakes"
	"github.com/google/chrome-ssh-agent/go/dom"
	"github.com/google/go-cmp/cmp"
	"github.com/norunners/vert"
)

func syncGet(s PersistentStore) (map[string]js.Value, error) {
	datac := make(chan map[string]js.Value, 1)
	errc := make(chan error, 1)
	s.Get(func(data map[string]js.Value, err error) {
		datac <- data
		errc <- err
	})
	return <-datac, <-errc
}

func syncGetJSON(s PersistentStore) (map[string]string, error) {
	data, err := syncGet(s)
	if err != nil {
		return nil, err
	}

	json := map[string]string{}
	for k, v := range data {
		json[k] = dom.ToJSON(v)
	}
	return json, nil
}

const (
	defaultMaxItemBytes = 1024
)

func TestSetAndGet(t *testing.T) {
	testcases := []struct {
		description  string
		maxItemBytes int
		set          map[string]js.Value
		wantRaw      map[string]string
		want         map[string]string
	}{
		{
			description: "Simple values of multiple types",
			set: map[string]js.Value{
				"myNumber": js.ValueOf(2),
				"myString": js.ValueOf("foo"),
				"myObject": vert.ValueOf(&myStruct{IntField: 2}).JSValue(),
			},
			wantRaw: map[string]string{
				"myNumber": "2",
				"myString": `"foo"`,
				"myObject": `{"intField":2,"stringField":""}`,
			},
			want: map[string]string{
				"myNumber": "2",
				"myString": `"foo"`,
				"myObject": `{"intField":2,"stringField":""}`,
			},
		},
		{
			description:  "Big values of multiple types",
			maxItemBytes: 200,
			set: map[string]js.Value{
				"myString": js.ValueOf(strings.Repeat("a", 200)),
				"myObject": vert.ValueOf(&myStruct{
					IntField:    2000000,
					StringField: strings.Repeat("a", 200),
				}).JSValue(),
			},
			wantRaw: map[string]string{
				"myString": `{"magic":"3cc36853-b864-4122-beaa-516aa24448f6","chunkKeys":["chunk-3cc36853-b864-4122-beaa-516aa24448f6:mdOE4dkIRwUUzG+lTF/Dy6yCD+029OEszXU6Gs7+OEU=","chunk-3cc36853-b864-4122-beaa-516aa24448f6:sK1Oa0GM/mf2Zaph9VLSf2U52eDriz2oDJ1BTuVnzXM="]}`,
				"chunk-3cc36853-b864-4122-beaa-516aa24448f6:mdOE4dkIRwUUzG+lTF/Dy6yCD+029OEszXU6Gs7+OEU=": `"\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`,
				"chunk-3cc36853-b864-4122-beaa-516aa24448f6:sK1Oa0GM/mf2Zaph9VLSf2U52eDriz2oDJ1BTuVnzXM=": `"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\""`,

				"myObject": `{"magic":"3cc36853-b864-4122-beaa-516aa24448f6","chunkKeys":["chunk-3cc36853-b864-4122-beaa-516aa24448f6:c7laRY8+hWoO+ZkG92GtBxBRt4nqTPCPO9Aa23Ozy4k=","chunk-3cc36853-b864-4122-beaa-516aa24448f6:Y3T3MgiFRHOCf29qP0Ox9T6qO4LCHBptaaIRCyp5uq0=","chunk-3cc36853-b864-4122-beaa-516aa24448f6:sXOAMv6Adtolok1EtQD38vtrlA0yX/Px1sEEvGW215w="]}`,
				"chunk-3cc36853-b864-4122-beaa-516aa24448f6:c7laRY8+hWoO+ZkG92GtBxBRt4nqTPCPO9Aa23Ozy4k=": `"{\"intField\":2000000,\"stringField\":\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`,
				"chunk-3cc36853-b864-4122-beaa-516aa24448f6:Y3T3MgiFRHOCf29qP0Ox9T6qO4LCHBptaaIRCyp5uq0=": `"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`,
				"chunk-3cc36853-b864-4122-beaa-516aa24448f6:sXOAMv6Adtolok1EtQD38vtrlA0yX/Px1sEEvGW215w=": `"aaaaaaaaaaaaa\"}"`,
			},
			want: map[string]string{
				"myString": fmt.Sprintf(`"%s"`, strings.Repeat("a", 200)),
				"myObject": fmt.Sprintf(`{"intField":2000000,"stringField":"%s"}`, strings.Repeat("a", 200)),
			},
		},
		{
			description:  "Boundary condition: value with max item size stored as simple value",
			maxItemBytes: 200,
			set: map[string]js.Value{
				"key": js.ValueOf(strings.Repeat("a", 195)), // Key (3) + value (195) + surrounding quotes (2) = 200
			},
			wantRaw: map[string]string{
				"key": fmt.Sprintf(`"%s"`, strings.Repeat("a", 195)), // 200 bytes total, including quotes
			},
			want: map[string]string{
				"key": fmt.Sprintf(`"%s"`, strings.Repeat("a", 195)),
			},
		},
		{
			description:  "Boundary condition: value just over maxitem size stored as big value",
			maxItemBytes: 200,
			set: map[string]js.Value{
				"key": js.ValueOf(strings.Repeat("a", 196)), // Key (3) + value (196) + surrounding quotes (2) = 201
			},
			wantRaw: map[string]string{
				"key": `{"magic":"3cc36853-b864-4122-beaa-516aa24448f6","chunkKeys":["chunk-3cc36853-b864-4122-beaa-516aa24448f6:mdOE4dkIRwUUzG+lTF/Dy6yCD+029OEszXU6Gs7+OEU=","chunk-3cc36853-b864-4122-beaa-516aa24448f6:HMz5p8STpbk1PdZVs8dvNuy7s4RBmAknToTexlTdkaE="]}`,
				"chunk-3cc36853-b864-4122-beaa-516aa24448f6:mdOE4dkIRwUUzG+lTF/Dy6yCD+029OEszXU6Gs7+OEU=": `"\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`,
				"chunk-3cc36853-b864-4122-beaa-516aa24448f6:HMz5p8STpbk1PdZVs8dvNuy7s4RBmAknToTexlTdkaE=": `"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\""`,
			},
			want: map[string]string{
				"key": fmt.Sprintf(`"%s"`, strings.Repeat("a", 196)),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			if tc.maxItemBytes == 0 {
				tc.maxItemBytes = defaultMaxItemBytes
			}

			b := &BigStorage{
				maxItemBytes: tc.maxItemBytes,
				s:            fakes.NewMemStorage(),
			}

			b.Set(tc.set, func(err error) {
				if err != nil {
					t.Fatalf("set failed: %v", err)
				}

				gotRaw, err := syncGetJSON(b.s)
				if err != nil {
					t.Fatalf("get failed for underlying storage: %v", err)
				}
				got, err := syncGetJSON(b)
				if err != nil {
					t.Fatalf("get failed for BigStorage: %v", err)
				}

				if diff := cmp.Diff(gotRaw, tc.wantRaw); diff != "" {
					t.Errorf("incorrect raw data: -got +want: %s", diff)
				}
				if diff := cmp.Diff(got, tc.want); diff != "" {
					t.Errorf("incorrect data: -got +want: %s", diff)
				}
			})
		})
	}
}

func TestDelete(t *testing.T) {
	testcases := []struct {
		description  string
		maxItemBytes int
		set          map[string]js.Value
		del          []string
		wantRaw      map[string]string
		want         map[string]string
	}{
		{
			description: "Delete simple values",
			set: map[string]js.Value{
				"myNumber": js.ValueOf(2),
				"myString": js.ValueOf("foo"),
				"myObject": vert.ValueOf(&myStruct{IntField: 2}).JSValue(),
			},
			del: []string{
				"myNumber",
				"myObject",
			},
			wantRaw: map[string]string{
				"myString": `"foo"`,
			},
			want: map[string]string{
				"myString": `"foo"`,
			},
		},
		{
			description:  "Delete big values",
			maxItemBytes: 200,
			set: map[string]js.Value{
				"myString": js.ValueOf(strings.Repeat("a", 200)),
				"myObject": vert.ValueOf(&myStruct{
					IntField:    2000000,
					StringField: strings.Repeat("a", 200),
				}).JSValue(),
			},
			del: []string{
				"myObject",
			},
			wantRaw: map[string]string{
				"myString": `{"magic":"3cc36853-b864-4122-beaa-516aa24448f6","chunkKeys":["chunk-3cc36853-b864-4122-beaa-516aa24448f6:mdOE4dkIRwUUzG+lTF/Dy6yCD+029OEszXU6Gs7+OEU=","chunk-3cc36853-b864-4122-beaa-516aa24448f6:sK1Oa0GM/mf2Zaph9VLSf2U52eDriz2oDJ1BTuVnzXM="]}`,
				"chunk-3cc36853-b864-4122-beaa-516aa24448f6:mdOE4dkIRwUUzG+lTF/Dy6yCD+029OEszXU6Gs7+OEU=": `"\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`,
				"chunk-3cc36853-b864-4122-beaa-516aa24448f6:sK1Oa0GM/mf2Zaph9VLSf2U52eDriz2oDJ1BTuVnzXM=": `"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\""`,
			},
			want: map[string]string{
				"myString": fmt.Sprintf(`"%s"`, strings.Repeat("a", 200)),
			},
		},
		{
			description:  "Delete big values that reference same data chunk",
			maxItemBytes: 200,
			set: map[string]js.Value{
				"myString":   js.ValueOf(strings.Repeat("a", 200)),
				"yourString": js.ValueOf(strings.Repeat("a", 200)),
			},
			del: []string{
				"yourString",
			},
			wantRaw: map[string]string{
				"myString": `{"magic":"3cc36853-b864-4122-beaa-516aa24448f6","chunkKeys":["chunk-3cc36853-b864-4122-beaa-516aa24448f6:mdOE4dkIRwUUzG+lTF/Dy6yCD+029OEszXU6Gs7+OEU=","chunk-3cc36853-b864-4122-beaa-516aa24448f6:sK1Oa0GM/mf2Zaph9VLSf2U52eDriz2oDJ1BTuVnzXM="]}`,
				"chunk-3cc36853-b864-4122-beaa-516aa24448f6:mdOE4dkIRwUUzG+lTF/Dy6yCD+029OEszXU6Gs7+OEU=": `"\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`,
				"chunk-3cc36853-b864-4122-beaa-516aa24448f6:sK1Oa0GM/mf2Zaph9VLSf2U52eDriz2oDJ1BTuVnzXM=": `"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\""`,
			},
			want: map[string]string{
				"myString": fmt.Sprintf(`"%s"`, strings.Repeat("a", 200)),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			if tc.maxItemBytes == 0 {
				tc.maxItemBytes = defaultMaxItemBytes
			}

			b := &BigStorage{
				maxItemBytes: tc.maxItemBytes,
				s:            fakes.NewMemStorage(),
			}

			b.Set(tc.set, func(err error) {
				if err != nil {
					t.Fatalf("set failed: %v", err)
				}

				b.Delete(tc.del, func(err error) {
					if err != nil {
						t.Fatalf("delete failed: %v", err)
					}

					gotRaw, err := syncGetJSON(b.s)
					if err != nil {
						t.Fatalf("get failed for underlying storage: %v", err)
					}
					got, err := syncGetJSON(b)
					if err != nil {
						t.Fatalf("get failed for BigStorage: %v", err)
					}

					if diff := cmp.Diff(gotRaw, tc.wantRaw); diff != "" {
						t.Errorf("incorrect raw data: -got +want: %s", diff)
					}
					if diff := cmp.Diff(got, tc.want); diff != "" {
						t.Errorf("incorrect data: -got +want: %s", diff)
					}
				})
			})
		})
	}
}
