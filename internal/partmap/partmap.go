// Package partmap provide a partitioned map.
package partmap

import (
	"hash/maphash"

	"github.com/go-ricrob/simplesolver/internal/packed"
	"github.com/go-ricrob/simplesolver/internal/spinlock"
)

type part[P packed.Packable] struct {
	mu             spinlock.Mutex
	m              map[P]P // to/from map
	source, target []P
}

// Map represent the partitioned map.
type Map[P packed.Packable] struct {
	numPart uint64
	seed    maphash.Seed
	parts   []*part[P]
}

// New creates a new partitioned map instance.
func New[P packed.Packable](startState P, numPart uint64) *Map[P] {
	pm := &Map[P]{
		numPart: numPart,
		seed:    maphash.MakeSeed(),
		parts:   make([]*part[P], numPart),
	}
	for i := range pm.parts {
		pm.parts[i] = &part[P]{m: make(map[P]P, 10000)}
	}
	// store start state
	var initState P // initial p
	part := pm.parts[startState.Hash(pm.seed)%pm.numPart]
	part.m[startState] = initState
	part.source = append(part.source, startState)
	return pm
}

// Load reads a map value.
func (pm *Map[P]) Load(k P) (P, bool) {
	part := pm.parts[k.Hash(pm.seed)%pm.numPart]
	part.mu.Lock()
	v, ok := part.m[k]
	part.mu.Unlock()
	return v, ok
}

// StoreTarget writes a map value if not existent and adds the key to the target list.
func (pm *Map[P]) StoreTarget(k, v P) bool {
	part := pm.parts[k.Hash(pm.seed)%pm.numPart]
	part.mu.Lock()
	if _, ok := part.m[k]; !ok {
		part.m[k] = v
		part.target = append(part.target, k)
		part.mu.Unlock()
		return true
	}
	part.mu.Unlock()
	return false
}

// Len returns the number of entries in the map.
func (pm *Map[P]) Len() int {
	size := 0
	for _, part := range pm.parts {
		size += len(part.m)
	}
	return size
}

// NumPart returns the number of partitions.
func (pm *Map[P]) NumPart() int { return int(pm.numPart) }

// Source returns the source list for partition idx.
func (pm *Map[P]) Source(idx int) []P { return pm.parts[idx].source }

// Swap swaps the source list with the target list.
func (pm *Map[P]) Swap() {
	for _, part := range pm.parts {
		part.source, part.target = part.target, part.source[:0]
	}
}
