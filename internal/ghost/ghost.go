// Package ghost propage dans un halo de deux cases, puis replie les ignitions.
// Fingerprint reste identique à counters pour isoler cette étape.
package ghost

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
	carte   fire.Map
	grid    []Cell
	next    []Cell
	ignite  []bool
	burning int
	stride  int
	offsets [8]int
	border  [][2]int
}

// New construit la simulation et allume les foyers de départ de la carte (§6).
func New(m fire.Map) *Sim {
	s := &Sim{carte: m}
	s.grid = make([]Cell, m.Width*m.Height)
	s.next = make([]Cell, m.Width*m.Height)
	s.stride = m.Width + 4
	s.ignite = make([]bool, s.stride*(m.Height+4))
	for d, v := range fire.Offsets {
		s.offsets[d] = v[1]*s.stride + v[0]
	}
	// Seuls les indices du halo sont mémorisés ; les cibles sont toujours intérieures.
	for y := 0; y < m.Height+4; y++ {
		for x := 0; x < s.stride; x++ {
			if y >= 2 && y < m.Height+2 && x >= 2 && x < m.Width+2 {
				continue
			}
			target := (fire.Mod(y-2, m.Height)+2)*s.stride + fire.Mod(x-2, m.Width) + 2
			s.border = append(s.border, [2]int{y*s.stride + x, target})
		}
	}
	for _, i := range m.Fires {
		x, y := i%m.Width, i/m.Width
		c := &s.grid[y*s.carte.Width+x]
		if c.feu == 0 && m.Terrain[i].Combustion() > 0 {
			s.burning++ // Un foyer dupliqué ou sur l'eau ne compte pas deux fois.
		}
		c.feu = m.Terrain[i].Combustion()
	}
	return s
}

func (s *Sim) Map() fire.Map { return s.carte }

func (s *Sim) Width() int  { return s.carte.Width }
func (s *Sim) Height() int { return s.carte.Height }

func (s *Sim) Fire(x, y int) uint8 { return s.grid[y*s.carte.Width+x].feu }
func (s *Sim) Rest(x, y int) uint8 { return s.grid[y*s.carte.Width+x].repos }

// Burning renvoie le compteur maintenu par Step, sans parcourir la grille.
func (s *Sim) Burning() int { return s.burning }

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
			i := y*s.carte.Width + x
			center := (y+2)*s.stride + x + 2
			wind := s.carte.Wind[i]
			d := int(wind) - 1
			for j, offset := range s.offsets {
				if wind != 0 {
					sector := (j - d) & 7
					if sector >= 3 && sector <= 5 {
						continue
					}
				}
				ignite[center+offset] = true
			}
			if wind != 0 {
				ignite[center+2*s.offsets[d]] = true
			}
		}
	}
	// Plusieurs sources et coins peuvent viser la même case : fusion par OU.
	for _, pair := range s.border {
		if ignite[pair[0]] {
			ignite[pair[1]] = true
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
					s.burning--
				}
			case c.repos > 0:
				c.repos--
			case ignite[(y+2)*s.stride+x+2] && terrain.Combustion() > 0:
				c.feu = terrain.Combustion()
				s.burning++
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
	fire.Register("ghost", func(m fire.Map) fire.Engine { return New(m) })
}
