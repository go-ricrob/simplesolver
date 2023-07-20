package solver

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/go-ricrob/exec/task"
	. "github.com/go-ricrob/game/types"
	"github.com/go-ricrob/simplesolver/internal/packed"
	"github.com/go-ricrob/simplesolver/internal/partmap"
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
	from, to P
	idx      int // robot index
}

type states[P packed.Packable] struct {
	solutionCh  chan struct{}
	pm          *partmap.Map[P]
	hasSolution atomic.Bool
	solutionTo  P // solution to value
	targetColor Color
	targetCoord byte
}

func newStates[P packed.Packable](startState P, targetColor Color, targetCoord byte) *states[P] {
	return &states[P]{
		solutionCh:  make(chan struct{}),
		targetColor: targetColor,
		targetCoord: targetCoord,
		pm:          partmap.New[P](startState, 10000)}
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
		from, ok = m.pm.Load(to)
		if !ok {
			panic("should never happen")
		}
		if from == pInit { // first move found
			return false
		}
	}
}

func (m *states[P]) add(state *state[P]) {
	idx := state.idx
	color := Colors[idx]

	if m.pm.StoreTarget(state.to, state.from) && (state.to[idx] == m.targetCoord) && (color&m.targetColor != 0) {
		// check if target robot did turn 90° at least once
		if m.hasTurned(state.from, state.to, state.idx) {
			if m.hasSolution.CompareAndSwap(false, true) {
				m.solutionTo = state.to
				close(m.solutionCh)
			}
		}
	}
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
	if !m.hasSolution.Load() {
		return nil, fmt.Errorf("no solution found")
	}

	var pInit P
	moves := task.Moves{}
	to := m.solutionTo
	for {
		from, ok := m.pm.Load(to)
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
func (m *states[P]) NumCalcMove() int { return m.pm.Size() }
