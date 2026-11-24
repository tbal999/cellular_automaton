package main

import (
	"cellular_automaton/object"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

func newTestGame(t *testing.T, w, h int) (*game, tcell.SimulationScreen) {
	t.Helper()
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Fini)
	s.SetSize(w, h)
	return newGame(s), s
}

func TestWorldFitsBox(t *testing.T) {
	g, _ := newTestGame(t, 120, 40)
	// box interior: cols 120-36-1-1 = 82, screen rows 40-1-1-1 = 37 -> 74 world rows
	if len(g.world) != 74 || len(g.world[0]) != 82 {
		t.Fatalf("world %dx%d, want 82x74", len(g.world[0]), len(g.world))
	}
}

func TestPaintAndRender(t *testing.T) {
	g, s := newTestGame(t, 120, 40)
	g.paint(1, 2, true) // first interior screen cell
	if g.world[0][0].Alive != 1 || g.world[1][0].Alive != 1 {
		t.Fatal("paint did not set both halves of the top-left cell")
	}
	g.render()
	r, _, st, _ := s.GetContent(1, 2)
	fg, bg, _ := st.Decompose()
	if r != '▀' || fg == colDead || bg == colDead {
		t.Fatalf("cell rendered as %q fg=%v bg=%v", r, fg, bg)
	}
	g.paint(1, 2, false)
	if g.world[0][0].Alive != 0 {
		t.Fatal("erase failed")
	}
}

func TestResizeKeepsCellsAndTooSmallDoesNotPanic(t *testing.T) {
	g, s := newTestGame(t, 120, 40)
	g.world[3][3].Alive = 1
	s.SetSize(150, 50)
	g.resizeWorld()
	if g.world[3][3].Alive != 1 {
		t.Fatal("cell lost on resize")
	}
	s.SetSize(30, 10)
	g.resizeWorld()
	g.render()
	g.step()
}

func TestInputButtonClick(t *testing.T) {
	g, s := newTestGame(t, 120, 40)
	g.render()
	bx, by, _ := g.inputButton()
	g.handleMouse(tcell.NewEventMouse(bx+1, by, tcell.Button1, 0))
	if !g.inputting {
		t.Fatal("click on button did not start input")
	}
	g.handleMouse(tcell.NewEventMouse(bx+1, by, tcell.ButtonNone, 0))
	typeRule(g, "B36/S23")
	if g.inputting || g.ruleName != "HighLife" || g.preset != 1 {
		t.Fatalf("typed preset not recognised: inputting=%v name=%q preset=%d", g.inputting, g.ruleName, g.preset)
	}
	g.startInput()
	typeRule(g, "B3/S2")
	if g.ruleName != "Custom" || g.preset != -1 || g.rule.String() != "B3/S2" {
		t.Fatalf("custom rule not applied: %q %v", g.ruleName, g.rule)
	}
	g.startInput()
	typeRule(g, "B9/S2")
	if !g.inputting || g.inputError == "" || g.rule.String() != "B3/S2" {
		t.Fatal("invalid rule should keep input open and the old rule")
	}
	_ = s
}

