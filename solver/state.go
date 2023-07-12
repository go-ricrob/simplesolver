package solver

import (
	"errors"
	"fmt"
	"log"

	"github.com/go-ricrob/exec/task"
	. "github.com/go-ricrob/game/types"
	"github.com/go-ricrob/simplesolver/internal/packed"
	"github.com/go-ricrob/simplesolver/internal/spinlock"
	"golang.org/x/exp/slices"
)

type Resulter interface {
	Moves() (task.Moves, error)
	NumCalcMove() int
}

var (
	_ Resulter = (*states[packed.P4])(nil)
	_ Resulter = (*states[packed.P5])(nil)
)

var errInconsistentState = errors.New("inconsistent state")

type state[P packed.Packable] struct {
	from, to   P
	idx        int // robot index
	isSolution bool
	found      bool
}

type states[P packed.Packable] struct {
	mu             spinlock.Mutex
	m              map[P]P // to/from map
	source, target []P
	hasSolution    bool
	solutionTo     P // solution to value
}

func newStates[P packed.Packable](start P) *states[P] {
	var pInit P // initial p
	return &states[P]{m: map[P]P{start: pInit}, source: []P{start}}
}

func (m *states[P]) hasTurned(from, to P, idx int) bool {
	var pInit P
	var numHorizontal, numVertical int

	addAxis := func(from, to P, idx int) {
		x1, y1 := packed.UnpackIdx(from, idx)
		x2, y2 := packed.UnpackIdx(to, idx)
		if x1 != x2 {
			numHorizontal++
		}
		if y1 != y2 {
			numVertical++
		}
	}

	for {
		addAxis(from, to, idx)
		if numHorizontal > 0 && numVertical > 0 {
			return true
		}

		var ok bool
		to = from
		from, ok = m.m[to]
		if !ok {
			panic("should never happen")
		}
		if from == pInit { // first move found
			return false
		}
	}
}

func (m *states[P]) add(state *state[P]) {
	m.mu.Lock()
	if state.isSolution {
		m.hasSolution = true
		m.solutionTo = state.to
		m.m[state.to] = state.from
		// check if target robot did turn 90° at least once
		if !m.hasTurned(state.from, state.to, state.idx) {
			// sorry, no solution...
			log.Println("no solution")
			state.isSolution = false
		}
		m.mu.Unlock()
		return
	}

	if _, ok := m.m[state.to]; ok {
		state.found = true
		m.mu.Unlock()
		return
	}

	state.found = false
	m.m[state.to] = state.from
	m.target = append(m.target, state.to)
	m.mu.Unlock()
}

func (m *states[P]) moveIdx(from, to P) (int, error) {
	for i := 0; i < len(from); i++ {
		if from[i] != to[i] {
			return i, nil
		}
	}
	return 0, errInconsistentState
}

// Moves returns the solver result.
func (m *states[P]) Moves() (task.Moves, error) {
	if !m.hasSolution {
		return nil, fmt.Errorf("no solution found")
	}

	var pInit P
	moves := task.Moves{}
	to := m.solutionTo
	for {
		from, ok := m.m[to]
		if !ok {
			panic("should never happen")
		}
		if from == pInit { // first move found
			return moves, nil
		}
		idx, err := m.moveIdx(from, to)
		if err != nil {
			return nil, err
		}
		x, y := packed.UnpackIdx(to, idx)
		moves = slices.Insert(moves, 0, &task.Move{To: task.Coordinate{X: x, Y: y}, Color: convertColorOut(Colors[idx])})
		to = from
	}
}

// NumCalcMove returns the number of calculated moves.
func (m *states[P]) NumCalcMove() int { return len(m.m) }
