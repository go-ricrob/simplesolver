package main

import (
	"github.com/go-ricrob/exec/task"
	"github.com/go-ricrob/simplesolver/internal/packed"
	"github.com/go-ricrob/simplesolver/internal/solver"
)

func solve(task *task.Task) {
	var s solver.Solver
	if task.Args.Robots.HasSilver() {
		s = solver.New[packed.P5](task, true)
	} else {
		s = solver.New[packed.P4](task, false)
	}
	result := s.Run()
	task.Result(result.Moves, "numCalcMove", result.NumCalcMove)
}

func main() {
	task, err := task.NewFromFlag()
	if err != nil {
		task.Exit(err)
	}
	solve(task)
}
