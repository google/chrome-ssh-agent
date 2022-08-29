//go:build js

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
	"sync"
)

// signal is a simple way to signal to callers that a condition has become true.
// Wait() may be invoked multiple times, and it will return immediately as long
// as Signal() has previously been invoked.
type signal struct {
	lock      *sync.Mutex
	cond      *sync.Cond
	signalled bool // Protected by lock.
}

// newSignal returns a new signal.
func newSignal() *signal {
	lock := &sync.Mutex{}
	cond := sync.NewCond(lock)
	return &signal{
		lock: lock,
		cond: cond,
	}
}

// Signal asserts that a condition has become true.  Any clients blocked on
// Wait will be woken up, and all future invocations of Wait will return
// immediately.
//
// Signal may be invoked multiple times.
func (s *signal) Signal() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.signalled = true
	s.cond.Broadcast()
}

// Wait blocks until Signal has been invoked.
func (s *signal) Wait() {
	s.lock.Lock()
	defer s.lock.Unlock()
	for !s.signalled {
		s.cond.Wait()
	}
}
