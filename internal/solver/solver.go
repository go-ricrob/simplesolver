// Package solver implements a simple solver.
package solver

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/go-ricrob/exec/task"
	"github.com/go-ricrob/game/board"
	"github.com/go-ricrob/game/coord"
	"github.com/go-ricrob/game/types"
	"github.com/go-ricrob/simplesolver/internal/packed"
	"github.com/go-ricrob/simplesolver/internal/partmap"
	"golang.org/x/exp/slices"
)

const (
	numCh    = 1000
	maxLevel = 21
)

var numWorker = runtime.NumCPU()

var robotColors = []types.Color{types.Yellow, types.Red, types.Green, types.Blue, types.Silver}

type nextReaderLevel[P packed.Packable] struct {
	wg       *sync.WaitGroup
	workerCh chan<- P
}

type nextWriterLevel[P packed.Packable] struct {
	wg       *sync.WaitGroup
	workerCh <-chan P
	level    int
}

// Result represents a solver result.
type Result struct {
	Moves       task.Moves
	NumCalcMove int
}

// Solver defines a solver interface.
type Solver interface {
	Run() *Result
}

var (
	_ Solver = (*solver[packed.P4])(nil)
	_ Solver = (*solver[packed.P5])(nil)
)

var errInconsistentState = errors.New("inconsistent state")

type solver[P packed.Packable] struct {
	task         *task.Task
	board        *board.Board
	targetSymbol board.Symbol
	targetColor  types.Color
	targetCoord  byte
	targetRobot  int

	minMoves [board.NumField]int

	solutionCh  chan struct{}
	pm          *partmap.Map[P]
	hasSolution atomic.Bool
	solutionTo  P // solution to value

	moveFn [types.NumDir]func(p P, moveRobot int) (byte, bool)
}

// New creates a new solver instance.
func New[P packed.Packable](task *task.Task, useSilverRobot bool) Solver {
	board := board.New([board.NumTile]string{
		board.TopLeft:     task.Args.TopLeftTile,
		board.TopRight:    task.Args.TopRightTile,
		board.BottomLeft:  task.Args.BottomLeftTile,
		board.BottomRight: task.Args.BottomRightTile,
	})

	//TODO: how to guarantee robot order?
	robots := []byte{
		coord.Ctob(task.Args.YellowRobot.X, task.Args.YellowRobot.Y),
		coord.Ctob(task.Args.RedRobot.X, task.Args.RedRobot.Y),
		coord.Ctob(task.Args.GreenRobot.X, task.Args.GreenRobot.Y),
		coord.Ctob(task.Args.BlueRobot.X, task.Args.BlueRobot.Y),
	}
	if useSilverRobot {
		robots = append(robots, coord.Ctob(task.Args.SilverRobot.X, task.Args.SilverRobot.X))
	}

	targetSymbol, targetColor := convertSymbolIn(task.Args.TargetSymbol)
	targetCoord := board.TargetCoord(targetSymbol, targetColor)

	var targetRobot int
	// determine target robot index
	for i, color := range robotColors {
		if color == targetColor {
			targetRobot = i
			break
		}
	}

	s := &solver[P]{
		task:         task,
		board:        board,
		targetSymbol: targetSymbol,
		targetColor:  targetColor,
		targetCoord:  targetCoord,
		targetRobot:  targetRobot,
		minMoves:     board.MinMoves(targetCoord),
		solutionCh:   make(chan struct{}),
		pm:           partmap.New[P](packed.SetRobots[P](robots), 10000),
	}

	s.moveFn[types.North] = s.moveNorth
	s.moveFn[types.East] = s.moveEast
	s.moveFn[types.South] = s.moveSouth
	s.moveFn[types.West] = s.moveWest

	return s
}

