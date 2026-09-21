# gol — Audit de performance d'un jeu de la vie en Go

TP noté *Optimisations & Performances Backend* (Sup de Vinci). Le livrable noté est `rapport/RAPPORT.md` ;
ce dépôt sert de preuve de reproductibilité.

## Prérequis

- Linux ou macOS. **Sous Windows : WSL2 (Ubuntu)**, indispensable pour `make`, `lscpu` et les scripts bash.
- Go ≥ 1.22, Hyperfine (`sudo apt install hyperfine`), benchstat (`make tools`).

## Démarrage

```bash
make test                 # conformité de toutes les implémentations
make quick                # une exécution de la baseline
make profile IMPL=naive   # profils CPU + allocations -> results/profiles/
make flame IMPL=naive     # flamegraph dans le navigateur (View > Flame Graph)
make bench                # campagne complète -> results/<date>-<commit>/
make help                 # toutes les cibles
```

## Structure

| Chemin | Rôle |
|---|---|
| `internal/life` | Interface `Engine`, registre, boucle `Run`, snapshots JSON |
| `internal/naive` | Baseline — **figée après les mesures de référence** |
| `internal/lifetest` | Suite de conformité contre une référence indépendante |
| `internal/engines` | Liste des implémentations compilées (une ligne par version) |
| `internal/bench` | Benchmarks communs, nommés `impl=…/size=…` pour benchstat |
| `scripts/` | `env.sh` (banc d'essai), `run_benchmarks.sh` (pipeline) |
| `results/` | Mesures brutes versionnées (pièces à conviction) |
| `rapport/` | Rapport d'audit et figures |

## Ajouter une version optimisée

1. Copier `internal/naive` vers `internal/<nom>`, renommer le package et l'appel `life.Register("<nom>", …)`.
2. Ajouter `_ "gol/internal/<nom>"` dans `internal/engines/engines.go`.
3. `make test`, puis `make bench` : la nouvelle version apparaît automatiquement dans toutes les mesures.
