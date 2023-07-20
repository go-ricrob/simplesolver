// Package partmap provide a partitioned map.
package partmap

import (
	"hash/maphash"
	"sync"

	"github.com/go-ricrob/simplesolver/internal/packed"
)

type part[P packed.Packable] struct {
	mu             sync.Mutex
	m              map[P]P // to/from map
	source, target []P
}

type Map[P packed.Packable] struct {
	numPart uint64
	seed    maphash.Seed
	parts   []*part[P]
}

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
	part := pm.parts[startState.MapHash(pm.seed)%pm.numPart]
	part.m[startState] = initState
	part.source = append(part.source, startState)
	return pm
}

func (pm *Map[P]) Load(k P) (P, bool) {
	part := pm.parts[k.MapHash(pm.seed)%pm.numPart]
	part.mu.Lock()
	v, ok := part.m[k]
	part.mu.Unlock()
	return v, ok
}

func (pm *Map[P]) StoreTarget(k, v P) bool {
	part := pm.parts[k.MapHash(pm.seed)%pm.numPart]
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

func (pm *Map[P]) Size() int {
	size := 0
	for _, part := range pm.parts {
		size += len(part.m)
	}
	return size
}

func (pm *Map[P]) NumPart() int { return int(pm.numPart) }

func (pm *Map[P]) Source(idx int) []P { return pm.parts[idx].source }

func (pm *Map[P]) SwapTargets() {
	for _, part := range pm.parts {
		part.source, part.target = part.target, nil
	}
}