func typeRule(g *game, s string) {
	for _, r := range s {
		g.handleInputKey(tcell.NewEventKey(tcell.KeyRune, r, 0))
	}
	g.handleInputKey(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
}

func key(r rune) *tcell.EventKey { return tcell.NewEventKey(tcell.KeyRune, r, 0) }

func TestPresetCycling(t *testing.T) {
	g, _ := newTestGame(t, 120, 40)
	frame := time.NewTicker(time.Hour)
	defer frame.Stop()
	g.handleKey(key('['), frame)
	if g.preset != len(object.Presets)-1 {
		t.Fatalf("[ from first should wrap to last, got %d", g.preset)
	}
	g.handleKey(key(']'), frame)
	g.handleKey(key(']'), frame)
	if g.preset != 1 || g.ruleName != "HighLife" {
		t.Fatalf("got preset %d %q", g.preset, g.ruleName)
	}
}

// Every preset's panel must fit inside the panel border and the minimum height.
func TestPanelFits(t *testing.T) {
	for i := range object.Presets {
		g, s := newTestGame(t, minWidth, 60)
		g.setPreset(i)
		bx, _, _ := g.inputButton()
		w, _ := s.Size()
		for _, ln := range g.panelLines() {
			if bx+len([]rune(ln.msg)) > w-1 {
				t.Errorf("preset %d: line %q overflows panel", i, ln.msg)
			}
		}
		s.SetSize(minWidth, g.minHeight())
		g.resizeWorld()
		g.render()
		if r, _, _, _ := s.GetContent(0, 0); r == 'T' { // "Terminal is too small"
			t.Errorf("preset %d: min size %dx%d rejected", i, minWidth, g.minHeight())
		}
	}
}

func TestDyingCellsRenderDistinctly(t *testing.T) {
	g, _ := newTestGame(t, 120, 40)
	g.setPreset(len(object.Presets) - 1) // a Generations rule
	alive := g.cellColour(object.Lifeform{Alive: 1})
	dying := g.cellColour(object.Lifeform{Alive: 2})
	if dying == colDead || dying == alive {
		t.Fatalf("dying colour %v should differ from dead and alive", dying)
	}
}

// A fast drag delivers sparse motion events; the cells between must be filled.
func TestDragFillsGaps(t *testing.T) {
	g, _ := newTestGame(t, 120, 40)
	g.handleMouse(tcell.NewEventMouse(2, 5, tcell.Button1, 0))
	g.handleMouse(tcell.NewEventMouse(40, 12, tcell.Button1, 0)) // big jump
	g.handleMouse(tcell.NewEventMouse(40, 12, tcell.ButtonNone, 0))
	left, top, _, _ := g.layout()
	for sx := 2; sx <= 40; sx++ {
		col := sx - left - 1
		found := false
		for sy := 5; sy <= 12; sy++ {
			if g.world[2*(sy-top-1)][col].Alive == 1 {
				found = true
			}
		}
		if !found {
			t.Fatalf("gap in dragged line at screen column %d", sx)
		}
	}
	// Releasing and clicking elsewhere must not draw a line from the old spot.
	g.clear()
	g.handleMouse(tcell.NewEventMouse(60, 30, tcell.Button1, 0))
	n := 0
	for y := range g.world {
		for x := range g.world[y] {
			n += g.world[y][x].Alive
		}
	}
	if n != 1 { // a click places only the top cell
		t.Fatalf("single click painted %d cells, want 1", n)
	}
}

func TestClickCyclesHalves(t *testing.T) {
	g, _ := newTestGame(t, 120, 40)
	left, top, _, _ := g.layout()
	sx, sy := 10, 8
	x, y := sx-left-1, 2*(sy-top-1)
	click := func() {
		g.handleMouse(tcell.NewEventMouse(sx, sy, tcell.Button1, 0))
		g.handleMouse(tcell.NewEventMouse(sx, sy, tcell.ButtonNone, 0))
	}
	want := [][2]int{{1, 0}, {0, 1}, {1, 1}, {0, 0}, {1, 0}} // top, bottom
	for i, w := range want {
		click()
		got := [2]int{g.world[y][x].Alive, g.world[y+1][x].Alive}
		if got != w {
			t.Fatalf("click %d: got top/bottom %v, want %v", i+1, got, w)
		}
	}
}

// A glider needs a bottom-only cell next to a top-only one, which cycling
// makes possible; check it then flies under Conway's rules.
func TestCanPlaceGliderByClicking(t *testing.T) {
	g, _ := newTestGame(t, 120, 40)
	left, top, _, _ := g.layout()
	click := func(wx, wy, times int) { // world coords -> screen clicks
		for i := 0; i < times; i++ {
			sx, sy := wx+left+1, wy/2+top+1
			g.handleMouse(tcell.NewEventMouse(sx, sy, tcell.Button1, 0))
			g.handleMouse(tcell.NewEventMouse(sx, sy, tcell.ButtonNone, 0))
		}
	}
	// Glider at rows 10-12:  .O. / ..O / OOO
	click(11, 10, 1) // row 10 = top of screen row 5
	click(12, 10, 2) // row 11 = bottom of screen row 5 (two clicks)
	click(10, 12, 1) // row 12 = top of screen row 6
	click(11, 12, 1)
	click(12, 12, 1)
	for i := 0; i < 4; i++ {
		g.step()
	}
	// After 4 generations a glider has moved one cell down-right.
	for _, c := range [][2]int{{12, 11}, {13, 12}, {11, 13}, {12, 13}, {13, 13}} {
		if g.world[c[1]][c[0]].Alive != 1 {
			t.Fatalf("glider cell (%d,%d) missing after 4 steps", c[0], c[1])
		}
	}
}
