package solver

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/go-ricrob/exec/task"
	"github.com/go-ricrob/game/board"
	. "github.com/go-ricrob/game/types"
	"github.com/go-ricrob/simplesolver/internal/packed"
	"github.com/go-ricrob/simplesolver/internal/partmap"
	"golang.org/x/exp/slices"
)

const numCh = 1000

var numWorker = runtime.NumCPU()

type nextReaderLevel[P packed.Packable] struct {
	wg       *sync.WaitGroup
	workerCh chan<- P
}

type nextWriterLevel[P packed.Packable] struct {
	wg       *sync.WaitGroup
	workerCh <-chan P
}

type Result struct {
	Moves       task.Moves
	NumCalcMove int
}

type Solver interface {
	Run() *Result
}

var (
	_ Solver = (*solver[packed.P4])(nil)
	_ Solver = (*solver[packed.P5])(nil)
)

var errInconsistentState = errors.New("inconsistent state")

type solver[P packed.Packable] struct {
	task        *task.Task
	board       *board.Board
	targetColor Color
	targetCoord byte

	solutionCh  chan struct{}
	pm          *partmap.Map[P]
	hasSolution atomic.Bool
	solutionTo  P // solution to value
}

func New[P packed.Packable](task *task.Task, useSilverRobot bool) Solver {
	board := board.New(map[board.TilePosition]string{
		board.TopLeft:     task.Args.TopLeftTile,
		board.TopRight:    task.Args.TopRightTile,
		board.BottomLeft:  task.Args.BottomLeftTile,
		board.BottomRight: task.Args.BottomRightTile,
	})

	robots := map[Color]Coordinate{
		Yellow: Coordinate(task.Args.YellowRobot),
		Red:    Coordinate(task.Args.RedRobot),
		Green:  Coordinate(task.Args.GreenRobot),
		Blue:   Coordinate(task.Args.BlueRobot),
	}
	if useSilverRobot {
		robots[Silver] = Coordinate(task.Args.SilverRobot)
	}

	targetSymbol := convertSymbolIn(task.Args.TargetSymbol)
	targetColor := convertColorIn(task.Args.TargetColor)
	x, y := board.TargetCoordinate(targetSymbol, targetColor)

	return &solver[P]{
		task:        task,
		board:       board,
		targetColor: targetColor,
		targetCoord: byte(x<<4) | byte(y),
		solutionCh:  make(chan struct{}),
		pm:          partmap.New[P](packed.Pack[P](robots), 10000),
	}
}

func (s *solver[P]) hasTurned(from, to P, idx int) bool {
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
		from, ok = s.pm.Load(to)
		if !ok {
			panic("should never happen")
		}
		if from == pInit { // first move found
			return false
		}
	}
}

func (s *solver[P]) reader(idx int, wg *sync.WaitGroup, nextLevelCh <-chan *nextReaderLevel[P]) {
	defer wg.Done()

	numPart := s.pm.NumPart()

	for nextLevel := range nextLevelCh {
		for j := idx; j < numPart; j += numWorker {
			for _, p := range s.pm.Source(j) {
				select {
				case <-s.solutionCh:
					goto done
				case nextLevel.workerCh <- p:
				}
			}
		}
	done:
		nextLevel.wg.Done()
	}
}

func (s *solver[P]) writer(wg *sync.WaitGroup, nextLevelCh <-chan *nextWriterLevel[P]) {
	defer wg.Done()

	robots := map[Color]Coordinate{}

	for nextLevel := range nextLevelCh {
		for from := range nextLevel.workerCh {
			packed.Unpack(from, robots)

			for idx := 0; idx < len(from); idx++ {
				color := Colors[idx]
				for _, direction := range board.Directions {
					if x, y, ok := s.board.Move(robots, color, direction); ok {
						to := packed.PackIdx(from, idx, x, y)
						if s.pm.StoreTarget(to, from) && (to[idx] == s.targetCoord) && (color&s.targetColor != 0) {
							// check if target robot did turn 90° at least once
							if s.hasTurned(from, to, idx) {
								if s.hasSolution.CompareAndSwap(false, true) {
									s.solutionTo = to
									close(s.solutionCh)
								}
							}
						}
					}
				}
			}
		}
		nextLevel.wg.Done()
	}
}

func (s *solver[P]) moveIdx(from, to P) int {
	for i := 0; i < len(from); i++ {
		if from[i] != to[i] {
			return i
		}
	}
	panic(errInconsistentState) // should never happen
}

func (s *solver[P]) moves() task.Moves {
	if !s.hasSolution.Load() {
		return nil
	}

	var pInit P
	moves := task.Moves{}
	to := s.solutionTo
	for {
		from, ok := s.pm.Load(to)
		if !ok {
			panic("should never happen")
		}
		if from == pInit { // first move found
			return moves
		}
		idx := s.moveIdx(from, to)
		x, y := packed.UnpackIdx(to, idx)
		moves = slices.Insert(moves, 0, &task.Move{To: task.Coordinate{X: x, Y: y}, Color: convertColorOut(Colors[idx])})
		to = from
	}
}

// Run starts the solving algorithm.
func (s *solver[P]) Run() *Result {
	// spin up workers
	workerWg := new(sync.WaitGroup)
	workerWg.Add(2 * numWorker)

	nextReaderLevelChs := make([]chan *nextReaderLevel[P], numWorker)
	nextWriterLevelChs := make([]chan *nextWriterLevel[P], numWorker)
	for i := 0; i < numWorker; i++ {
		nextReaderLevelChs[i] = make(chan *nextReaderLevel[P], numCh)
		go s.reader(i, workerWg, nextReaderLevelChs[i])

		nextWriterLevelChs[i] = make(chan *nextWriterLevel[P], numCh)
		go s.writer(workerWg, nextWriterLevelChs[i])
	}

	for level := 0; ; level++ {
		s.task.Level(level)

		nextReaderLevelWg := new(sync.WaitGroup)
		nextReaderLevelWg.Add(numWorker)

		nextWriterLevelWg := new(sync.WaitGroup)
		nextWriterLevelWg.Add(numWorker)

		workerChs := make([]chan P, numWorker)

		for i := 0; i < numWorker; i++ {
			workerChs[i] = make(chan P, numCh)
			nextWriterLevelChs[i] <- &nextWriterLevel[P]{wg: nextWriterLevelWg, workerCh: workerChs[i]}
			nextReaderLevelChs[i] <- &nextReaderLevel[P]{wg: nextReaderLevelWg, workerCh: workerChs[i]}
		}

		// wait for readers to be finalized
		nextReaderLevelWg.Wait()

		// wait for writers to be finalized
		for _, workerCh := range workerChs {
			close(workerCh)
		}
		nextWriterLevelWg.Wait()

		if s.hasSolution.Load() {
			break
		}

		s.pm.Swap()
	}

	for i := 0; i < numWorker; i++ {
		close(nextReaderLevelChs[i])
		close(nextWriterLevelChs[i])
	}
	workerWg.Wait()

	return &Result{
		Moves:       s.moves(),
		NumCalcMove: s.pm.Len(),
	}
}
