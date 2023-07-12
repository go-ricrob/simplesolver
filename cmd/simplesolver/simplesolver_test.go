package main

import (
	"testing"

	"github.com/go-ricrob/exec/task"
)

/*
	complexBoard = map[types.TilePosition]string{
			types.TopLeft:      {SetID: 'A', TileNo: 3, Front: true},
			common.TopRight:    {SetID: 'A', TileNo: 1, Front: false},
			common.BottomRight: {SetID: 'A', TileNo: 4, Front: true},
			common.BottomLeft:  {SetID: 'A', TileNo: 2, Front: false},
		},
			map[common.Color]common.Coordinate{
				common.Yellow: {X: 15, Y: 0},
				common.Red:    {X: 14, Y: 2},
				common.Green:  {X: 1, Y: 13},
				common.Blue:   {X: 13, Y: 11},
			},
			common.Targets[common.TnBluePyramid],
*/
func TestSolver(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long running solves")
	}

	tests := []struct {
		flag int
		args *task.Args
	}{
		{
			task.NoSymbolCheck,
			&task.Args{
				TopLeftTile:     "A3F",
				TopRightTile:    "A1B",
				BottomLeftTile:  "A2B",
				BottomRightTile: "A4F",

				YellowRobot: task.Coordinate{X: 15, Y: 0},
				RedRobot:    task.Coordinate{X: 14, Y: 2},
				GreenRobot:  task.Coordinate{X: 1, Y: 13},
				BlueRobot:   task.Coordinate{X: 13, Y: 11},
				SilverRobot: task.Coordinate{X: -1, Y: -1},

				TargetSymbol: task.Pyramid,
				TargetColor:  task.Blue,
			},
		},
	}

	for _, test := range tests {
		if err := test.args.Check(test.flag); err != nil {
			t.Fatal(err)
		}
		solve(task.New(test.args))
	}
}
