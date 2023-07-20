package solver

import (
	"github.com/go-ricrob/exec/task"
	"github.com/go-ricrob/game/board"
	. "github.com/go-ricrob/game/types"
)

var convertSymbolInMap = map[task.Symbol]board.Symbol{
	task.Pyramid: board.Pyramid,
	task.Star:    board.Star,
	task.Moon:    board.Moon,
	task.Saturn:  board.Saturn,
	task.Cosmic:  board.Cosmic,
}

func convertSymbolIn(ts task.Symbol) board.Symbol {
	s, ok := convertSymbolInMap[ts]
	if !ok {
		return board.NoSymbol
	}
	return s
}

var convertColorInMap = map[task.Color]Color{
	task.Yellow: Yellow,
	task.Red:    Red,
	task.Green:  Green,
	task.Blue:   Blue,
}

var convertColorOutMap = map[Color]task.Color{}

func init() {
	for tc, c := range convertColorInMap {
		convertColorOutMap[c] = tc
	}
}

func convertColorIn(tc task.Color) Color {
	c, ok := convertColorInMap[tc]
	if !ok {
		return CosmicColor
	}
	return c
}

func convertColorOut(c Color) task.Color {
	tc, ok := convertColorOutMap[c]
	if !ok {
		return 0
	}
	return tc
}
