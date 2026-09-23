// Package engines importe toutes les implémentations pour qu'elles
// s'enregistrent. Ajouter une nouvelle version = créer son package, puis
// ajouter une ligne ici.
package engines

import (
	_ "gol-wildfire/internal/flat"  // étape 1 : grille plate, double tampon, tampon d'ignition réutilisé
	_ "gol-wildfire/internal/naive" // baseline — ne plus modifier après les mesures de référence
	// _ ".../counters"  // étape 2 : Burning incrémental (6,74 % sur M1)
	// _ ".../ghost"     // étape 3 : bordure fantôme, plus aucun modulo (41 % sur x86, 5 % sur M1)
	// _ ".../bitpack"   // étape 4 : plans de bits, propagation par OU
	// _ ".../front"     // étape 5 : liste des cases actives — gain selon le régime (§4)
	// _ ".../parallel"  // étape 6 : worker pool, cœurs performance
)
