package main

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func BenchmarkRender(b *testing.B) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()
	s.SetSize(240, 65)
	g := newGame(s)
	g.randomFill()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.render()
	}
}

func BenchmarkStep(b *testing.B) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()
	s.SetSize(240, 65)
	g := newGame(s)
	g.randomFill()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.step()
	}
}
