// Package engines importe toutes les implémentations pour qu'elles
// s'enregistrent. Ajouter une version = créer son package, puis ajouter une
// ligne ici.
//
// Les parts de CPU ci-dessous viennent des profils du commit 1d3a093 sur les
// deux bancs (results/1d3a093/*/profiles/). Elles sont données pour le **banc A**
// (M1), celui qui fait foi, et dans les deux régimes, parce qu'ils ne désignent
// pas les mêmes cibles :
//
//	saturé — 64 foyers, la carte brûle partout : Step 76-79 %, Mod 7-9 %, Burning 3-6 %
//	front  — un seul foyer, presque rien ne brûle : Step 73-78 %, Burning 21 %, Mod absent
//
// Deux conséquences pour la suite. D'abord, après l'étape 2, tout gain notable
// passera par le corps de Step lui-même : rien d'autre ne dépasse 21 % sur ce
// banc. Ensuite, un gain mesuré par BenchmarkRun doit être confronté au binaire
// complet avant d'être annoncé — voir le §2.4 du rapport, où BenchmarkRun
// annonçait ×3,4 pour flat là où le programme réel donne ×1,12.
package engines

import (
	_ "gol-wildfire/internal/counters" // étape 2 : Burning incrémental — 21 % en front, 3-6 % en saturé.
	_ "gol-wildfire/internal/flat"     // étape 1 : grille plate, double tampon, tampon d'ignition réutilisé
	_ "gol-wildfire/internal/naive"    // baseline — ne plus modifier après les mesures de référence
	//                   //           La modification la moins coûteuse du lot, et le plus gros
	//                   //           gisement encore ouvert sur le banc A.
	// _ ".../ghost"     // étape 3 : bordure fantôme, plus aucun modulo — 7-9 % en saturé sur le
	//                   //           banc A, 37-41 % sur le banc B, et RIEN en front (sans
	//                   //           propagation, pas de modulo). Gardée à cette place pour deux
	//                   //           raisons : elle prépare le bit-packing, qu'un enroulement
	//                   //           torique calculé par modulo rendrait pénible ; et l'écart entre
	//                   //           les deux bancs est la meilleure démonstration de
	//                   //           non-portabilité du rapport. Ne pas en attendre un gain majeur.
	// _ ".../bitpack"   // étape 4 : plans de bits, propagation par OU — attaque Step, seul poste
	//                   //           qui dépasse 21 % sur le banc A. C'est là qu'est le gros du gain.
	// _ ".../front"     // étape 5 : liste des cases actives — attaque Step ET Burning quand peu de
	//                   //           cases brûlent, et ne rapporte rien en régime saturé. Candidat
	//                   //           du §4 : le gain dépend du régime, pas de l'implémentation.
	// _ ".../parallel"  // étape 6 : worker pool dimensionné aux cœurs performance (4 sur M1), pas
	//                   //           à GOMAXPROCS — voir §1.1, réserve 1.
)
