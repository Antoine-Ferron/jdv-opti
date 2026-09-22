// Package flat isole la grille contiguë et la réutilisation de deux buffers.
// Les modulos et l'empreinte de naive sont conservés pour cette comparaison
// contrôlée ; leurs allocations et coûts ne sont pas optimisés à cette étape.
package flat

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"

	"gol/internal/life"
)

func init() {
	life.Register("flat", func(w, h int, cells []bool) life.Engine { return New(w, h, cells) })
}

// Sim conserve les mêmes métadonnées que naive et possède deux grilles plates.
type Sim struct {
	wrap       bool
	width      int
	changed    bool
	height     int
	extinct    bool
	generation int
	grid       []bool
	next       []bool
}

// New construit la simulation à partir d'un état plat cells[y*w+x].
func New(w, h int, cells []bool) *Sim {
	s := &Sim{wrap: true, width: w, height: h}
	s.grid = make([]bool, w*h)
	s.next = make([]bool, w*h)
	copy(s.grid, cells[:w*h])
	return s
}

func (s *Sim) Width() int  { return s.width }
func (s *Sim) Height() int { return s.height }

func (s *Sim) Alive(x, y int) bool { return s.grid[y*s.width+x] }

// neighbors conserve le calcul torique de naive.
func (s *Sim) neighbors(x, y int) int {
	n := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if s.wrap {
				nx = (nx + s.width) % s.width
				ny = (ny + s.height) % s.height
			} else if nx < 0 || ny < 0 || nx >= s.width || ny >= s.height {
				continue
			}
			if s.grid[ny*s.width+nx] {
				n++
			}
		}
	}
	return n
}

// Step calcule la génération suivante.
func (s *Sim) Step() {
	next := s.next
	s.changed = false
	for y := 0; y < s.height; y++ {
		for x := 0; x < s.width; x++ {
			n := s.neighbors(x, y)
			alive := s.grid[y*s.width+x]
			nextAlive := n == 3 || (alive && n == 2)
			next[y*s.width+x] = nextAlive
			if nextAlive != alive {
				s.changed = true
			}
		}
	}
	s.grid, s.next = next, s.grid
	s.generation++
}

// Population compte les cellules vivantes.
func (s *Sim) Population() int {
	p := 0
	for y := 0; y < s.height; y++ {
		for x := 0; x < s.width; x++ {
			if s.grid[y*s.width+x] {
				p++
			}
		}
	}
	s.extinct = p == 0
	return p
}

// Fingerprint conserve volontairement le formatage et les allocations de naive.
func (s *Sim) Fingerprint() uint64 {
	var sb strings.Builder
	for y := 0; y < s.height; y++ {
		for x := 0; x < s.width; x++ {
			if s.grid[y*s.width+x] {
				sb.WriteString(fmt.Sprintf("%d,%d;", x, y))
			}
		}
	}
	sum := sha256.Sum256([]byte(sb.String()))
	return binary.LittleEndian.Uint64(sum[:8])
}
