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

	// Import muet : le seul rôle de ce paquet est de déclencher les init()
	// qui enregistrent les implémentations dans le registre de life.
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

	// Résolution du nom vers la fabrique enregistrée : échoue avant tout
	// travail coûteux si le nom est faux.
	f, ok := life.Get(*impl)
	if !ok {
		fail("implémentation inconnue %q (disponibles : %s)", *impl, strings.Join(life.Names(), ", "))
	}

	// -w/-h priment individuellement sur -size, ce qui permet des grilles
	// rectangulaires sans avoir à fournir les deux dimensions.
	w, h := *size, *size
	if *width > 0 {
		w = *width
	}
	if *height > 0 {
		h = *height
	}

	// La génération de la grille initiale est hors du périmètre profilé.
	cells := life.RandomCells(w, h, *seed, *density)

	// Réglé avant toute allocation à mesurer : le taux d'échantillonnage n'est
	// lu qu'au moment des allocations, le changer après ne rétroagit pas.
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
		// LIFO : StopCPUProfile vide son tampon dans pf avant que pf ne soit
		// fermé. Inverser ces deux lignes tronquerait le profil.
		// Attention : fail() appelle os.Exit, qui n'exécute aucun defer — une
		// erreur après ce point perd le profil CPU.
		defer pf.Close()
		defer pprof.StopCPUProfile()
	}

	// Périmètre chronométré : construction du moteur + boucle de générations.
	// La construction en fait partie car elle diffère d'une implémentation à
	// l'autre (allocation de la grille, des buffers, du pool de workers…).
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

	// Profil mémoire écrit après coup : "allocs" est cumulatif depuis le
	// démarrage, il n'a donc pas besoin d'être démarré/arrêté comme le CPU.
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
		// res.Generations et non *gens : la détection de cycles peut avoir
		// arrêté la simulation plus tôt, le débit doit refléter le travail réel.
		cellsPerSec := float64(w*h) * float64(res.Generations) / elapsed.Seconds()
		fmt.Printf("impl=%s grille=%dx%d générations=%d population=%d durée=%s débit=%.3g cellules/s",
			*impl, w, h, res.Generations, res.Population, elapsed.Round(time.Millisecond), cellsPerSec)
		if res.Period > 0 {
			fmt.Printf(" cycle(période=%d, depuis gén. %d)", res.Period, res.CycleFrom)
		}
		fmt.Println()
	}
}

// fail écrit un message sur stderr et quitte en code 1 : les defer en cours
// ne sont pas exécutés (voir la note sur le profil CPU ci-dessus).
func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gol: "+format+"\n", args...)
	os.Exit(1)
}
