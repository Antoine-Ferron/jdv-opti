// Commande gol : exécute une simulation, mesurable par Hyperfine et profilable par pprof.
//
//	gol -impl naive -size 1024 -gens 200
//	gol -impl naive -cpuprofile results/cpu.prof -memprofile results/mem.prof
//	gol -list
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	_ "gol/internal/engines"
	"gol/internal/life"
)

func main() {
	var (
		impl       = flag.String("impl", "naive", "implémentation à exécuter (voir -list)")
		list       = flag.Bool("list", false, "affiche les implémentations disponibles et quitte")
		size       = flag.Int("size", 1024, "côté de la grille carrée (ignoré si -w/-h sont fournis)")
		width      = flag.Int("w", 0, "largeur de la grille")
		height     = flag.Int("h", 0, "hauteur de la grille")
		gens       = flag.Int("gens", 200, "nombre maximal de générations")
		seed       = flag.Int64("seed", 42, "graine de la grille initiale (fixe = reproductible)")
		density    = flag.Float64("density", 0.3, "proportion initiale de cellules vivantes")
		cycles     = flag.Bool("cycles", true, "détection de cycles et arrêt précoce")
		snapDir    = flag.String("snapshot-dir", "", "répertoire des snapshots JSON (vide = désactivé)")
		snapEvery  = flag.Int("snapshot-every", 50, "période des snapshots en générations")
		cpuProfile = flag.String("cpuprofile", "", "écrit un profil CPU pprof dans ce fichier")
		memProfile = flag.String("memprofile", "", "écrit un profil d'allocations pprof dans ce fichier")
		quiet      = flag.Bool("quiet", false, "n'affiche rien (pour Hyperfine)")
	)
	flag.Parse()

	if *list {
		fmt.Println(strings.Join(life.Names(), "\n"))
		return
	}
	f, ok := life.Get(*impl)
	if !ok {
		fail("implémentation inconnue %q (disponibles : %s)", *impl, strings.Join(life.Names(), ", "))
	}
	w, h := *size, *size
	if *width > 0 {
		w = *width
	}
	if *height > 0 {
		h = *height
	}

	// La génération de la grille initiale est hors du périmètre profilé.
	cells := life.RandomCells(w, h, *seed, *density)

	if *memProfile != "" {
		runtime.MemProfileRate = 4096 // échantillonnage plus fin que le défaut (512 Kio)
	}
	if *cpuProfile != "" {
		pf, err := os.Create(*cpuProfile)
		if err != nil {
			fail("%v", err)
		}
		if err := pprof.StartCPUProfile(pf); err != nil {
			fail("%v", err)
		}
		defer pf.Close()
		defer pprof.StopCPUProfile()
	}

	start := time.Now()
	e := f(w, h, cells)
	res, err := life.Run(e, life.Options{
		Generations:   *gens,
		DetectCycles:  *cycles,
		SnapshotDir:   *snapDir,
		SnapshotEvery: *snapEvery,
	})
	if err != nil {
		fail("%v", err)
	}
	elapsed := time.Since(start)

	if *memProfile != "" {
		mf, err := os.Create(*memProfile)
		if err != nil {
			fail("%v", err)
		}
		if err := pprof.Lookup("allocs").WriteTo(mf, 0); err != nil {
			fail("%v", err)
		}
		mf.Close()
	}

	if !*quiet {
		cellsPerSec := float64(w*h) * float64(res.Generations) / elapsed.Seconds()
		fmt.Printf("impl=%s grille=%dx%d générations=%d population=%d durée=%s débit=%.3g cellules/s",
			*impl, w, h, res.Generations, res.Population, elapsed.Round(time.Millisecond), cellsPerSec)
		if res.Period > 0 {
			fmt.Printf(" cycle(période=%d, depuis gén. %d)", res.Period, res.CycleFrom)
		}
		fmt.Println()
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gol: "+format+"\n", args...)
	os.Exit(1)
}
