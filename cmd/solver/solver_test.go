package main

import (
	"testing"

	"github.com/go-ricrob/exec/task"
)

func TestSolver(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long running solves")
	}

	tests := []struct {
		args *task.Args
	}{
		{
			&task.Args{
				Tiles: &task.Tiles{
					TopLeft:     "A3F",
					TopRight:    "A1B",
					BottomLeft:  "A2B",
					BottomRight: "A4F",
				},

				Robots: &task.Robots{
					Yellow: task.Coordinate{X: 15, Y: 0},
					Red:    task.Coordinate{X: 14, Y: 2},
					Green:  task.Coordinate{X: 1, Y: 13},
					Blue:   task.Coordinate{X: 13, Y: 11},
					Silver: task.Coordinate{X: -1, Y: -1},
				},

				TargetSymbol: task.BluePyramid,
			},
		},
		{
			&task.Args{
				Tiles: &task.Tiles{
					TopLeft:     "A1F",
					TopRight:    "A4F",
					BottomLeft:  "A3F",
					BottomRight: "A2B",
				},

				Robots: &task.Robots{
					Yellow: task.Coordinate{X: 12, Y: 15},
					Red:    task.Coordinate{X: 12, Y: 14},
					Green:  task.Coordinate{X: 1, Y: 0},
					Blue:   task.Coordinate{X: 15, Y: 15},
					Silver: task.Coordinate{X: -1, Y: -1},
				},

				TargetSymbol: task.BluePyramid,
			},
		},
	}

	for i, test := range tests {
		if i != 0 {
			solve(task.New(test.args))
		}
	}
}