func (s *solver[P]) hasTurned(from, to P, robot int) bool {
	var pInit P
	var numHorizontal, numVertical int

	addAxis := func(from, to P, robot int) {
		x1, y1 := coord.Btoc(from[robot])
		x2, y2 := coord.Btoc(to[robot])
		if x1 != x2 {
			numHorizontal++
		}
		if y1 != y2 {
			numVertical++
		}
	}

	for {
		addAxis(from, to, robot)
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

func (s *solver[P]) moveNorth(p P, moveRobot int) (byte, bool) {
	// handle redirects here
	// field needs to handle redirection as a boarder
	// routes needs to deliver redirect field coords
	c := p[moveRobot]
	x0, y0 := coord.Btoc(c)
	yt := coord.Y(s.board.Fields[c].Targets[types.North])
	for robot := 0; robot < len(p); robot++ {
		if robot != moveRobot {
			x, y := coord.Btoc(p[robot])
			if x == x0 && y > y0 && y <= yt {
				yt = y - 1
			}
		}
	}
	if yt == y0 {
		return p[moveRobot], false
	}
	return coord.Ctob(x0, yt), true
}

func (s *solver[P]) moveEast(p P, moveRobot int) (byte, bool) {
	// handle redirects here
	// field needs to handle redirection as a boarder
	// routes needs to deliver redirect field coords
	c := p[moveRobot]
	x0, y0 := coord.Btoc(c)
	xt := coord.X(s.board.Fields[c].Targets[types.East])
	for robot := 0; robot < len(p); robot++ {
		if robot != moveRobot {
			x, y := coord.Btoc(p[robot])
			if y == y0 && x > x0 && x <= xt {
				xt = x - 1
			}
		}
	}
	if xt == x0 {
		return p[moveRobot], false
	}
	return coord.Ctob(xt, y0), true
}

func (s *solver[P]) moveSouth(p P, moveRobot int) (byte, bool) {
	// handle redirects here
	// field needs to handle redirection as a boarder
	// routes needs to deliver redirect field coords
	c := p[moveRobot]
	x0, y0 := coord.Btoc(c)
	yt := coord.Y(s.board.Fields[c].Targets[types.South])
	for robot := 0; robot < len(p); robot++ {
		if robot != moveRobot {
			x, y := coord.Btoc(p[robot])
			if x == x0 && y < y0 && y >= yt {
				yt = y + 1
			}
		}
	}
	if yt == y0 {
		return p[moveRobot], false
	}
	return coord.Ctob(x0, yt), true
}

func (s *solver[P]) moveWest(p P, moveRobot int) (byte, bool) {
	// handle redirects here
	// field needs to handle redirection as a boarder
	// routes needs to deliver redirect field coords
	c := p[moveRobot]
	x0, y0 := coord.Btoc(c)
	xt := coord.X(s.board.Fields[c].Targets[types.West])
	for robot := 0; robot < len(p); robot++ {
		if robot != moveRobot {
			x, y := coord.Btoc(p[robot])
			if y == y0 && x < x0 && x >= xt {
				xt = x + 1
			}
		}
	}
	if xt == x0 {
		return p[moveRobot], false
	}
	return coord.Ctob(xt, y0), true
}

func (s *solver[P]) checkMinMoveCosmic(p P, remMoves int) bool {
	for robot := 0; robot < len(p); robot++ {
		if s.minMoves[p[robot]] <= remMoves {
			return true
		}
	}
	return false
}

func (s *solver[P]) checkMinMove(p P, remMoves int) bool {
	return s.minMoves[p[s.targetRobot]] <= remMoves
}

func (s *solver[P]) writer(wg *sync.WaitGroup, nextLevelCh <-chan *nextWriterLevel[P], checkMinMove func(p P, remMoves int) bool) {
	defer wg.Done()

	for nextLevel := range nextLevelCh {
		remMoves := maxLevel - nextLevel.level
		//log.Printf("max level %d this level %d rem moves %d", maxLevel, nextLevel.level, remMoves)
		for from := range nextLevel.workerCh {
			if checkMinMove(from, remMoves) {
				for robot := 0; robot < len(from); robot++ {
					for dir := types.Dir(0); dir < types.NumDir; dir++ {
						if pos, ok := s.moveFn[dir](from, robot); ok {
							to := packed.SetRobot(from, robot, byte(pos))
							if s.pm.StoreTarget(to, from) && (to[robot] == s.targetCoord) && (s.targetSymbol == board.Cosmic || robotColors[robot] == s.targetColor) {
								// check if target robot did turn 90° at least once
								if s.hasTurned(from, to, robot) {
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
		}
		nextLevel.wg.Done()
	}
}

func (s *solver[P]) moves() task.Moves {
	robotMoved := func(from, to P) int {
		for robot := 0; robot < len(from); robot++ {
			if from[robot] != to[robot] {
				return robot
			}
		}
		panic(errInconsistentState) // should never happen

	}

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
		moveRobot := robotMoved(from, to)
		x, y := coord.Btoc(to[moveRobot])
		moves = slices.Insert(moves, 0, &task.Move{To: task.Coordinate{X: x, Y: y}, Robot: convertRobotOut(robotColors[moveRobot])})
		to = from
	}
}

// Run starts the solving algorithm.
func (s *solver[P]) Run() *Result {

	//log.Printf("min moves %v", s.minMoves)

	// spin up workers
	workerWg := new(sync.WaitGroup)
	workerWg.Add(2 * numWorker)

	var checkMinMove func(p P, remMoves int) bool
	if s.targetSymbol == board.Cosmic {
		checkMinMove = s.checkMinMoveCosmic
	} else {
		checkMinMove = s.checkMinMove
	}

	nextReaderLevelChs := make([]chan *nextReaderLevel[P], numWorker)
	nextWriterLevelChs := make([]chan *nextWriterLevel[P], numWorker)
	for i := 0; i < numWorker; i++ {
		nextReaderLevelChs[i] = make(chan *nextReaderLevel[P], numCh)
		go s.reader(i, workerWg, nextReaderLevelChs[i])

		nextWriterLevelChs[i] = make(chan *nextWriterLevel[P], numCh)
		go s.writer(workerWg, nextWriterLevelChs[i], checkMinMove)
	}

	for level := 0; level < maxLevel; level++ {
		s.task.Level(level)

		nextReaderLevelWg := new(sync.WaitGroup)
		nextReaderLevelWg.Add(numWorker)

		nextWriterLevelWg := new(sync.WaitGroup)
		nextWriterLevelWg.Add(numWorker)

		workerChs := make([]chan P, numWorker)

		for i := 0; i < numWorker; i++ {
			workerChs[i] = make(chan P, numCh)
			nextWriterLevelChs[i] <- &nextWriterLevel[P]{wg: nextWriterLevelWg, workerCh: workerChs[i], level: level}
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
