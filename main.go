package main

import (
	"cellular_automaton/object"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
)

// Layout. Row 0 holds the title; the world box fills the rest of the screen
// left of a fixed-width side panel. Each terminal cell inside the box shows
// two world cells (top and bottom half), so cells come out roughly square.
const (
	panelWidth  = 36
	minWidth    = panelWidth + 20
	boxTop      = 1
	presetRows  = 9 // how many presets the panel lists around the current one
	maxInputLen = 32
)

var (
	colBorder = tcell.NewRGBColor(70, 110, 200)
	colPanel  = tcell.NewRGBColor(200, 170, 60)
	colDead   = tcell.NewRGBColor(12, 12, 18)
	colText   = tcell.NewRGBColor(220, 220, 220)
	colDim    = tcell.NewRGBColor(130, 130, 140)
	colError  = tcell.NewRGBColor(240, 80, 80)
	colButton = tcell.NewRGBColor(160, 40, 40)
	colDying  = tcell.NewRGBColor(150, 70, 230) // first dying state; fades to dead

	// Age gradient for live cells: newborn → long-lived.
	ageStops = []tcell.Color{
		tcell.NewRGBColor(255, 70, 50),   // red
		tcell.NewRGBColor(255, 190, 40),  // amber
		tcell.NewRGBColor(90, 220, 100),  // green
		tcell.NewRGBColor(50, 190, 230),  // cyan
		tcell.NewRGBColor(110, 110, 255), // blue
	}
)

type game struct {
	screen tcell.Screen
	world  [][]object.Lifeform

	timer   int // ms per frame
	running bool

	rule     object.Rule
	ruleName string
	preset   int // index into object.Presets, or -1 for a typed rule

	// Rule entry state.
	inputting  bool
	rawinput   string
	inputError string

	prevButtons  tcell.ButtonMask
	lastX, lastY int // last mouse position, for filling gaps in drags
}

func newGame(s tcell.Screen) *game {
	g := &game{screen: s, timer: 30, running: true}
	g.setPreset(0) // Conway's Life
	g.resizeWorld()
	return g
}

func (g *game) setPreset(i int) {
	n := len(object.Presets)
	i = ((i % n) + n) % n
	g.preset = i
	g.rule = object.Presets[i].Rule
	g.ruleName = object.Presets[i].Name
}

// layout returns the world box's border coordinates for the current size.
func (g *game) layout() (left, top, right, bottom int) {
	w, h := g.screen.Size()
	return 0, boxTop, w - panelWidth - 1, h - 1
}

func (g *game) panelLeft() int {
	_, _, right, _ := g.layout()
	return right + 1
}

func (g *game) inputButton() (x, y, width int) {
	return g.panelLeft() + 2, boxTop + 2, panelWidth - 4
}

// resizeWorld fits the world to the box, keeping any cells that still fit.
func (g *game) resizeWorld() {
	left, top, right, bottom := g.layout()
	rows, cols := 2*(bottom-top-1), right-left-1
	if rows < 2 || cols < 2 {
		return
	}
	nw := object.Newworld(rows, cols)
	for y := 0; y < rows && y < len(g.world); y++ {
		for x := 0; x < cols && x < len(g.world[y]); x++ {
			nw[y][x] = g.world[y][x]
		}
	}
	g.world = nw
}

func (g *game) text(x, y int, style tcell.Style, msg string) {
	for _, c := range msg {
		g.screen.SetContent(x, y, c, nil, style)
		x++
	}
}

func (g *game) box(left, top, right, bottom int, colour tcell.Color) {
	st := tcell.StyleDefault.Foreground(colour).Background(tcell.ColorReset)
	for x := left + 1; x < right; x++ {
		g.screen.SetContent(x, top, '─', nil, st)
		g.screen.SetContent(x, bottom, '─', nil, st)
	}
	for y := top + 1; y < bottom; y++ {
		g.screen.SetContent(left, y, '│', nil, st)
		g.screen.SetContent(right, y, '│', nil, st)
	}
	g.screen.SetContent(left, top, '┌', nil, st)
	g.screen.SetContent(right, top, '┐', nil, st)
	g.screen.SetContent(left, bottom, '└', nil, st)
	g.screen.SetContent(right, bottom, '┘', nil, st)
}

