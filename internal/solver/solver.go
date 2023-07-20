// Package solver implemets a solver algorithm.
package solver

import (
	"runtime"
	"sync"

	"github.com/go-ricrob/exec/task"
	"github.com/go-ricrob/game/board"
	. "github.com/go-ricrob/game/types"
	"github.com/go-ricrob/simplesolver/internal/packed"
)

const numCh = 1000

var numWorker = runtime.NumCPU()

type Runner interface {
	Run() Resulter
}

var (
	_ Runner = (*solver[packed.P4])(nil)
	_ Runner = (*solver[packed.P5])(nil)
)

type solver[P packed.Packable] struct {
	task        *task.Task
	board       *board.Board
	start       P
	targetColor Color
	targetCoord byte
}

func New[P packed.Packable](task *task.Task, useSilverRobot bool) Runner {
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
		start:       packed.Pack[P](robots),
		targetColor: targetColor,
		targetCoord: byte(x<<4) | byte(y),
	}
}

type nextReaderLevel[P packed.Packable] struct {
	wg       *sync.WaitGroup
	workerCh chan<- P
}

type nextWriterLevel[P packed.Packable] struct {
	wg       *sync.WaitGroup
	workerCh <-chan P
}

func (s *solver[P]) writer(states *states[P], wg *sync.WaitGroup, nextLevelCh <-chan *nextWriterLevel[P]) {
	defer wg.Done()

	robots := map[Color]Coordinate{}
	state := new(state[P])

	for nextLevel := range nextLevelCh {
		for p := range nextLevel.workerCh {
			packed.Unpack(p, robots)

			for idx := 0; idx < len(p); idx++ {
				color := Colors[idx]
				for _, move := range s.board.Moves {
					if x, y, ok := move(robots, color); ok {
						state.from = p
						state.to = packed.PackIdx(p, idx, x, y)
						state.idx = idx
						states.add(state)
					}
				}
			}
		}
		nextLevel.wg.Done()
	}
}

func (s *solver[P]) reader(idx int, states *states[P], wg *sync.WaitGroup, nextLevelCh <-chan *nextReaderLevel[P]) {
	defer wg.Done()

	numPart := states.pm.NumPart()

	for nextLevel := range nextLevelCh {
		for j := idx; j < numPart; j += numWorker {
			for _, p := range states.pm.Source(j) {
				select {
				case <-states.solutionCh:
					goto done
				case nextLevel.workerCh <- p:
				}
			}
		}
	done:
		nextLevel.wg.Done()
	}
}

func (s *solver[P]) Run() Resulter {

	states := newStates[P](s.start, s.targetColor, s.targetCoord)

	// spin up workers
	workerWg := new(sync.WaitGroup)
	workerWg.Add(2 * numWorker)

	nextReaderLevelChs := make([]chan *nextReaderLevel[P], numWorker)
	nextWriterLevelChs := make([]chan *nextWriterLevel[P], numWorker)
	for i := 0; i < numWorker; i++ {
		nextReaderLevelChs[i] = make(chan *nextReaderLevel[P], numCh)
		go s.reader(i, states, workerWg, nextReaderLevelChs[i])

		nextWriterLevelChs[i] = make(chan *nextWriterLevel[P], numCh)
		go s.writer(states, workerWg, nextWriterLevelChs[i])
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

		if states.hasSolution.Load() {
			break
		}

		states.pm.Swap()
	}

	for i := 0; i < numWorker; i++ {
		close(nextReaderLevelChs[i])
		close(nextWriterLevelChs[i])
	}
	workerWg.Wait()

	return states
}
