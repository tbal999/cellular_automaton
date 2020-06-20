package main

import (
	"corridors/gamesound"
	"corridors/object"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nsf/termbox-go"
)

var (
	backbuffer     []termbox.Cell
	timer          = 30
	pausegame      = false
	gameover       = false
	inputkey       = false
	secondoption   = false
	world          [][]object.Lifeform
	mouseX, mouseY int
	tryinput       = "Click here to change rules"
	rawinput       string
	l              []int
	d              []int
	lstring        string
	dstring        string
)

var Colors = []termbox.Attribute{
	termbox.ColorBlack,
	termbox.ColorRed,
	termbox.ColorGreen,
	termbox.ColorYellow,
	termbox.ColorBlue,
	termbox.ColorMagenta,
	termbox.ColorCyan,
	termbox.ColorWhite,
}

func regenerateBuffer(x, y int) {
	backbuffer = make([]termbox.Cell, x*y)
}

func adjuststring(stringtoedit *string, copy string) {
	s := *stringtoedit
	s = copy
	*stringtoedit = s
}

func renderText(y, x int, foreground, background termbox.Attribute, msg string) {
	for _, c := range msg {
		termbox.SetCell(x, y, c, foreground, background)
		x++
	}
}

func renderRect(y1, y2, x1, x2 int, colour termbox.Attribute, pen bool) {
	for yindex := y1; yindex <= y2; yindex++ {
		for xindex := x1; xindex <= x2; xindex++ {
			if yindex == y1 || yindex == y2 {
				termbox.SetCell(xindex, yindex, 'x', colour, colour)
			}
			if xindex == x1 || xindex == x2 {
				termbox.SetCell(xindex, yindex, 'x', colour, colour)
			}
			if yindex != y1 && yindex != y2 && xindex != x1 && xindex != x2 {
				if pen == true {
					if world[yindex][xindex].Alive == 1 {
						termbox.SetCell(xindex, yindex, 'x', termbox.ColorRed, termbox.ColorRed)
					}
				}
			}
		}
	}
}

func renderLife(y1, y2, x1, x2 int, colour termbox.Attribute, pen bool) {
	i := world
	for y := y1; y <= y2; y++ {
		for x := x1; x <= x2; x++ {
			if i[y][x].Alive == 0 {
				termbox.SetCell(x, y, 'x', termbox.ColorBlack, termbox.ColorBlack)
			} else {
				if i[y][x].Still < 4 {
					termbox.SetCell(x, y, 'x', termbox.ColorRed, termbox.ColorRed)
				} else if i[y][x].Still < 30 {
					termbox.SetCell(x, y, 'x', termbox.ColorBlue, termbox.ColorBlue)
				} else if i[y][x].Still < 60 {
					termbox.SetCell(x, y, 'x', termbox.ColorYellow, termbox.ColorYellow)
				} else {
					termbox.SetCell(x, y, 'x', termbox.ColorGreen, termbox.ColorGreen)
				}
			}
			if i[y][x].Next == 0 {
				i[y][x].Alive = 0
				if i[y][x].Still > 0 {
					i[y][x].Still--
				}
			} else {
				i[y][x].Alive = 1
				if i[y][x].Still <= 61 {
					i[y][x].Still++
				}
			}
		}
	}
}

func renderAll() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	renderText(1, 70, termbox.ColorWhite, termbox.ColorDefault, "CELLULAR AUTOMATON - V2") //TITLE
	renderRect(3, 49, 3, 139, termbox.ColorBlue, true)
	renderRect(3, 49, 145, 180, termbox.ColorYellow, false) //ARMOUR
	renderText(4, 148, termbox.ColorWhite, termbox.ColorDefault, "Type in like this: 1,2,3")
	renderText(5, 148, termbox.ColorWhite, termbox.ColorRed, tryinput)
	renderText(6, 148, termbox.ColorWhite, termbox.ColorDefault, rawinput)
	renderText(7, 148, termbox.ColorWhite, termbox.ColorDefault, "B RULES: "+dstring)
	renderText(9, 148, termbox.ColorWhite, termbox.ColorDefault, "S RULES: "+lstring)
	renderText(11, 148, termbox.ColorWhite, termbox.ColorDefault, "Instructions:")
	renderText(13, 148, termbox.ColorWhite, termbox.ColorDefault, "Press 'space' to reset world")
	renderText(14, 148, termbox.ColorWhite, termbox.ColorDefault, "Press 'up arrow' to pause world")
	renderText(15, 148, termbox.ColorWhite, termbox.ColorDefault, "Press 'left arrow' to speed up")
	renderText(16, 148, termbox.ColorWhite, termbox.ColorDefault, "Press 'right arrow' to slow down")
	timertext := fmt.Sprintf("Speed: %d ms per frame", timer)
	renderText(18, 148, termbox.ColorWhite, termbox.ColorDefault, timertext)
	renderText(20, 148, termbox.ColorWhite, termbox.ColorDefault, "Example rules (B / S):")
	renderText(22, 148, termbox.ColorWhite, termbox.ColorDefault, "3,6/1,2,5 - 2x2:")
	renderText(23, 148, termbox.ColorWhite, termbox.ColorDefault, "3,5,7/1,3,5,8 - Amoeba:")
	renderText(24, 148, termbox.ColorWhite, termbox.ColorDefault, "3/2,3 - Conway's Life:")
	renderText(25, 148, termbox.ColorWhite, termbox.ColorDefault, "3/4,5,6,7,8 - Growing Coral")
	renderText(26, 148, termbox.ColorWhite, termbox.ColorDefault, "3,6,7,8/3,4,6,7,8 - Day&Night")
	renderText(27, 148, termbox.ColorWhite, termbox.ColorDefault, "3/1,2,3,4,5 - Maze")
	renderText(28, 148, termbox.ColorWhite, termbox.ColorDefault, "3,6,8/2,4,5 - Move")
	renderText(29, 148, termbox.ColorWhite, termbox.ColorDefault, "1/1 - Gnarl")
	if pausegame != true {
		renderLife(4, 48, 4, 138, termbox.ColorBlue, false) //MAIN
	}
	termbox.Flush()
}

