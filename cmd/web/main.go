// Command web affiche le moteur Go dans un navigateur.
package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"time"

	"gol/internal/life"
	"gol/internal/naive"
)

//go:embed index.html
var page []byte

const size = 50

func handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	})
	mux.HandleFunc("POST /api/simulation", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Action  string `json:"action"`
			Pattern string `json:"pattern"`
			Seed    int64  `json:"seed"`
			Cells   []bool `json:"cells"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 32768)
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Requête invalide", http.StatusBadRequest)
			return
		}
		cells := input.Cells
		switch input.Action {
		case "reset":
			cells = make([]bool, size*size)
			var points [][2]int
			switch input.Pattern {
			case "random":
				cells = life.RandomCells(size, size, input.Seed, .3)
			case "glider":
				points = [][2]int{{24, 23}, {25, 24}, {23, 25}, {24, 25}, {25, 25}}
			case "blinker":
				points = [][2]int{{24, 25}, {25, 25}, {26, 25}}
			case "block":
				points = [][2]int{{24, 24}, {25, 24}, {24, 25}, {25, 25}}
			default:
				http.Error(w, "Motif inconnu", http.StatusBadRequest)
				return
			}
			for _, p := range points {
				cells[p[1]*size+p[0]] = true
			}
		case "step":
			if len(cells) != size*size {
				http.Error(w, "La grille doit contenir 2500 cellules", http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, "Action inconnue", http.StatusBadRequest)
			return
		}
		e := naive.New(size, size, cells)
		if input.Action == "step" {
			e.Step()
		}
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				cells[y*size+x] = e.Alive(x, y)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Cells      []bool `json:"cells"`
			Population int    `json:"population"`
		}{cells, e.Population()})
	})
	return mux
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8081", "adresse du serveur web")
	flag.Parse()
	log.Printf("Jeu de la vie : http://%s (Ctrl+C pour arrêter)", *addr)
	server := &http.Server{Addr: *addr, Handler: handler(), ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(server.ListenAndServe())
}
