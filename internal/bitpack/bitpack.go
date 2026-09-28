// Package bitpack traite les états et la propagation sans vent par mots de 64 bits.
package bitpack

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"gol-wildfire/internal/fire"
	"math/bits"
	"strings"
)

type target struct {
	word int
	mask uint64
}
type windy struct {
	source  target
	targets [6]target
}

// Sim conserve quatre plans d'état (poids 1 et 2 de feu et repos).
type Sim struct {
	carte                               fire.Map
	words                               int
	lastMask                            uint64
	lastBit                             uint
	f1, f2, r1, r2                      []uint64
	plain, forest, wind, source, ignite []uint64
	winds                               []windy
	burning                             int
}

func New(m fire.Map) *Sim {
	s := &Sim{carte: m, words: (m.Width + 63) / 64, lastBit: uint((m.Width - 1) & 63)}
	s.lastMask = ^uint64(0) >> (63 - s.lastBit)
	n := s.words * m.Height
	s.f1 = make([]uint64, n)
	s.f2 = make([]uint64, n)
	s.r1 = make([]uint64, n)
	s.r2 = make([]uint64, n)
	s.plain = make([]uint64, n)
	s.forest = make([]uint64, n)
	s.wind = make([]uint64, n)
	s.source = make([]uint64, n)
	s.ignite = make([]uint64, n)
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			i := y*m.Width + x
			t := s.at(x, y)
			switch m.Terrain[i] {
			case fire.Plain:
				s.plain[t.word] |= t.mask
			case fire.Forest:
				s.forest[t.word] |= t.mask
			}
			if m.Wind[i] == 0 {
				continue
			}
			s.wind[t.word] |= t.mask
			w := windy{source: t}
			d := int(m.Wind[i]) - 1
			k := 0
			for j, v := range fire.Offsets {
				sector := (j - d) & 7
				if sector >= 3 && sector <= 5 {
					continue
				}
				w.targets[k] = s.at(fire.Mod(x+v[0], m.Width), fire.Mod(y+v[1], m.Height))
				k++
			}
			v := fire.Offsets[d]
			w.targets[5] = s.at(fire.Mod(x+2*v[0], m.Width), fire.Mod(y+2*v[1], m.Height))
			s.winds = append(s.winds, w)
		}
	}
	for _, i := range m.Fires {
		t := s.at(i%m.Width, i/m.Width)
		if (s.f1[t.word]|s.f2[t.word])&t.mask != 0 {
			continue
		}
		switch m.Terrain[i] {
		case fire.Plain:
			s.f1[t.word] |= t.mask
			s.burning++
		case fire.Forest:
			s.f2[t.word] |= t.mask
			s.burning++
		}
	}
	return s
}
func (s *Sim) at(x, y int) target { return target{y*s.words + (x >> 6), uint64(1) << uint(x&63)} }
func (s *Sim) Map() fire.Map      { return s.carte }
func (s *Sim) Width() int         { return s.carte.Width }
func (s *Sim) Height() int        { return s.carte.Height }
func (s *Sim) Fire(x, y int) uint8 {
	t := s.at(x, y)
	var v uint8
	if s.f1[t.word]&t.mask != 0 {
		v = 1
	}
	if s.f2[t.word]&t.mask != 0 {
		v |= 2
	}
	return v
}
func (s *Sim) Rest(x, y int) uint8 {
	t := s.at(x, y)
	var v uint8
	if s.r1[t.word]&t.mask != 0 {
		v = 1
	}
	if s.r2[t.word]&t.mask != 0 {
		v |= 2
	}
	return v
}
func (s *Sim) Burning() int { return s.burning }

// spread fusionne les décalages horizontaux -1, 0, +1 d'une ligne source.
// Le bouclage utilise la largeur réelle, jamais les bits de remplissage.
func (s *Sim) spread(src, dst []uint64) {
	last := len(src) - 1
	for i, v := range src {
		left, right := v<<1, v>>1
		if i > 0 {
			left |= src[i-1] >> 63
		} else {
			left |= (src[last] >> s.lastBit) & 1
		}
		if i < last {
			right |= src[i+1] << 63
		} else {
			right |= (src[0] & 1) << s.lastBit
		}
		dst[i] |= v | left | right
	}
	dst[last] &= s.lastMask
}

func (s *Sim) Step() {
	clear(s.ignite)
	for i := range s.source {
		s.source[i] = (s.f1[i] | s.f2[i]) &^ s.wind[i]
	}
	for y := 0; y < s.carte.Height; y++ {
		prev, next := y-1, y+1
		if prev < 0 {
			prev = s.carte.Height - 1
		}
		if next == s.carte.Height {
			next = 0
		}
		row := s.source[y*s.words : (y+1)*s.words]
		s.spread(row, s.ignite[prev*s.words:(prev+1)*s.words])
		s.spread(row, s.ignite[y*s.words:(y+1)*s.words])
		s.spread(row, s.ignite[next*s.words:(next+1)*s.words])
	}
	// Le centre est inclus par spread mais une case déjà en feu ignore l'ignition.
	for _, w := range s.winds {
		if (s.f1[w.source.word]|s.f2[w.source.word])&w.source.mask == 0 {
			continue
		}
		for _, t := range w.targets {
			s.ignite[t.word] |= t.mask
		}
	}
	count := 0
	for i := range s.f1 {
		f1, f2, r1, r2 := s.f1[i], s.f2[i], s.r1[i], s.r2[i]
		fresh := s.ignite[i] &^ (f1 | f2 | r1 | r2)
		// 2 -> 1 -> 0 pour le feu. Un feu à 1 expire ce tour-ci.
		s.f1[i] = f2 | (fresh & s.plain[i])
		s.f2[i] = fresh & s.forest[i]
		// Repos : 3 -> 2 -> 1 -> 0 ; extinction : plaine=2, forêt=3.
		s.r1[i] = (r2 &^ r1) | (f1 & s.forest[i])
		s.r2[i] = (r2 & r1) | f1
		count += bits.OnesCount64(s.f1[i] | s.f2[i])
	}
	s.burning = count
}

// Fingerprint conserve le contrat et le formatage des variantes précédentes.
func (s *Sim) Fingerprint() uint64 {
	var sb strings.Builder
	for y := 0; y < s.Height(); y++ {
		for x := 0; x < s.Width(); x++ {
			f, r := s.Fire(x, y), s.Rest(x, y)
			if f > 0 || r > 0 {
				sb.WriteString(fmt.Sprintf("%d,%d,%d,%d;", x, y, f, r))
			}
		}
	}
	sum := sha256.Sum256([]byte(sb.String()))
	return binary.LittleEndian.Uint64(sum[:8])
}
func init() { fire.Register("bitpack", func(m fire.Map) fire.Engine { return New(m) }) }
