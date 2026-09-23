// Package flat utilise deux grilles contiguës et un tampon d’allumage réutilisé.
// Les modulos et le formatage de Fingerprint sont conservés pour isoler cette étape.
package flat

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
	carte  fire.Map
	grid   []Cell
	next   []Cell
	ignite []bool
}

// New construit la simulation et allume les foyers de départ de la carte (§6).
func New(m fire.Map) *Sim {
	s := &Sim{carte: m}
	s.grid = make([]Cell, m.Width*m.Height)
	s.next = make([]Cell, m.Width*m.Height)
	s.ignite = make([]bool, m.Width*m.Height)
	for _, i := range m.Fires {
		x, y := i%m.Width, i/m.Width
		s.grid[y*s.carte.Width+x].feu = m.Terrain[i].Combustion()
	}
	return s
}

func (s *Sim) Map() fire.Map { return s.carte }

func (s *Sim) Width() int  { return s.carte.Width }
func (s *Sim) Height() int { return s.carte.Height }

func (s *Sim) Fire(x, y int) uint8 { return s.grid[y*s.carte.Width+x].feu }
func (s *Sim) Rest(x, y int) uint8 { return s.grid[y*s.carte.Width+x].repos }

// Burning recompte les cases en feu à chaque appel.
func (s *Sim) Burning() int {
	n := 0
	for y := 0; y < s.carte.Height; y++ {
		for x := 0; x < s.carte.Width; x++ {
			if s.grid[y*s.carte.Width+x].feu > 0 {
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
	ignite := s.ignite
	clear(ignite) // Les contagions ne survivent pas au tour courant.
	for y := 0; y < s.carte.Height; y++ {
		for x := 0; x < s.carte.Width; x++ {
			if s.grid[y*s.carte.Width+x].feu == 0 {
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

	for y := 0; y < s.carte.Height; y++ {
		for x := 0; x < s.carte.Width; x++ {
			i := y*s.carte.Width + x
			terrain := s.carte.Terrain[i]
			c := s.grid[i]
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
			s.next[i] = c // Écrire chaque cellule, y compris inactive : aucun état périmé.
		}
	}
	s.grid, s.next = s.next, s.grid
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
	fire.Register("flat", func(m fire.Map) fire.Engine { return New(m) })
}
