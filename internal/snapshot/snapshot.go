// Package snapshot sérialise l'état d'un incendie, dans plusieurs formats
// comparables entre eux.
//
// C'est l'axe I/O de l'audit : un format naïf sert de baseline, les suivants
// cherchent à réduire la taille produite et le temps de sérialisation. Comme
// pour les moteurs de simulation, chaque format s'enregistre sous un nom et la
// suite de conformité les rejoue tous.
//
// Un format doit savoir relire ce qu'il écrit : sans cela, comparer des tailles
// n'a aucun sens — le plus petit serait celui qui perd le plus d'information.
package snapshot

import (
	"fmt"
	"io"
	"sort"

	"gol-wildfire/internal/fire"
)

// State est l'état d'un tour, tel qu'un format doit pouvoir le restituer.
// Le décor (terrain, vent) en fait partie : un format qui ne l'écrirait qu'une
// fois doit malgré tout pouvoir le rendre à la lecture.
type State struct {
	Turn          int
	Width, Height int
	Terrain       []fire.Terrain // ligne par ligne
	Wind          []uint8        // 0 = aucun ; sinon 1+direction
	Fire          []uint8        // tours de combustion restants
	Rest          []uint8        // tours de repos restants
}

// Capture relève l'état courant d'un moteur.
func Capture(e fire.Engine, turn int) *State {
	m := e.Map()
	s := &State{
		Turn: turn, Width: m.Width, Height: m.Height,
		Terrain: m.Terrain, Wind: m.Wind,
		Fire: make([]uint8, m.Width*m.Height),
		Rest: make([]uint8, m.Width*m.Height),
	}
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			s.Fire[y*m.Width+x] = e.Fire(x, y)
			s.Rest[y*m.Width+x] = e.Rest(x, y)
		}
	}
	return s
}

// Equal compare deux états case par case. Utilisé par la suite de conformité
// pour vérifier qu'un format relit exactement ce qu'il a écrit.
func (s *State) Equal(o *State) error {
	switch {
	case o == nil:
		return fmt.Errorf("état nul")
	case s.Turn != o.Turn:
		return fmt.Errorf("tour : %d, attendu %d", o.Turn, s.Turn)
	case s.Width != o.Width || s.Height != o.Height:
		return fmt.Errorf("dimensions : %dx%d, attendu %dx%d", o.Width, o.Height, s.Width, s.Height)
	}
	for i := range s.Fire {
		if s.Terrain[i] != o.Terrain[i] {
			return fmt.Errorf("case %d : terrain %d, attendu %d", i, o.Terrain[i], s.Terrain[i])
		}
		if s.Wind[i] != o.Wind[i] {
			return fmt.Errorf("case %d : vent %d, attendu %d", i, o.Wind[i], s.Wind[i])
		}
		if s.Fire[i] != o.Fire[i] {
			return fmt.Errorf("case %d : feu %d, attendu %d", i, o.Fire[i], s.Fire[i])
		}
		if s.Rest[i] != o.Rest[i] {
			return fmt.Errorf("case %d : repos %d, attendu %d", i, o.Rest[i], s.Rest[i])
		}
	}
	return nil
}

// Format sérialise et désérialise un State.
type Format interface {
	// Ext est l'extension de fichier associée, sans le point.
	Ext() string
	Write(w io.Writer, s *State) error
	Read(r io.Reader) (*State, error)
}

var registry = map[string]Format{}

// Register enregistre un format sous un nom (ex : "json", "proto", "packed").
func Register(name string, f Format) {
	if _, dup := registry[name]; dup {
		panic("snapshot: format enregistré deux fois : " + name)
	}
	registry[name] = f
}

// Get renvoie un format par son nom.
func Get(name string) (Format, bool) {
	f, ok := registry[name]
	return f, ok
}

// Names renvoie les formats enregistrés, triés.
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// errHorsGrille signale une case dont les coordonnées sortent de la grille
// déclarée par l'en-tête : un fichier tronqué ou corrompu.
func errHorsGrille(x, y int) error {
	return fmt.Errorf("snapshot: case (%d,%d) hors de la grille déclarée", x, y)
}
