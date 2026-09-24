package snapshot

import (
	"encoding/binary"
	"fmt"
	"io"

	"gol-wildfire/internal/fire"
)

// Packed est le format bit-packé : l'aboutissement de l'axe I/O.
//
// Il lève les deux limites que Protobuf ne pouvait pas franchir :
//
//	[S5] Un état de case tient sur 4 bits — feu ≤ 2 et repos ≤ 3 occupent
//	     chacun 2 bits (REGLES.md §2) — au lieu d'un octet minimum par varint.
//	     Deux cases par octet.
//	[S2] Le décor est immuable pour toute la partie : il n'est écrit que dans le
//	     premier snapshot d'une série, les suivants ne portent que l'état.
//
// Le décor lui-même est bit-packé : terrain sur 2 bits (3 valeurs) et vent sur
// 4 bits (0 = aucun, sinon 1+direction parmi 8), soit 6 bits par case, arrondis
// à un octet pour garder un décodage trivial.
//
// Format, en entier non signé petit-boutiste :
//
//	magie   uint32  "WFS1"
//	turn    uint32
//	width   uint32
//	height  uint32
//	flags   uint8   bit 0 : le décor est présent
//	[décor] width*height octets : terrain (2 bits) | vent (4 bits) << 2
//	état    ceil(width*height/2) octets : feu (2 bits) | repos (2 bits) << 2,
//	        case paire dans les bits de poids faible
type Packed struct {
	// SansDecor demande d'omettre le décor : à réserver aux snapshots qui
	// suivent un premier, dans une même série.
	SansDecor bool
}

func init() { Register("packed", Packed{}) }

const (
	packedMagie   = 0x31534657 // "WFS1" en petit-boutiste
	packedAvecDec = 1 << 0
)

func (Packed) Ext() string { return "wfs" }

func (p Packed) Write(w io.Writer, s *State) error {
	n := s.Width * s.Height
	if n == 0 {
		return fmt.Errorf("snapshot: grille vide")
	}

	// Tableau et non tranche : l'en-tête ne quitte pas la pile, donc n'alloue
	// pas du tout — inutile de le faire passer par un pool pour 17 octets.
	var entete [17]byte
	binary.LittleEndian.PutUint32(entete[0:], packedMagie)
	binary.LittleEndian.PutUint32(entete[4:], uint32(s.Turn))
	binary.LittleEndian.PutUint32(entete[8:], uint32(s.Width))
	binary.LittleEndian.PutUint32(entete[12:], uint32(s.Height))
	if !p.SansDecor {
		entete[16] = packedAvecDec
	}
	if _, err := w.Write(entete[:]); err != nil {
		return err
	}

	if !p.SansDecor {
		decor := prendreOctets(n)
		defer rendreOctets(decor)
		// Chaque octet est affecté, jamais combiné : le tampon repris est donc
		// intégralement recouvert.
		for i := 0; i < n; i++ {
			decor[i] = byte(s.Terrain[i])&0x03 | s.Wind[i]<<2
		}
		if _, err := w.Write(decor); err != nil {
			return err
		}
	}

	// Deux cases par octet : la paire dans les bits 0-3, l'impaire dans 4-7.
	//
	// L'indice pair *affecte* l'octet avant que l'impair ne l'enrichisse ; tout
	// octet est donc écrit au moins une fois, y compris le dernier quand n est
	// impair. C'est ce qui rend le tampon recyclé sûr — inverser les deux
	// branches laisserait fuiter le snapshot précédent.
	etat := prendreOctets((n + 1) / 2)
	defer rendreOctets(etat)
	for i := 0; i < n; i++ {
		quartet := s.Fire[i]&0x03 | s.Rest[i]&0x03<<2
		if i%2 == 0 {
			etat[i/2] = quartet
		} else {
			etat[i/2] |= quartet << 4
		}
	}
	_, err := w.Write(etat)
	return err
}

func (p Packed) Read(r io.Reader) (*State, error) {
	entete := make([]byte, 17)
	if _, err := io.ReadFull(r, entete); err != nil {
		return nil, fmt.Errorf("snapshot: en-tête illisible : %w", err)
	}
	if binary.LittleEndian.Uint32(entete) != packedMagie {
		return nil, fmt.Errorf("snapshot: ce n'est pas un fichier wfs")
	}
	s := &State{
		Turn:   int(binary.LittleEndian.Uint32(entete[4:])),
		Width:  int(binary.LittleEndian.Uint32(entete[8:])),
		Height: int(binary.LittleEndian.Uint32(entete[12:])),
	}
	n := s.Width * s.Height
	if n <= 0 {
		return nil, fmt.Errorf("snapshot: dimensions invalides %dx%d", s.Width, s.Height)
	}

	s.Terrain = make([]fire.Terrain, n)
	s.Wind = make([]uint8, n)
	if entete[16]&packedAvecDec != 0 {
		decor := make([]byte, n)
		if _, err := io.ReadFull(r, decor); err != nil {
			return nil, fmt.Errorf("snapshot: décor illisible : %w", err)
		}
		for i, b := range decor {
			s.Terrain[i] = fire.Terrain(b & 0x03)
			s.Wind[i] = b >> 2
		}
	}

	etat := make([]byte, (n+1)/2)
	if _, err := io.ReadFull(r, etat); err != nil {
		return nil, fmt.Errorf("snapshot: état illisible : %w", err)
	}
	s.Fire = make([]uint8, n)
	s.Rest = make([]uint8, n)
	for i := 0; i < n; i++ {
		quartet := etat[i/2]
		if i%2 == 1 {
			quartet >>= 4
		}
		s.Fire[i] = quartet & 0x03
		s.Rest[i] = quartet >> 2 & 0x03
	}
	return s, nil
}