func lerpColour(a, b tcell.Color, t float64) tcell.Color {
	r1, g1, b1 := a.RGB()
	r2, g2, b2 := b.RGB()
	l := func(x, y int32) int32 { return x + int32(float64(y-x)*t) }
	return tcell.NewRGBColor(l(r1, r2), l(g1, g2), l(b1, b2))
}

// ageColour maps a cell's age counter onto the gradient.
func ageColour(still int) tcell.Color {
	t := float64(still) / float64(object.MaxStill)
	if t > 1 {
		t = 1
	}
	pos := t * float64(len(ageStops)-1)
	i := int(pos)
	if i >= len(ageStops)-1 {
		return ageStops[len(ageStops)-1]
	}
	return lerpColour(ageStops[i], ageStops[i+1], pos-float64(i))
}

func (g *game) cellColour(c object.Lifeform) tcell.Color {
	switch {
	case c.Alive == 1:
		return ageColour(c.Still)
	case c.Alive >= 2 && c.Alive < g.rule.States:
		// Dying states fade from colDying towards the background.
		t := float64(c.Alive-2) / float64(g.rule.States-1)
		return lerpColour(colDying, colDead, t)
	}
	return colDead
}

func (g *game) renderWorld() {
	left, top, _, _ := g.layout()
	for y := 0; y+1 < len(g.world); y += 2 {
		for x := range g.world[y] {
			st := tcell.StyleDefault.
				Foreground(g.cellColour(g.world[y][x])).
				Background(g.cellColour(g.world[y+1][x]))
			g.screen.SetContent(left+1+x, top+1+y/2, '▀', nil, st)
		}
	}
}

type panelLine struct {
	st  tcell.Style
	msg string
}

// panelLines is everything printed below the input button.
func (g *game) panelLines() []panelLine {
	plain := tcell.StyleDefault.Foreground(colText)
	dim := tcell.StyleDefault.Foreground(colDim)
	state := "RUNNING"
	if !g.running {
		state = "PAUSED"
	}
	worldSize := "-"
	if len(g.world) > 0 {
		worldSize = fmt.Sprintf("%dx%d", len(g.world[0]), len(g.world))
	}
	lines := []panelLine{
		{plain, "> " + g.rawinput},
		{tcell.StyleDefault.Foreground(colError), g.inputError},
		{plain.Bold(true), g.ruleName},
		{plain, g.rule.String()},
		{plain, ""},
		{plain.Bold(true), "Controls"},
		{plain, "  click  cycle top/bottom cell"},
		{plain, "  left/right drag  draw/erase"},
		{plain, "  [ ]   prev/next rule"},
		{plain, "  e     type a rule"},
		{plain, "  space clear     r random"},
		{plain, "  p/up  pause     n step"},
		{plain, "  <- -> speed     q quit"},
		{plain, ""},
		{plain, fmt.Sprintf("%s  %dms  %s", state, g.timer, worldSize)},
		{plain, ""},
		{plain.Bold(true), fmt.Sprintf("Rules (%d)", len(object.Presets))},
	}
	// A window of presets centred on the current one.
	n := len(object.Presets)
	centre := g.preset
	if centre < 0 {
		centre = 0
	}
	start := centre - presetRows/2
	if start < 0 {
		start = 0
	}
	if start > n-presetRows {
		start = n - presetRows
	}
	for i := start; i < start+presetRows && i < n; i++ {
		p := object.Presets[i]
		st, mark := dim, "  "
		if i == g.preset {
			st, mark = plain.Bold(true), "> "
		}
		lines = append(lines, panelLine{st, mark + p.Name})
	}
	return lines
}

// minHeight is the smallest terminal height that fits the whole panel.
func (g *game) minHeight() int {
	_, by, _ := g.inputButton()
	return by + 1 + len(g.panelLines()) + 1
}

