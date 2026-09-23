// Commande wildfire : déroule un incendie, en démo animée ou en exécution
// mesurable par Hyperfine et profilable par pprof.
//
//	wildfire -w 100 -h 40 -fires 2 -render
//	wildfire -quiet -size 512 -turns 200
//	wildfire -impl naive -cpuprofile results/cpu.prof -memprofile results/mem.prof
//	wildfire -list
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
	// qui enregistrent les implémentations dans le registre de fire.
	_ "gol-wildfire/internal/engines"
	"gol-wildfire/internal/fire"
)

func main() {
	var (
		impl       = flag.String("impl", "naive", "implémentation à exécuter (voir -list)")
		list       = flag.Bool("list", false, "affiche les implémentations disponibles et quitte")
		size       = flag.Int("size", 0, "côté d'une carte carrée (prime sur -w/-h)")
		width      = flag.Int("w", 80, "largeur de la carte")
		height     = flag.Int("h", 30, "hauteur de la carte")
		turns      = flag.Int("turns", 500, "nombre maximal de tours")
		seed       = flag.Int64("seed", 42, "graine : même graine, même carte et mêmes foyers")
		fires      = flag.Int("fires", 1, "foyers de départ")
		wind       = flag.Float64("wind", 0.01, "proportion de cases vent")
		lakes      = flag.Float64("lakes", 0.08, "proportion de lacs et étangs")
		rivers     = flag.Float64("rivers", 0.04, "proportion de rivières")
		forest     = flag.Float64("forest", 0.45, "proportion de forêt hors de l'eau")
		scale      = flag.Int("scale", 4, "passes de lissage : plus haut = massifs plus vastes")
		render     = flag.Bool("render", false, "anime la carte en console (démo : fausse les mesures)")
		delay      = flag.Duration("delay", 80*time.Millisecond, "pause entre deux images")
		cpuProfile = flag.String("cpuprofile", "", "écrit un profil CPU pprof dans ce fichier")
		memProfile = flag.String("memprofile", "", "écrit un profil d'allocations pprof dans ce fichier")
		quiet      = flag.Bool("quiet", false, "n'affiche rien (pour Hyperfine)")
	)
	flag.Parse()

	if *list {
		fmt.Println(strings.Join(fire.Names(), "\n"))
		return
	}

	// Résolution du nom avant tout travail coûteux : échouer tout de suite si
	// l'implémentation demandée n'existe pas.
	f, ok := fire.Get(*impl)
	if !ok {
		fail("implémentation inconnue %q (disponibles : %s)", *impl, strings.Join(fire.Names(), ", "))
	}

	w, h := *width, *height
	if *size > 0 {
		w, h = *size, *size
	}
	cfg := fire.Config{
		Width: w, Height: h, Seed: *seed, Scale: *scale,
		Lakes: *lakes, Rivers: *rivers, Forest: *forest, Wind: *wind, Fires: *fires,
	}

	// La carte est engendrée hors du périmètre chronométré : identique pour
	// toutes les implémentations à graine égale, elle n'a pas à peser dans la
	// comparaison.
	m := fire.Generate(cfg)

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
		// fermé. Attention : fail() appelle os.Exit, qui n'exécute aucun defer.
		defer pf.Close()
		defer pprof.StopCPUProfile()
	}

	opt := fire.Options{Turns: *turns}
	if *render {
		opt.Render, opt.RenderDelay = os.Stdout, *delay
		fmt.Print("\033[2J") // efface l'écran une fois, les images suivantes se superposent
	}

	// Périmètre chronométré : construction du moteur + boucle de tours. La
	// construction en fait partie car elle diffère d'une implémentation à
	// l'autre (allocation des grilles, des tampons, du pool de workers…).
	start := time.Now()
	e := f(m)
	res := fire.Run(e, opt)
	elapsed := time.Since(start)
	if res.Err != nil {
		fail("%v", res.Err)
	}

	// Profil mémoire écrit après coup : "allocs" est cumulatif depuis le
	// démarrage, il n'a pas besoin d'être démarré/arrêté comme le CPU.
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
		// res.Turns et non *turns : l'incendie a pu s'éteindre plus tôt, le
		// débit doit refléter le travail réellement fait.
		parSec := float64(w*h) * float64(res.Turns) / elapsed.Seconds()
		fin := "tours épuisés"
		if res.Extinct {
			fin = "éteint"
		}
		fmt.Printf("impl=%s carte=%dx%d tours=%d fin=%s feu=%d durée=%s débit=%.3g cases/s\n",
			*impl, w, h, res.Turns, fin, e.Burning(), elapsed.Round(time.Millisecond), parSec)
	}
}

// fail écrit un message sur stderr et quitte en code 1 : les defer en cours ne
// sont pas exécutés (voir la note sur le profil CPU ci-dessus).
func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "wildfire: "+format+"\n", args...)
	os.Exit(1)
}
