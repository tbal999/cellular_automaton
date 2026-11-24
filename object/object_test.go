package object

import "testing"

func alive(w [][]Lifeform) map[[2]int]bool {
	m := map[[2]int]bool{}
	for y := range w {
		for x := range w[y] {
			if w[y][x].Alive == 1 {
				m[[2]int{y, x}] = true
			}
		}
	}
	return m
}

func TestNewworldSize(t *testing.T) {
	w := Newworld(45, 135)
	if len(w) != 45 || len(w[0]) != 135 {
		t.Fatalf("got %dx%d, want 45x135", len(w), len(w[0]))
	}
}

// A blinker that straddles the left/right edge must oscillate via wrap-around.
func TestBlinkerWrapsAcrossEdge(t *testing.T) {
	w := Newworld(10, 10)
	for _, x := range []int{9, 0, 1} {
		w[5][x].Alive = 1
	}
	life := MustParseRule("B3/S23")
	Adjust(w, life)
	Step(w)
	want := map[[2]int]bool{{4, 0}: true, {5, 0}: true, {6, 0}: true}
	if got := alive(w); len(got) != 3 || !got[[2]int{4, 0}] || !got[[2]int{5, 0}] || !got[[2]int{6, 0}] {
		t.Fatalf("after 1 step got %v, want %v", got, want)
	}
	Adjust(w, life)
	Step(w)
	if got := alive(w); len(got) != 3 || !got[[2]int{5, 9}] || !got[[2]int{5, 0}] || !got[[2]int{5, 1}] {
		t.Fatalf("after 2 steps got %v", got)
	}
}

// Brian's Brain: alive -> dying -> dead, and dying cells don't count as live.
func TestGenerationsCycle(t *testing.T) {
	rule := MustParseRule("B2/S/C3")
	w := Newworld(5, 5)
	w[2][2].Alive = 1
	Adjust(w, rule)
	Step(w)
	if w[2][2].Alive != 2 {
		t.Fatalf("alive cell should start dying, got state %d", w[2][2].Alive)
	}
	Adjust(w, rule)
	Step(w)
	if w[2][2].Alive != 0 {
		t.Fatalf("dying cell should be dead, got state %d", w[2][2].Alive)
	}
	// Two live cells give their shared neighbours exactly 2 -> births; a
	// dying neighbour must not count.
	w = Newworld(5, 5)
	w[2][1].Alive, w[2][3].Alive, w[1][2].Alive = 1, 1, 2
	Adjust(w, rule)
	if w[3][2].Next != 1 {
		t.Fatal("cell with 2 live neighbours should be born")
	}
	if w[1][1].Next != 0 { // neighbours: live (2,1) + dying (1,2) = 1 live
		t.Fatal("dying cell was counted as live")
	}
}

func TestLeftoverDyingStatesClearUnderTwoStateRule(t *testing.T) {
	w := Newworld(3, 3)
	w[1][1].Alive = 5
	Adjust(w, MustParseRule("B3/S23"))
	if w[1][1].Next != 0 {
		t.Fatalf("got %d, want 0", w[1][1].Next)
	}
}

func TestParseRule(t *testing.T) {
	good := map[string]string{
		"B3/S23":     "B3/S23",
		"b36/s23":    "B36/S23",
		"S23/B3":     "B3/S23",
		"B3,6/S2,3":  "B36/S23",
		"B2/S/C3":    "B2/S/C3",
		"B2/S":       "B2/S",
		"B33/S2/C2":  "B3/S2",
		" B3 / S23 ": "B3/S23",
	}
	for in, want := range good {
		r, err := ParseRule(in)
		if err != nil || r.String() != want {
			t.Errorf("ParseRule(%q) = %v, %v; want %s", in, r, err, want)
		}
	}
	for _, in := range []string{"", "B3", "S23", "B9/S23", "B3/S23/C1", "B3/S23/C99", "B3/S23/X1", "B3//S23", "B3/B4/S2", "3/23"} {
		if _, err := ParseRule(in); err == nil {
			t.Errorf("ParseRule(%q) should fail", in)
		}
	}
}

func TestPresetsRoundTrip(t *testing.T) {
	seen := map[string]string{}
	for _, p := range Presets {
		r, err := ParseRule(p.Rule.String())
		if err != nil || r.String() != p.Rule.String() {
			t.Errorf("%s: %v does not round-trip", p.Name, p.Rule)
		}
		if other, dup := seen[p.Rule.String()]; dup {
			t.Errorf("%s duplicates %s", p.Name, other)
		}
		seen[p.Rule.String()] = p.Name
	}
}
