// Package engines importe toutes les implémentations pour qu'elles
// s'enregistrent. Ajouter une nouvelle version = créer son package, puis
// ajouter une ligne ici.
package engines

import (
	_ "gol-wildfire/internal/naive" // baseline — ne plus modifier après les mesures de référence
	// _ "gol-wildfire/internal/flat"     // étape 1 : grille plate + double buffering
	// _ "gol-wildfire/internal/front"    // étape 2 : liste des cases actives
	// _ "gol-wildfire/internal/parallel" // étape 3 : worker pool
)
