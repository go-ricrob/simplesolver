package solver

import (
	"fmt"

	"github.com/go-ricrob/exec/task"
	"github.com/go-ricrob/game/board"
)

func convertSymbolIn(symbol task.Symbol) (board.Symbol, board.Color) {
	switch symbol {

	case task.YellowPyramid:
		return board.Pyramid, board.Yellow
	case task.YellowStar:
		return board.Star, board.Yellow
	case task.YellowMoon:
		return board.Moon, board.Yellow
	case task.YellowSaturn:
		return board.Saturn, board.Yellow

	case task.RedPyramid:
		return board.Pyramid, board.Red
	case task.RedStar:
		return board.Star, board.Red
	case task.RedMoon:
		return board.Moon, board.Red
	case task.RedSaturn:
		return board.Saturn, board.Red

	case task.GreenPyramid:
		return board.Pyramid, board.Green
	case task.GreenStar:
		return board.Star, board.Green
	case task.GreenMoon:
		return board.Moon, board.Green
	case task.GreenSaturn:
		return board.Saturn, board.Green

	case task.BluePyramid:
		return board.Pyramid, board.Blue
	case task.BlueStar:
		return board.Star, board.Blue
	case task.BlueMoon:
		return board.Moon, board.Blue
	case task.BlueSaturn:
		return board.Saturn, board.Blue

	case task.Cosmic:
		return board.Cosmic, 0

	default:
		panic(fmt.Sprintf("invalid symbol %d", symbol))

	}
}

func convertRobotOut(color board.Color) task.Robot {
	switch color {
	case board.Yellow:
		return task.YellowRobot
	case board.Red:
		return task.RedRobot
	case board.Green:
		return task.GreenRobot
	case board.Blue:
		return task.BlueRobot
	case board.Silver:
		return task.SilverRobot
	default:
		panic(fmt.Sprintf("invalid color %s", color))
	}
}
