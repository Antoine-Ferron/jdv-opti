// Package engines importe toutes les implémentations pour qu'elles s'enregistrent.
// Ajouter une nouvelle version = créer son package, puis ajouter une ligne ici.
package engines

import (
	_ "gol/internal/naive" // baseline — ne plus modifier après les mesures de référence
	// _ "gol/internal/flat"    // étape 1 : grille plate + double buffering
	// _ "gol/internal/bitpack" // étape 2 : 64 cellules par uint64
	// _ "gol/internal/parallel" // étape 3 : worker pool
)