func (g *game) render() {
	s := g.screen
	s.Clear()
	w, h := s.Size()
	plain := tcell.StyleDefault.Foreground(colText)
	if mh := g.minHeight(); w < minWidth || h < mh {
		g.text(0, 0, plain, fmt.Sprintf("Terminal is %dx%d; need at least %dx%d.", w, h, minWidth, mh))
		g.text(0, 1, plain, "Resize, or ctrl-c to quit.")
		s.Show()
		return
	}

	left, top, right, bottom := g.layout()
	title := "CELLULAR AUTOMATON - V3"
	g.text((left+right-len(title))/2, 0, plain.Bold(true), title)
	g.box(left, top, right, bottom, colBorder)
	g.box(g.panelLeft(), top, w-1, bottom, colPanel)
	g.renderWorld()

	bx, by, bw := g.inputButton()
	button := "Click / e: type a rule"
	if g.inputting {
		button = "Rule, e.g. B36/S23 (enter/esc)"
	}
	g.text(bx, by-1, plain, "Format: B3/S23 or B2/S/C3")
	g.text(bx, by, tcell.StyleDefault.Foreground(colText).Background(colButton),
		fmt.Sprintf("%-*s", bw, button))
	for i, ln := range g.panelLines() {
		g.text(bx, by+1+i, ln.st, ln.msg)
	}
	s.Show()
}

// paint sets or clears both world cells under screen position (sx, sy).
func (g *game) paint(sx, sy int, alive bool) {
	left, top, _, _ := g.layout()
	x, y := sx-left-1, 2*(sy-top-1)
	if x < 0 || y < 0 || y+1 >= len(g.world) || x >= len(g.world[y]) {
		return
	}
	for _, yy := range []int{y, y + 1} {
		c := &g.world[yy][x]
		if alive {
			c.Alive = 1
		} else {
			*c = object.Newlife()
		}
		c.Next = c.Alive
	}
}

func (g *game) clear() {
	for y := range g.world {
		for x := range g.world[y] {
			g.world[y][x] = object.Newlife()
		}
	}
}

func (g *game) randomFill() {
	for y := range g.world {
		for x := range g.world[y] {
			g.world[y][x] = object.Newlife()
			if rand.Intn(4) == 0 {
				g.world[y][x].Alive = 1
			}
		}
	}
}

func (g *game) step() {
	object.Adjust(g.world, g.rule)
	object.Step(g.world)
}

func (g *game) startInput() {
	g.inputting = true
	g.rawinput, g.inputError = "", ""
}

func (g *game) stopInput() {
	g.inputting = false
	g.rawinput = ""
}

// handleInputKey processes a key while a rule is being typed.
func (g *game) handleInputKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEsc:
		g.inputError = ""
		g.stopInput()
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(g.rawinput) > 0 {
			g.rawinput = g.rawinput[:len(g.rawinput)-1]
		}
	case tcell.KeyEnter:
		rule, err := object.ParseRule(g.rawinput)
		if err != nil {
			g.inputError = err.Error()
			return
		}
		g.inputError = ""
		g.rule, g.ruleName, g.preset = rule, "Custom", -1
		// Name it if it matches a preset.
		for i, p := range object.Presets {
			if p.Rule.String() == rule.String() {
				g.setPreset(i)
				break
			}
		}
		g.stopInput()
	case tcell.KeyRune:
		r := ev.Rune()
		if len(g.rawinput) < maxInputLen && strings.ContainsRune("0123456789bscBSC/,", r) {
			g.rawinput += string(r)
		}
	}
}

func (g *game) setSpeed(frame *time.Ticker, ms int) {
	g.timer = ms
	frame.Reset(time.Duration(ms) * time.Millisecond)
}

// handleKey returns false when the program should exit.
func (g *game) handleKey(ev *tcell.EventKey, frame *time.Ticker) bool {
	if ev.Key() == tcell.KeyCtrlC {
		return false
	}
	if g.inputting {
		g.handleInputKey(ev)
		return true
	}
	switch ev.Key() {
	case tcell.KeyUp:
		g.running = !g.running
	case tcell.KeyLeft:
		t := g.timer
		if t >= 2 {
			t--
		}
		if t >= 20 {
			t -= 10
		}
		if t >= 100 {
			t -= 50
		}
		g.setSpeed(frame, t)
	case tcell.KeyRight:
		t := g.timer
		if t <= 500 {
			t += 10
		}
		if t <= 1000 {
			t += 30
		}
		g.setSpeed(frame, t)
	case tcell.KeyRune:
		switch ev.Rune() {
		case ' ':
			g.clear()
		case 'r':
			g.randomFill()
		case 'p':
			g.running = !g.running
		case 'n':
			if !g.running {
				g.step()
			}
		case ']':
			g.setPreset(g.preset + 1)
		case '[':
			if g.preset < 0 {
				g.setPreset(len(object.Presets) - 1)
			} else {
				g.setPreset(g.preset - 1)
			}
		case 'e':
			g.startInput()
		case 'q':
			return false
		}
	}
	return true
}

