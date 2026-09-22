// Package naive est la BASELINE de la simulation d'incendie : écrite « comme on
// l'écrirait spontanément », sans souci de performance. Elle ne doit plus être
// modifiée une fois les mesures de référence prises : toute optimisation se fait
// dans un NOUVEAU package.
package naive

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"

	"gol-wildfire/internal/fire"
)

// Cell est l'état d'une case (§3).
type Cell struct {
	feu   uint8
	repos uint8
}

// Sim est la simulation.
type Sim struct {
	carte fire.Map
	grid  [][]Cell
}

// New construit la simulation et allume les foyers de départ de la carte (§6).
func New(m fire.Map) *Sim {
	s := &Sim{carte: m}
	s.grid = make([][]Cell, m.Height)
	for y := range s.grid {
		s.grid[y] = make([]Cell, m.Width)
	}
	for _, i := range m.Fires {
		x, y := i%m.Width, i/m.Width
		s.grid[y][x].feu = m.Terrain[i].Combustion()
	}
	return s
}

func (s *Sim) Width() int  { return s.carte.Width }
func (s *Sim) Height() int { return s.carte.Height }

func (s *Sim) Fire(x, y int) uint8 { return s.grid[y][x].feu }
func (s *Sim) Rest(x, y int) uint8 { return s.grid[y][x].repos }

// Burning recompte les cases en feu à chaque appel.
func (s *Sim) Burning() int {
	n := 0
	for y := range s.grid {
		for x := range s.grid[y] {
			if s.grid[y][x].feu > 0 {
				n++
			}
		}
	}
	return n
}

// Step calcule le tour suivant.
//
// La propagation est intégralement calculée avant d'être appliquée : toutes les
// cases voient donc le même état, celui du tour précédent (§1).
func (s *Sim) Step() {
	ignite := make([]bool, s.carte.Width*s.carte.Height)
	for y := range s.grid {
		for x := range s.grid[y] {
			if s.grid[y][x].feu == 0 {
				continue
			}
			// Une case vent épargne son secteur amont et projette le feu deux
			// cases sous le vent, par-dessus ce qui s'y trouve (§5).
			d, vente := s.carte.WindAt(x, y)
			for j, v := range fire.Offsets {
				if vente {
					if secteur := fire.Mod(j-int(d), 8); secteur >= 3 && secteur <= 5 {
						continue
					}
				}
				ignite[s.carte.At(x+v[0], y+v[1])] = true
			}
			if vente {
				v := fire.Offsets[d]
				ignite[s.carte.At(x+2*v[0], y+2*v[1])] = true
			}
		}
	}

	for y := range s.grid {
		for x := range s.grid[y] {
			i := y*s.carte.Width + x
			terrain := s.carte.Terrain[i]
			c := &s.grid[y][x]
			switch {
			case c.feu > 0:
				c.feu--
				if c.feu == 0 {
					c.repos = terrain.Repos()
				}
			case c.repos > 0:
				c.repos--
			case ignite[i] && terrain.Combustion() > 0:
				c.feu = terrain.Combustion()
			}
		}
	}
}

// Fingerprint — [F6] une chaîne "x,y,feu,repos;" par case active, puis SHA-256.
func (s *Sim) Fingerprint() uint64 {
	var sb strings.Builder
	for y := range s.grid {
		for x, c := range s.grid[y] {
			if c.feu > 0 || c.repos > 0 {
				sb.WriteString(fmt.Sprintf("%d,%d,%d,%d;", x, y, c.feu, c.repos))
			}
		}
	}
	sum := sha256.Sum256([]byte(sb.String()))
	return binary.LittleEndian.Uint64(sum[:8])
}
