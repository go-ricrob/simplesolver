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

type nextLevel[P packed.Packable] struct {
	wg       *sync.WaitGroup
	workerCh <-chan P
}

func (s *solver[P]) worker(no int, states *states[P], wg *sync.WaitGroup, nextLevelCh <-chan *nextLevel[P], solutionCh chan<- struct{}) {
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
						state.isSolution = (state.to[idx] == s.targetCoord) && (color&s.targetColor != 0)
						states.add(state)
						if state.isSolution { // could be changed by stateMap
							solutionCh <- struct{}{}
						}
					}
				}
			}
		}
		nextLevel.wg.Done()
	}
}
func (s *solver[P]) Run() Resulter {
	states := newStates[P](s.start)
	// spin up workers
	workerWg := new(sync.WaitGroup)
	workerWg.Add(numWorker)
	solutionCh := make(chan struct{}, numWorker) // enough space for all workers

	var nextLevelChs []chan *nextLevel[P]
	for i := 0; i < numWorker; i++ {
		nextLevelCh := make(chan *nextLevel[P], numCh)
		go s.worker(i, states, workerWg, nextLevelCh, solutionCh)
		nextLevelChs = append(nextLevelChs, nextLevelCh)
	}

	for level := 0; ; level++ {
		workerCh := make(chan P, numCh)
		nextLevelWg := new(sync.WaitGroup)
		nextLevelWg.Add(numWorker)
		nextLevel := &nextLevel[P]{wg: nextLevelWg, workerCh: workerCh}
		for _, nextLevelCh := range nextLevelChs {
			nextLevelCh <- nextLevel
		}

		for _, p := range states.source {
			select {
			case <-solutionCh:
				goto done
			case workerCh <- p:
			}
		}
	done:
		// wait for level to be finalized
		close(workerCh)
		nextLevelWg.Wait()

		if states.hasSolution {
			break
		}

		states.source, states.target = states.target, nil
		s.task.IncrProgress(5)
	}

	for _, nextLevelCh := range nextLevelChs {
		close(nextLevelCh)
	}
	workerWg.Wait()

	s.task.SetProgress(100)
	return states
}
