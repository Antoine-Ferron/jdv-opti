package snapshot

import (
	"encoding/json"
	"io"

	"gol-wildfire/internal/fire"
)

// JSON est la BASELINE de l'axe I/O : le format qu'on écrirait spontanément,
// sans se soucier de ce qu'il coûte.
//
// Défauts volontaires (cibles d'optimisation, à comparer aux formats suivants) :
//
//	[S1] Une structure par case, avec ses champs nommés : le nom de chaque
//	     champ est réécrit pour chacune des cases, soit des dizaines d'octets
//	     là où l'état tient sur 4 bits.
//	[S2] Le décor (terrain, vent) est réécrit à chaque snapshot alors qu'il est
//	     immuable pour toute la partie.
//	[S3] Encodage texte : un entier de 0 à 3 occupe un caractère, plus les
//	     séparateurs et l'indentation.
//	[S4] Tout l'état est matérialisé en mémoire avant d'être écrit.
type JSON struct{}

func init() { Register("json", JSON{}) }

func (JSON) Ext() string { return "json" }

// cellJSON est l'état d'une seule case, avec des noms de champs explicites.
type cellJSON struct {
	X       int `json:"x"`
	Y       int `json:"y"`
	Terrain int `json:"terrain"`
	Wind    int `json:"wind"`
	Fire    int `json:"fire"`
	Rest    int `json:"rest"`
}

type docJSON struct {
	Turn   int        `json:"turn"`
	Width  int        `json:"width"`
	Height int        `json:"height"`
	Cells  []cellJSON `json:"cells"`
}

func (JSON) Write(w io.Writer, s *State) error {
	doc := docJSON{Turn: s.Turn, Width: s.Width, Height: s.Height}
	doc.Cells = make([]cellJSON, 0, len(s.Fire)) // [S4]
	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			i := y*s.Width + x
			doc.Cells = append(doc.Cells, cellJSON{ // [S1] [S2]
				X: x, Y: y,
				Terrain: int(s.Terrain[i]),
				Wind:    int(s.Wind[i]),
				Fire:    int(s.Fire[i]),
				Rest:    int(s.Rest[i]),
			})
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ") // [S3]
	return enc.Encode(doc)
}

func (JSON) Read(r io.Reader) (*State, error) {
	var doc docJSON
	if err := json.NewDecoder(r).Decode(&doc); err != nil {
		return nil, err
	}
	n := doc.Width * doc.Height
	s := &State{
		Turn: doc.Turn, Width: doc.Width, Height: doc.Height,
		Terrain: make([]fire.Terrain, n),
		Wind:    make([]uint8, n),
		Fire:    make([]uint8, n),
		Rest:    make([]uint8, n),
	}
	for _, c := range doc.Cells {
		i := c.Y*doc.Width + c.X
		if i < 0 || i >= n {
			return nil, errHorsGrille(c.X, c.Y)
		}
		s.Terrain[i] = fire.Terrain(c.Terrain)
		s.Wind[i] = uint8(c.Wind)
		s.Fire[i] = uint8(c.Fire)
		s.Rest[i] = uint8(c.Rest)
	}
	return s, nil
}
