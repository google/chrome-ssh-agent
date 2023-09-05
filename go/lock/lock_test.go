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

package lock

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/chrome-ssh-agent/go/jsutil"
	jut "github.com/google/chrome-ssh-agent/go/jsutil/testing"
)

func TestExclusive(t *testing.T) {
	t.Parallel()

	jut.DoSync(func(ctx jsutil.AsyncContext) {
		const workers = 10
		var count, concurrent, finished int32

		// Spawn all workers concurrently.
		var promises []*jsutil.Promise
		for i := 0; i < workers; i++ {
			i := i
			promises = append(promises, Async("my-resource", func(ctx jsutil.AsyncContext) {
				t.Logf("Start %d", i)
				defer t.Logf("End %d", i)
				defer atomic.AddInt32(&finished, 1)

				// Increment a counter while worker is running.
				atomic.AddInt32(&count, 1)
				defer atomic.AddInt32(&count, -1)

				// Give some time for other workers to start up.
				time.Sleep(100 * time.Millisecond)
				// Check if at least one other worker is currently running.
				if atomic.LoadInt32(&count) >= 2 {
					atomic.AddInt32(&concurrent, 1)
				}
			}))
		}

		// Wait for all routines to complete.
		for _, p := range promises {
			if _, err := p.Await(ctx); err != nil {
				t.Errorf("promise returned error: %v", err)
			}
		}

		// Validate that all workers completed.
		if f := atomic.LoadInt32(&finished); f != workers {
			t.Errorf("incorrect number of workers finished; got %d", f)
		}

		// Validate that none of the routines observed concurrent access.
		if c := atomic.LoadInt32(&concurrent); c > 0 {
			t.Errorf("%d workers observed concurrent access", c)
		}
	})
}
