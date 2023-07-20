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

const numCh = 100000

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
	workerWg.Add(numWorker * 2)

	nextLevelReaderChs := make([]chan *nextReaderLevel[P], numWorker)
	nextLevelWriterChs := make([]chan *nextWriterLevel[P], numWorker)
	for i := 0; i < numWorker; i++ {
		nextLevelReaderChs[i] = make(chan *nextReaderLevel[P], numCh)
		go s.reader(i, states, workerWg, nextLevelReaderChs[i])

		nextLevelWriterChs[i] = make(chan *nextWriterLevel[P], numCh)
		go s.writer(states, workerWg, nextLevelWriterChs[i])
	}

	for level := 0; ; level++ {
		nextLevelReaderWg := new(sync.WaitGroup)
		nextLevelReaderWg.Add(numWorker)

		nextLevelWriterWg := new(sync.WaitGroup)
		nextLevelWriterWg.Add(numWorker)

		workerChs := make([]chan P, numWorker)

		for i := 0; i < numWorker; i++ {
			workerChs[i] = make(chan P, numCh)
			nextLevelWriterChs[i] <- &nextWriterLevel[P]{wg: nextLevelWriterWg, workerCh: workerChs[i]}
			nextLevelReaderChs[i] <- &nextReaderLevel[P]{wg: nextLevelReaderWg, workerCh: workerChs[i]}
		}

		nextLevelReaderWg.Wait()

		// wait for level to be finalized
		for _, workerCh := range workerChs {
			close(workerCh)
		}
		nextLevelWriterWg.Wait()

		if states.hasSolution.Load() {
			break
		}

		states.pm.SwapTargets()
		s.task.IncrProgress(5)
	}

	for i := 0; i < numWorker; i++ {
		close(nextLevelReaderChs[i])
		close(nextLevelWriterChs[i])
	}
	workerWg.Wait()

	s.task.SetProgress(100)
	return states
}