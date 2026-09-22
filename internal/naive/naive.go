// Package naive est la BASELINE de la simulation d'incendie : écrite « comme on
// l'écrirait spontanément », sans souci de performance. Elle ne doit plus être
// modifiée une fois les mesures de référence prises : toute optimisation se fait
// dans un NOUVEAU package.
//
// Les règles qu'elle applique sont dans internal/fire/REGLES.md ; les numéros de
// section en commentaire y renvoient.
//
// Défauts volontaires (cibles d'optimisation, à relier aux profils pprof) :
//
//	[F1] Grille [][]Cell : un slice par ligne -> lignes dispersées sur le tas,
//	     double indirection, la ligne y-1 n'est pas contiguë à la ligne y.
//	[F2] Cell occupe 2 octets alors que l'état tient sur 4 bits (feu ≤ 2, repos
//	     ≤ 3) : 4x trop de mémoire touchée, donc 4x trop de lignes de cache.
//	[F3] Un tampon d'ignition alloué à chaque tour -> pression GC inutile,
//	     alors qu'un seul tampon réutilisé suffirait.
//	[F4] Balayage intégral de la carte pour la propagation, alors que seules les
//	     cases EN FEU propagent : en début de partie, quelques dizaines sur un
//	     million. C'est le gisement propre à ce modèle.
//	[F5] 8 modulos par case en feu (plus un pour le saut du vent) : division
//	     entière là où un masque ou une bordure fantôme suffirait.
//	[F6] Fingerprint : un fmt.Sprintf par case active puis SHA-256 -> des
//	     centaines de milliers d'allocations par appel.
//	[F7] Aucune concurrence, alors que la propagation est un OU logique
//	     (REGLES.md §4) : associative, donc découpable en blocs sans verrou.
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

func (s *Sim) Map() fire.Map { return s.carte }

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

func init() {
	fire.Register("naive", func(m fire.Map) fire.Engine { return New(m) })
}
