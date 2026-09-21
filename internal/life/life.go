// Package life définit le contrat commun à toutes les implémentations du jeu de la vie.
//
// Chaque version (baseline naïve, puis versions optimisées) implémente Engine et
// s'enregistre via Register. Le binaire, les tests de conformité et les benchmarks
// itèrent sur ce registre : ajouter une version = créer un package + une ligne dans
// internal/engines.
//
// Règles du monde : grille torique (les bords se rejoignent), règle B3/S23.
package life

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
)

// Engine est une simulation du jeu de la vie sur une grille torique.
type Engine interface {
	// Step calcule la génération suivante.
	Step()
	// Alive indique si la cellule (x, y) est vivante. 0 <= x < Width, 0 <= y < Height.
	Alive(x, y int) bool
	// Population renvoie le nombre de cellules vivantes.
	Population() int
	// Fingerprint renvoie une empreinte de l'état courant, utilisée pour détecter
	// les états stables et les cycles (arrêt précoce).
	Fingerprint() uint64
	Width() int
	Height() int
}

// Factory construit un Engine à partir d'un état initial "plat" en ordre ligne
// par ligne : cells[y*w+x].
type Factory func(w, h int, cells []bool) Engine

var registry = map[string]Factory{}

// Register enregistre une implémentation sous un nom (ex : "naive", "flat", "bitpack").
func Register(name string, f Factory) {
	if _, dup := registry[name]; dup {
		panic("life: implémentation enregistrée deux fois : " + name)
	}
	registry[name] = f
}

// Get renvoie la factory d'une implémentation.
func Get(name string) (Factory, bool) {
	f, ok := registry[name]
	return f, ok
}

// Names renvoie les implémentations enregistrées, triées.
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// RandomCells génère un état initial déterministe (même graine => même grille),
// indispensable à la reproductibilité des mesures.
func RandomCells(w, h int, seed int64, density float64) []bool {
	r := rand.New(rand.NewSource(seed))
	cells := make([]bool, w*h)
	for i := range cells {
		cells[i] = r.Float64() < density
	}
	return cells
}

// Result résume une exécution.
type Result struct {
	Generations int // générations effectivement calculées
	Population  int // population finale
	CycleFrom   int // génération où l'état répété a été vu pour la première fois (-1 si aucun)
	Period      int // période du cycle détecté (1 = état stable), 0 si aucun
}

// Options de Run.
type Options struct {
	Generations   int
	DetectCycles  bool   // arrêt précoce dès qu'un état déjà vu réapparaît
	SnapshotDir   string // "" = pas de snapshot
	SnapshotEvery int    // période des snapshots (en générations)
}

// Run exécute la simulation. Version séquentielle et volontairement simple : c'est
// un bon candidat pour l'annulation par context (axe concurrence).
func Run(e Engine, opt Options) (Result, error) {
	res := Result{CycleFrom: -1}
	var seen map[uint64]int
	if opt.DetectCycles {
		seen = make(map[uint64]int)
		seen[e.Fingerprint()] = 0
	}
	for gen := 1; gen <= opt.Generations; gen++ {
		e.Step()
		res.Generations = gen

		if opt.SnapshotDir != "" && opt.SnapshotEvery > 0 && gen%opt.SnapshotEvery == 0 {
			if err := WriteJSONSnapshot(opt.SnapshotDir, gen, e); err != nil {
				return res, err
			}
		}
		if opt.DetectCycles {
			fp := e.Fingerprint()
			if first, ok := seen[fp]; ok {
				res.CycleFrom = first
				res.Period = gen - first
				break
			}
			seen[fp] = gen
		}
	}
	res.Population = e.Population()
	return res, nil
}

// snapshot est le format JSON naïf de sauvegarde d'une grille.
// À comparer plus tard avec Protobuf / un format binaire bit-packé (axe I/O).
type snapshot struct {
	Generation int      `json:"generation"`
	Width      int      `json:"width"`
	Height     int      `json:"height"`
	Cells      [][]bool `json:"cells"`
}

// WriteJSONSnapshot sauvegarde la grille en JSON (baseline I/O, volontairement coûteux).
func WriteJSONSnapshot(dir string, gen int, e Engine) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	s := snapshot{Generation: gen, Width: e.Width(), Height: e.Height()}
	s.Cells = make([][]bool, e.Height())
	for y := range s.Cells {
		s.Cells[y] = make([]bool, e.Width())
		for x := range s.Cells[y] {
			s.Cells[y][x] = e.Alive(x, y)
		}
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(fmt.Sprintf("%s/gen-%06d.json", dir, gen), data, 0o644)
}