func setToOne(x, y int, w *[][]object.Lifeform, live bool) {
	switch live {
	case true:
		world := *w
		if x <= len(world[0])-1 && y <= len(world)-1 {
			world[y][x].Alive = 1
		}
		*w = world
	case false:
		world := *w
		if x <= len(world[0])-1 && y <= len(world)-1 {
			world[y][x].Alive = 0
		}
		*w = world
	}
}

//Resets the map
func reset(world *[][]object.Lifeform) {
	i := *world
	for y := range i {
		for x := range i[y] {
			i[y][x].Alive = 0
			i[y][x].Next = 0
		}
	}
	*world = i
}

func main() {
	//Scanner := bufio.NewScanner(os.Stdin)
	world = object.Newworld(50, 140)
	go gamesound.Play("main")
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	Frame := time.NewTicker(time.Duration(timer) * time.Millisecond)
	termbox.SetInputMode(termbox.InputEsc | termbox.InputMouse | termbox.InputAlt)
	eventQueue := make(chan termbox.Event)
	regenerateBuffer(65, 185)
	termbox.Flush()
	go func() {
		for {
			eventQueue <- termbox.PollEvent()
		}
	}()
	for gameover == false {
		select {
		case ev := <-eventQueue:
			if ev.Type == termbox.EventResize {
				regenerateBuffer(termbox.Size())
			}
			if ev.Type == termbox.EventKey {
				if ev.Key == termbox.KeyEsc {
					gameover = true
					termbox.Close()
				}
				if ev.Key == termbox.KeySpace {
					reset(&world)
					break
				}
				if ev.Key == termbox.KeyArrowUp {
					switch pausegame {
					case false:
						pausegame = true
					case true:
						pausegame = false
					}
				}
				if ev.Key == termbox.KeyArrowLeft {
					if timer >= 2 {
						timer--
					}
					if timer >= 20 {
						timer -= 10
					}
					if timer >= 100 {
						timer -= 50
					}
					Frame = time.NewTicker(time.Duration(timer) * time.Millisecond)
					gamesound.Up()
				}
				if ev.Key == termbox.KeyArrowRight {
					if timer <= 500 {
						timer += 10
					}
					if timer <= 1000 {
						timer += 30
					}
					Frame = time.NewTicker(time.Duration(timer) * time.Millisecond)
					gamesound.Down()
				}
				if inputkey == true {
					if ev.Key == termbox.KeyEnter {
						switch secondoption {
						case false:
							secondoption = true
							adjuststring(&dstring, rawinput)
							rawinput = ""
							tryinput = "S rule: " + rawinput
							ruleB := strings.Split(dstring, ",")
							d = []int{}
							for indexx := range ruleB {
								integer, _ := strconv.Atoi(ruleB[indexx])

								d = append(d, integer)
							}
						case true:
							secondoption = false
							inputkey = false
							adjuststring(&lstring, rawinput)
							ruleS := strings.Split(lstring, ",")
							l = []int{}
							for indexxx := range ruleS {
								integer2, _ := strconv.Atoi(ruleS[indexxx])
								l = append(l, integer2)
							}
							rawinput = ""
							tryinput = "Click here to change rules"
						}

					} else {
						switch string(ev.Ch) {
						default:
							rawinput += string(ev.Ch)
						}
					}
				}
			}
			if ev.Type == termbox.EventMouse {
				if ev.Key == termbox.MouseLeft {
					mouseX, mouseY = ev.MouseX, ev.MouseY
					if mouseX < 140 && mouseY < 50 {
						//test = fmt.Sprintf("%d, %d", mouseX, mouseY)
						setToOne(mouseX, mouseY, &world, true)
					}
					if mouseX > 147 && mouseX < 176 && mouseY == 5 {
						switch inputkey {
						case false:
							inputkey = true
						case true:
							inputkey = false
						}

						tryinput = "B rule: " + rawinput
					}
				}
				if ev.Key == termbox.MouseRight {
					mouseX, mouseY = ev.MouseX, ev.MouseY
					tryinput = fmt.Sprintf("%d, %d", mouseX, mouseY)
					setToOne(mouseX, mouseY, &world, false)
				}
			}
		case <-Frame.C:
			object.Adjust(&world, l, d)
			renderAll()
		}
	}
}
