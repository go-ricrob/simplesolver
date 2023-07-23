// Package packed provides types and functions for memory efficient representations of robots.
package packed

import (
	"hash/maphash"
)

// Packable interface defines P4 and P5 contraints as packable data types.
type Packable interface {
	P4 | P5
	Hash(seed maphash.Seed) uint64
}

// P4 is a compressed representation of 4 robots.
type P4 [4]byte

// Hash returns a hash value of P4.
func (p P4) Hash(seed maphash.Seed) uint64 { return maphash.Bytes(seed, p[:]) }

// P5 is a compressed representation of 5 robots.
type P5 [5]byte

// Hash returns a hash value of P5.
func (p P5) Hash(seed maphash.Seed) uint64 { return maphash.Bytes(seed, p[:]) }

// SetRobot sets one byte at index robot into p and returns the result.
func SetRobot[P Packable](p P, robot int, b byte) P { p[robot] = b; return p }

// SetRobots returns a packed representation of robots.
func SetRobots[P Packable](robots []byte) P {
	var p P
	for robot := 0; robot < len(p); robot++ {
		p[robot] = robots[robot]
	}
	return p
}
