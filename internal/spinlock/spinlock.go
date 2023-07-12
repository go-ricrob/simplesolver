// Package spinlock provides a spinlock mutex.
package spinlock

import (
	"runtime"
	"sync/atomic"
)

// Mutex represents a spinlock.
type Mutex struct {
	state atomic.Int32 // should be a little bit faster like atomic.Bool
}

// Lock locks the mutex busy waiting (spinlock).
func (m *Mutex) Lock() {
	for !m.state.CompareAndSwap(0, 1) {
		runtime.Gosched()
	}
}

// Unlock unlocks the mutex.
func (m *Mutex) Unlock() { m.state.Store(0) }
