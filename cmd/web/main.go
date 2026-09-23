// Command web affiche la simulation d'incendie dans un navigateur.
//
// Contrairement au CLI, le serveur garde la simulation en mémoire : le contrat
// fire.Engine se construit à partir d'une carte, pas à partir d'un état de
// combustion, et il n'est pas question d'ajouter un setter d'état au contrat
// pour les seuls besoins d'une démo. Le terrain et le vent, immuables, ne sont
// donc envoyés qu'au reset ; ensuite seuls les deux compteurs circulent.
//
// Conséquence assumée : un seul incendie, partagé par tous les onglets. C'est
// une démo locale, pas un service.
package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"sync"
	"time"

	"gol-wildfire/internal/fire"
	"gol-wildfire/internal/naive"
)

//go:embed index.html
var page []byte

// Carte fixe : la page est dimensionnée pour elle.
const (
	width  = 80
	height = 50
)

// presets sont les paysages proposés par la page, tous bâtis sur les réglages
// par défaut (REGLES.md §8).
var presets = map[string]func(*fire.Config){
	"varie": func(c *fire.Config) { c.Fires = 2 },
	"eau":   func(c *fire.Config) { c.Lakes, c.Rivers, c.Fires = 0.18, 0.09, 2 },
	"foret": func(c *fire.Config) { c.Forest, c.Fires = 0.85, 2 },
	"vent":  func(c *fire.Config) { c.Wind, c.Fires = 0.06, 2 },
}

// sim est l'incendie en cours, protégé par son mutex.
type sim struct {
	mu     sync.Mutex
	carte  fire.Map
	engine fire.Engine
	turn   int
}

// reset engendre une nouvelle carte et rallume tout depuis le début.
// À appeler sous verrou.
func (s *sim) reset(preset string, seed int64) bool {
	tweak, ok := presets[preset]
	if !ok {
		return false
	}
	cfg := fire.DefaultConfig(width, height, seed)
	tweak(&cfg)
	s.carte = fire.Generate(cfg)
	s.engine = naive.New(s.carte)
	s.turn = 0
	return true
}

// state relève les deux compteurs de toutes les cases. À appeler sous verrou.
func (s *sim) state() (feu, repos []byte, burning, turn int) {
	feu = make([]byte, width*height)
	repos = make([]byte, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			feu[y*width+x] = s.engine.Fire(x, y)
			repos[y*width+x] = s.engine.Rest(x, y)
		}
	}
	return feu, repos, s.engine.Burning(), s.turn
}

// réponse envoyée à la page. Les champs []byte sont sérialisés en base64 par
// encoding/json, ce que la page décode en une ligne.
type payload struct {
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
	Terrain []byte `json:"terrain,omitempty"`
	Wind    []byte `json:"wind,omitempty"`
	Fire    []byte `json:"fire"`
	Rest    []byte `json:"rest"`
	Turn    int    `json:"turn"`
	Burning int    `json:"burning"`
	Extinct bool   `json:"extinct"`
}

func handler() http.Handler {
	s := &sim{}
	s.reset("varie", 42)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	})
	mux.HandleFunc("POST /api/simulation", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Action string `json:"action"`
			Preset string `json:"preset"`
			Seed   int64  `json:"seed"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Requête invalide", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		var out payload
		switch input.Action {
		case "reset":
			if !s.reset(input.Preset, input.Seed) {
				http.Error(w, "Paysage inconnu", http.StatusBadRequest)
				return
			}
			// Le décor ne change plus jusqu'au prochain reset : il n'est envoyé
			// qu'ici.
			out.Width, out.Height = width, height
			out.Terrain = make([]byte, len(s.carte.Terrain))
			for i, t := range s.carte.Terrain {
				out.Terrain[i] = byte(t)
			}
			out.Wind = s.carte.Wind
		case "step":
			// Un incendie éteint ne repart pas : inutile de brasser la carte.
			if s.engine.Burning() > 0 {
				s.engine.Step()
				s.turn++
			}
		default:
			http.Error(w, "Action inconnue", http.StatusBadRequest)
			return
		}

		out.Fire, out.Rest, out.Burning, out.Turn = s.state()
		out.Extinct = out.Burning == 0
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})
	return mux
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8081", "adresse du serveur web")
	flag.Parse()
	log.Printf("Simulation d'incendie : http://%s (Ctrl+C pour arrêter)", *addr)
	server := &http.Server{Addr: *addr, Handler: handler(), ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(server.ListenAndServe())
}
