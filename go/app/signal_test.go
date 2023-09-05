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

package app

import (
	"testing"
	"time"
)

func TestWaitForSignal(t *testing.T) {
	s := newSignal()

	// Wait for Signal() to be invoked. Validate that signal
	// was invoked.
	signalled := false
	waited := make(chan struct{})
	go func() {
		s.Wait()
		if !signalled {
			t.Errorf("Wait returned before Signal invoked")
		}
		close(waited)
	}()

	// Give time for the wait routine to start and block.
	time.Sleep(100 * time.Millisecond)

	// Wake up the wait routine.
	signalled = true
	s.Signal()

	// Ensure wait routine has completed.
	<-waited
}

func TestMultipleWaitOnSameSignal(t *testing.T) {
	s := newSignal()
	s.Signal()
	t.Log("wait #1")
	s.Wait()
	t.Log("wait #2")
	s.Wait()
	t.Log("wait #3")
	s.Wait()
}
