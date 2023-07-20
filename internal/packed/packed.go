// Package packed provides types and functions for memory efficient representations of robots.
package packed

import (
	"hash/maphash"

	"github.com/go-ricrob/game/types"
)

// Packable interface defines P4 and P5 contraints as packable data types.
type Packable interface {
	P4 | P5
	MapHash(seed maphash.Seed) uint64
}

// P4 is a compressed representation of 4 robots.
type P4 [4]byte

func (p P4) MapHash(seed maphash.Seed) uint64 { return maphash.Bytes(seed, p[:]) }

// P5 is a compressed representation of 5 robots.
type P5 [5]byte

func (p P5) MapHash(seed maphash.Seed) uint64 { return maphash.Bytes(seed, p[:]) }

// PackIdx packs one byte at index idx into p and returns the result.
func PackIdx[P Packable](p P, idx, x, y int) P { p[idx] = byte(x)<<4 | byte(y); return p }

// UnpackIdx unpacks one byte at index idx and returns x and y coordinates.
func UnpackIdx[P Packable](p P, idx int) (x, y int) { return int(p[idx] >> 4), int(p[idx] & 0x0f) }

// Pack returns a packed representation of robots.
func Pack[P Packable](robots map[types.Color]types.Coordinate) P {
	var p P
	for i := 0; i < len(p); i++ {
		coord := robots[types.Colors[i]]
		p[i] = byte(coord.X)<<4 | byte(coord.Y)
	}
	return p
}

// Unpack unpacks the packed representation of p into robots.
func Unpack[P Packable](p P, robots map[types.Color]types.Coordinate) {
	for i := 0; i < len(p); i++ {
		robots[types.Colors[i]] = types.Coordinate{X: int(p[i] >> 4), Y: int(p[i] & 0x0f)}
	}
}
