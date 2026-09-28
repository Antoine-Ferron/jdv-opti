// Package front ne visite que les cases en feu, en repos et nouvellement allumées.
package front

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"gol-wildfire/internal/fire"
	"strings"
)

type Cell struct{ feu, repos uint8 }
type point struct{ x, y int }
type Sim struct {
	carte   fire.Map
	grid    []Cell
	marked  []bool
	active  []point
	resting []int
	pending []point
}

func New(m fire.Map) *Sim {
	n := m.Width * m.Height
	s := &Sim{carte: m, grid: make([]Cell, n), marked: make([]bool, n),
		active: make([]point, 0, n), resting: make([]int, 0, n), pending: make([]point, 0, n)}
	for _, i := range m.Fires {
		if s.grid[i].feu != 0 {
			continue
		}
		f := m.Terrain[i].Combustion()
		if f == 0 {
			continue
		}
		s.grid[i].feu = f
		s.active = append(s.active, point{i % m.Width, i / m.Width})
	}
	return s
}
func (s *Sim) Map() fire.Map       { return s.carte }
func (s *Sim) Width() int          { return s.carte.Width }
func (s *Sim) Height() int         { return s.carte.Height }
func (s *Sim) Fire(x, y int) uint8 { return s.grid[y*s.carte.Width+x].feu }
func (s *Sim) Rest(x, y int) uint8 { return s.grid[y*s.carte.Width+x].repos }
func (s *Sim) Burning() int        { return len(s.active) }

// Les coordonnées ne dépassent les bords que de deux cases (saut de vent).
// Les boucles gèrent aussi les dimensions 1 et 2 sans division.
func wrap(x, n int) int {
	for x < 0 {
		x += n
	}
	for x >= n {
		x -= n
	}
	return x
}
func (s *Sim) mark(x, y int) {
	x = wrap(x, s.Width())
	y = wrap(y, s.Height())
	i := y*s.Width() + x
	c := s.grid[i]
	if s.marked[i] || c.feu > 0 || c.repos > 0 || s.carte.Terrain[i].Combustion() == 0 {
		return
	}
	s.marked[i] = true
	// Au plus N cases distinctes, capacité réservée par New.
	s.pending = append(s.pending, point{x, y})
}
func (s *Sim) Step() {
	// Toutes les cibles sont sélectionnées dans l'ancien état.
	for _, p := range s.active {
		wind := s.carte.Wind[p.y*s.Width()+p.x]
		d := int(wind) - 1
		for j, v := range fire.Offsets {
			if wind != 0 {
				sector := (j - d) & 7
				if sector >= 3 && sector <= 5 {
					continue
				}
			}
			s.mark(p.x+v[0], p.y+v[1])
		}
		if wind != 0 {
			v := fire.Offsets[d]
			s.mark(p.x+2*v[0], p.y+2*v[1])
		}
	}
	// Compacter les anciens repos avant d'ajouter les nouvelles extinctions.
	count := 0
	for _, i := range s.resting {
		s.grid[i].repos--
		if s.grid[i].repos > 0 {
			s.resting[count] = i
			count++
		}
	}
	s.resting = s.resting[:count]
	count = 0
	for _, p := range s.active {
		i := p.y*s.Width() + p.x
		s.grid[i].feu--
		if s.grid[i].feu > 0 {
			s.active[count] = p
			count++
		} else {
			s.grid[i].repos = s.carte.Terrain[i].Repos()
			s.resting = append(s.resting, i)
		}
	}
	s.active = s.active[:count]
	for _, p := range s.pending {
		i := p.y*s.Width() + p.x
		s.grid[i].feu = s.carte.Terrain[i].Combustion()
		s.active = append(s.active, p)
		s.marked[i] = false
	}
	s.pending = s.pending[:0]
}

// Fingerprint — [F6] une chaîne "x,y,feu,repos;" par case active, puis SHA-256.
func (s *Sim) Fingerprint() uint64 {
	var sb strings.Builder
	for y := 0; y < s.carte.Height; y++ {
		for x := 0; x < s.carte.Width; x++ {
			c := s.grid[y*s.carte.Width+x]
			if c.feu > 0 || c.repos > 0 {
				sb.WriteString(fmt.Sprintf("%d,%d,%d,%d;", x, y, c.feu, c.repos))
			}
		}
	}
	sum := sha256.Sum256([]byte(sb.String()))
	return binary.LittleEndian.Uint64(sum[:8])
}

func init() {
	fire.Register("front", func(m fire.Map) fire.Engine { return New(m) })
}
