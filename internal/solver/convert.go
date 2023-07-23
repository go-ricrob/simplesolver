package solver

import (
	"fmt"

	"github.com/go-ricrob/exec/task"
	"github.com/go-ricrob/game/board"
	"github.com/go-ricrob/game/types"
)

func convertSymbolIn(symbol task.Symbol) (board.Symbol, types.Color) {
	switch symbol {

	case task.YellowPyramid:
		return board.Pyramid, types.Yellow
	case task.YellowStar:
		return board.Star, types.Yellow
	case task.YellowMoon:
		return board.Moon, types.Yellow
	case task.YellowSaturn:
		return board.Saturn, types.Yellow

	case task.RedPyramid:
		return board.Pyramid, types.Red
	case task.RedStar:
		return board.Star, types.Red
	case task.RedMoon:
		return board.Moon, types.Red
	case task.RedSaturn:
		return board.Saturn, types.Red

	case task.GreenPyramid:
		return board.Pyramid, types.Green
	case task.GreenStar:
		return board.Star, types.Green
	case task.GreenMoon:
		return board.Moon, types.Green
	case task.GreenSaturn:
		return board.Saturn, types.Green

	case task.BluePyramid:
		return board.Pyramid, types.Blue
	case task.BlueStar:
		return board.Star, types.Blue
	case task.BlueMoon:
		return board.Moon, types.Blue
	case task.BlueSaturn:
		return board.Saturn, types.Blue

	case task.Cosmic:
		return board.Cosmic, 0

	default:
		panic(fmt.Sprintf("invalid symbol %d", symbol))

	}
}

func convertRobotOut(color types.Color) task.Robot {
	switch color {
	case types.Yellow:
		return task.YellowRobot
	case types.Red:
		return task.RedRobot
	case types.Green:
		return task.GreenRobot
	case types.Blue:
		return task.BlueRobot
	case types.Silver:
		return task.SilverRobot
	default:
		panic(fmt.Sprintf("invalid color %s", color))
	}
}
