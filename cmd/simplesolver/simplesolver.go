package main

import (
	"github.com/go-ricrob/exec/task"
	"github.com/go-ricrob/simplesolver/internal/packed"
	"github.com/go-ricrob/simplesolver/solver"
)

func solve(task *task.Task) {
	var runner solver.Runner
	if task.Args.HasSilverRobot() {
		runner = solver.New[packed.P5](task, true)
	} else {
		runner = solver.New[packed.P4](task, false)
	}
	result := runner.Run()
	moves, err := result.Moves()
	if err != nil {
		task.Exit(err)
	}
	task.Result(moves, "numCalcMove", result.NumCalcMove())
}

func main() {
	task, err := task.NewByFlag()
	if err != nil {
		task.Exit(err)
	}
	solve(task)
}
