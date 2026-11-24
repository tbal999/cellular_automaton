// Package object holds the cellular automaton world and its update rules.
package object

// Lifeform is one cell.
//
// Alive is the cell's state: 0 = dead, 1 = alive, and for Generations rules
// (States > 2) values 2..States-1 are "dying" states that the cell passes
// through on its way back to dead. Only state 1 counts as a live neighbour.
//
// Still counts how long the cell has been alive (capped), and is used to
// colour older cells differently.
type Lifeform struct {
	Alive int
	Next  int
	Still int
}

// MaxStill caps the age counter.
const MaxStill = 62

func Newlife() Lifeform {
	return Lifeform{}
}

// Newworld returns a height x width grid of dead cells.
func Newworld(height, width int) [][]Lifeform {
	w := make([][]Lifeform, height)
	for y := range w {
		w[y] = make([]Lifeform, width)
	}
	return w
}

func live(c *Lifeform) int {
	if c.Alive == 1 {
		return 1
	}
	return 0
}

// Adjust computes every cell's Next state from the current states. The grid
// wraps at the edges (toroidal); only state 1 counts as a live neighbour.
func Adjust(world [][]Lifeform, rule Rule) {
	h := len(world)
	if h == 0 || len(world[0]) == 0 {
		return
	}
	w := len(world[0])
	birth, survive := rule.tables()
	for y := range world {
		up, row, down := world[(y-1+h)%h], world[y], world[(y+1)%h]
		for x := range row {
			l, r := x-1, x+1
			if l < 0 {
				l = w - 1
			}
			if r == w {
				r = 0
			}
			n := live(&up[l]) + live(&up[x]) + live(&up[r]) +
				live(&row[l]) + live(&row[r]) +
				live(&down[l]) + live(&down[x]) + live(&down[r])
			row[x].Next = rule.nextFast(row[x].Alive, n, &birth, &survive)
		}
	}
}

// Step commits each cell's computed Next state and updates its age counter.
func Step(world [][]Lifeform) {
	for y := range world {
		for x := range world[y] {
			c := &world[y][x]
			c.Alive = c.Next
			if c.Alive == 1 {
				if c.Still < MaxStill {
					c.Still++
				}
			} else if c.Still > 0 {
				c.Still--
			}
		}
	}
}