func (g *game) handleMouse(ev *tcell.EventMouse) {
	x, y := ev.Position()
	btn := ev.Buttons() & (tcell.Button1 | tcell.Button2)
	prev, lastX, lastY := g.prevButtons, g.lastX, g.lastY
	g.prevButtons, g.lastX, g.lastY = btn, x, y
	pressed := btn &^ prev // buttons that just went down

	bx, by, bw := g.inputButton()
	if pressed&tcell.Button1 != 0 && y == by && x >= bx && x < bx+bw {
		if g.inputting {
			g.stopInput()
		} else {
			g.startInput()
		}
		g.prevButtons = 0 // don't start a drag from the button
		return
	}
	if btn == 0 {
		return
	}
	alive := btn&tcell.Button1 != 0
	switch {
	case btn == prev && x == lastX && y == lastY:
		// Repeated event at the same spot; nothing to do.
	case btn == prev:
		// Continuing a drag: fill the gap since the last position, since a
		// fast drag skips cells between motion events.
		g.paintLine(lastX, lastY, x, y, alive)
	case alive && pressed&tcell.Button1 != 0:
		// A fresh left click cycles this character's two cells.
		g.cycle(x, y)
	default:
		g.paint(x, y, alive)
	}
}

// cycle steps the two world cells under screen position (sx, sy) through
// empty -> top -> bottom -> both -> empty, so single cells can be placed in
// either half of a character.
func (g *game) cycle(sx, sy int) {
	left, top, _, _ := g.layout()
	x, y := sx-left-1, 2*(sy-top-1)
	if x < 0 || y < 0 || y+1 >= len(g.world) || x >= len(g.world[y]) {
		return
	}
	t, b := &g.world[y][x], &g.world[y+1][x]
	state := 0
	if t.Alive == 1 {
		state |= 1
	}
	if b.Alive == 1 {
		state |= 2
	}
	next := [4]int{1, 2, 3, 0}[state] // 0=empty 1=top 2=bottom 3=both
	for i, c := range []*object.Lifeform{t, b} {
		if next&(1<<i) != 0 {
			c.Alive = 1
		} else {
			*c = object.Newlife()
		}
		c.Next = c.Alive
	}
}

// paintLine paints every screen cell on the line from (x0,y0) to (x1,y1).
func (g *game) paintLine(x0, y0, x1, y1 int, alive bool) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := sign(x1-x0), sign(y1-y0)
	e := dx + dy
	for {
		g.paint(x0, y0, alive)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * e
		if e2 >= dy {
			e += dy
			x0 += sx
		}
		if e2 <= dx {
			e += dx
			y0 += sy
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

func run() error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()
	screen.EnableMouse()
	screen.HideCursor()

	g := newGame(screen)

	frame := time.NewTicker(time.Duration(g.timer) * time.Millisecond)
	defer frame.Stop()

	events := make(chan tcell.Event, 1024)
	quit := make(chan struct{})
	defer close(quit)
	go screen.ChannelEvents(events, quit)

	// handle returns false when the program should exit.
	handle := func(ev tcell.Event) bool {
		switch ev := ev.(type) {
		case *tcell.EventResize:
			screen.Sync()
			g.resizeWorld()
		case *tcell.EventKey:
			return g.handleKey(ev, frame)
		case *tcell.EventMouse:
			g.handleMouse(ev)
		}
		return true
	}

	for {
		select {
		case ev := <-events:
			if !handle(ev) {
				return nil
			}
		case <-frame.C:
			if g.running && len(g.world) > 0 {
				g.step()
			}
		}
		// Apply every event that queued up (e.g. a burst of mouse motion
		// during a drag) before drawing, so the screen never lags behind
		// the input by more than one frame.
	drain:
		for {
			select {
			case ev := <-events:
				if !handle(ev) {
					return nil
				}
			default:
				break drain
			}
		}
		g.render()
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
