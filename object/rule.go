package object

import (
	"fmt"
	"strconv"
	"strings"
)

// MaxStates bounds Generations rules.
const MaxStates = 64

// Rule is a Life-like rule in B/S notation, optionally with a state count
// (Generations rules, written B.../S.../C<n>). States == 2 is a plain
// alive/dead rule.
type Rule struct {
	Birth   []int
	Survive []int
	States  int
}

// tables returns lookup tables indexed by live-neighbour count (0-8).
func (r Rule) tables() (birth, survive [9]bool) {
	for _, n := range r.Birth {
		birth[n] = true
	}
	for _, n := range r.Survive {
		survive[n] = true
	}
	return
}

// next returns the state a cell moves to given its state and live neighbours.
func (r Rule) next(state, live int) int {
	birth, survive := r.tables()
	return r.nextFast(state, live, &birth, &survive)
}

// nextFast is next with precomputed tables (see tables).
func (r Rule) nextFast(state, live int, birth, survive *[9]bool) int {
	switch state {
	case 0:
		if birth[live] {
			return 1
		}
		return 0
	case 1:
		if survive[live] {
			return 1
		}
	}
	// Alive cells that don't survive, and dying cells, advance one state.
	// Anything at or past the last state is dead (this also cleans up dying
	// cells left over after switching to a rule with fewer states).
	if state+1 >= r.States {
		return 0
	}
	return state + 1
}

// String formats the rule as e.g. "B3/S23" or "B2/S/C3".
func (r Rule) String() string {
	digits := func(xs []int) string {
		var b strings.Builder
		for _, x := range xs {
			b.WriteString(strconv.Itoa(x))
		}
		return b.String()
	}
	s := "B" + digits(r.Birth) + "/S" + digits(r.Survive)
	if r.States > 2 {
		s += "/C" + strconv.Itoa(r.States)
	}
	return s
}

// ParseRule parses "B3/S23", "b36/s23", "B2/S/C3" or "B3,6/S2,3".
// B and S are required (either may be empty); C defaults to 2.
func ParseRule(s string) (Rule, error) {
	r := Rule{States: 2}
	var haveB, haveS, haveC bool
	s = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(s), " ", ""))
	if s == "" {
		return r, fmt.Errorf("empty rule")
	}
	for _, part := range strings.Split(s, "/") {
		if part == "" {
			return r, fmt.Errorf("empty section in %q", s)
		}
		body := strings.ReplaceAll(part[1:], ",", "")
		switch part[0] {
		case 'B', 'S':
			counts, err := parseCounts(body)
			if err != nil {
				return r, err
			}
			if part[0] == 'B' {
				if haveB {
					return r, fmt.Errorf("B given twice")
				}
				r.Birth, haveB = counts, true
			} else {
				if haveS {
					return r, fmt.Errorf("S given twice")
				}
				r.Survive, haveS = counts, true
			}
		case 'C':
			n, err := strconv.Atoi(body)
			if haveC || err != nil || n < 2 || n > MaxStates {
				return r, fmt.Errorf("bad C%s (use C2-C%d)", body, MaxStates)
			}
			r.States, haveC = n, true
		default:
			return r, fmt.Errorf("unknown section %q (use B, S, C)", part)
		}
	}
	if !haveB || !haveS {
		return r, fmt.Errorf("need both B and S, e.g. B3/S23")
	}
	return r, nil
}

// parseCounts turns "236" into [2 3 6], rejecting anything but 0-8 and
// ignoring duplicates.
func parseCounts(s string) ([]int, error) {
	out := []int{}
	seen := map[int]bool{}
	for _, c := range s {
		if c < '0' || c > '8' {
			return nil, fmt.Errorf("bad count %q (use 0-8)", c)
		}
		n := int(c - '0')
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out, nil
}

// MustParseRule is ParseRule for known-good literals; it panics on error.
func MustParseRule(s string) Rule {
	r, err := ParseRule(s)
	if err != nil {
		panic(err)
	}
	return r
}

// Preset is a named rule.
type Preset struct {
	Name string
	Rule Rule
}

// Presets is the built-in rule list, cycled with [ and ] in the UI.
var Presets = func() []Preset {
	defs := []struct{ name, rule string }{
		// Life-like (2 states)
		{"Conway's Life", "B3/S23"},
		{"HighLife", "B36/S23"},
		{"DryLife", "B37/S23"},
		{"Move (Morley)", "B368/S245"},
		{"Flock", "B3/S12"},
		{"Pseudo Life", "B357/S238"},
		{"34 Life", "B34/S34"},
		{"Long Life", "B345/S5"},
		{"Seeds", "B2/S"},
		{"Serviettes", "B234/S"},
		{"Life without Death", "B3/S012345678"},
		{"Maze", "B3/S12345"},
		{"Mazectric", "B3/S1234"},
		{"Coral", "B3/S45678"},
		{"Walled Cities", "B45678/S2345"},
		{"Coagulations", "B378/S235678"},
		{"Stains", "B3678/S235678"},
		{"Day & Night", "B3678/S34678"},
		{"Anneal", "B4678/S35678"},
		{"Vote", "B5678/S45678"},
		{"Diamoeba", "B35678/S5678"},
		{"Amoeba", "B357/S1358"},
		{"2x2", "B36/S125"},
		{"Bacteria", "B34/S456"},
		{"Assimilation", "B345/S4567"},
		{"Replicator", "B1357/S1357"},
		{"Fredkin", "B1357/S02468"},
		{"Gnarl", "B1/S1"},
		// Generations (dying states)
		{"Brian's Brain", "B2/S/C3"},
		{"Star Wars", "B2/S345/C4"},
		{"Frogs", "B34/S12/C3"},
		{"Spirals", "B234/S2/C5"},
		{"Sticks", "B2/S3456/C6"},
		{"Cooties", "B2/S23/C8"},
		{"Swirl", "B34/S23/C8"},
		{"Belzhab", "B23/S23/C8"},
		{"Lava", "B45678/S12345/C8"},
		{"Fireworks", "B13/S2/C21"},
		{"Bloomerang", "B34678/S234/C24"},
	}
	out := make([]Preset, len(defs))
	for i, d := range defs {
		out[i] = Preset{d.name, MustParseRule(d.rule)}
	}
	return out
}()
