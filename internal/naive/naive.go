// Package naive est la BASELINE : écrite "comme on l'écrirait spontanément",
// sans souci de performance. Elle ne doit plus être modifiée une fois les mesures
// de référence prises : toute optimisation se fait dans un NOUVEAU package.
//
// Défauts volontaires (cibles d'optimisation, à relier aux profils pprof) :
//
//	[D1] Grille [][]bool : un slice par ligne -> lignes dispersées sur le tas,
//	     double indirection, 1 octet par cellule (8x plus que nécessaire).
//	[D2] Step alloue une nouvelle grille complète à chaque génération -> pression GC.
//	[D3] Comptage des voisins : 8 modulos + branchements par voisin, pour chaque cellule.
//	[D4] Fingerprint : fmt.Sprintf par cellule vivante + conversion string -> []byte
//	     superflue -> des centaines de milliers d'allocations par génération.
//	[D5] Struct Sim mal ordonnée : champs bool intercalés entre des int -> padding.
//	[D6] Aucune concurrence : un seul cœur utilisé.
package naive

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"

	"gol/internal/life"
)

func init() {
	life.Register("naive", func(w, h int, cells []bool) life.Engine { return New(w, h, cells) })
}

// Sim — [D5] ordre des champs non optimisé (vérifier avec unsafe.Sizeof).
type Sim struct {
	wrap       bool
	width      int
	changed    bool
	height     int
	extinct    bool
	generation int
	grid       [][]bool
}

// New construit la simulation à partir d'un état plat cells[y*w+x].
func New(w, h int, cells []bool) *Sim {
	s := &Sim{wrap: true, width: w, height: h}
	s.grid = make([][]bool, h) // [D1]
	for y := 0; y < h; y++ {
		s.grid[y] = make([]bool, w)
		for x := 0; x < w; x++ {
			s.grid[y][x] = cells[y*w+x]
		}
	}
	return s
}

func (s *Sim) Width() int  { return s.width }
func (s *Sim) Height() int { return s.height }

func (s *Sim) Alive(x, y int) bool { return s.grid[y][x] }

// neighbors compte les voisins vivants de (x, y). [D3]
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
			if s.grid[ny][nx] {
				n++
			}
		}
	}
	return n
}

// Step calcule la génération suivante.
func (s *Sim) Step() {
	next := make([][]bool, s.height) // [D2] nouvelle grille à chaque génération
	s.changed = false
	for y := 0; y < s.height; y++ {
		next[y] = make([]bool, s.width)
		for x := 0; x < s.width; x++ {
			n := s.neighbors(x, y)
			alive := s.grid[y][x]
			nextAlive := n == 3 || (alive && n == 2)
			next[y][x] = nextAlive
			if nextAlive != alive {
				s.changed = true
			}
		}
	}
	s.grid = next
	s.generation++
}

// Population compte les cellules vivantes.
func (s *Sim) Population() int {
	p := 0
	for y := 0; y < s.height; y++ {
		for x := 0; x < s.width; x++ {
			if s.grid[y][x] {
				p++
			}
		}
	}
	s.extinct = p == 0
	return p
}

// Fingerprint — [D4] une chaîne "x,y;" par cellule vivante, puis SHA-256.
func (s *Sim) Fingerprint() uint64 {
	var sb strings.Builder
	for y := 0; y < s.height; y++ {
		for x := 0; x < s.width; x++ {
			if s.grid[y][x] {
				sb.WriteString(fmt.Sprintf("%d,%d;", x, y))
			}
		}
	}
	sum := sha256.Sum256([]byte(sb.String()))
	return binary.LittleEndian.Uint64(sum[:8])
}
