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

## La commande `gol`

Les cibles `make` appellent toutes le même binaire. En direct :

```bash
go run ./cmd/gol -list                      # implémentations enregistrées
go run ./cmd/gol -impl naive -size 512 -gens 100
go run ./cmd/gol -impl naive -quiet         # forme utilisée par Hyperfine
```

| Drapeau | Défaut | Rôle |
|---|---|---|
| `-impl` | `naive` | Implémentation à exécuter, parmi celles listées par `-list` |
| `-list` | `false` | Affiche les implémentations disponibles et quitte |
| `-size` | `1024` | Côté de la grille carrée |
| `-w` / `-h` | `0` | Largeur / hauteur ; chacun prime sur `-size` (grilles rectangulaires) |
| `-gens` | `200` | Nombre **maximal** de générations |
| `-seed` | `42` | Graine de la grille initiale — à garder fixe entre deux mesures comparées |
| `-density` | `0.3` | Proportion initiale de cellules vivantes |
| `-cycles` | `true` | Détection de cycles et arrêt précoce |
| `-snapshot-dir` | `""` | Répertoire des snapshots JSON ; vide = désactivé (axe I/O du TP) |
| `-snapshot-every` | `50` | Période des snapshots, en générations |
| `-cpuprofile` | `""` | Écrit un profil CPU pprof dans ce fichier |
| `-memprofile` | `""` | Écrit un profil d'allocations pprof dans ce fichier |
| `-quiet` | `false` | N'affiche rien — évite que le coût de `stdout` pollue le chronométrage |

Sortie :

```
impl=naive grille=1024x1024 générations=200 population=78251 durée=10.186s débit=2.06e+07 cellules/s
```

`générations` est le nombre de générations **réellement** calculées : avec `-cycles`, la simulation
s'arrête dès qu'un état déjà vu réapparaît, et la ligne se termine alors par
`cycle(période=…, depuis gén. …)`. Le débit est calculé sur ce nombre réel, pas sur `-gens`.

### Ce qui est chronométré

Le chronomètre couvre la construction du moteur **et** la boucle de générations. La grille initiale
(`life.RandomCells`) est générée avant, hors périmètre : identique pour toutes les implémentations
à graine égale, elle n'a pas à peser dans la comparaison. La construction, elle, est comptée car
elle diffère d'une version à l'autre (allocation de la grille, des buffers, d'un pool de workers).

Avec `-memprofile`, `runtime.MemProfileRate` passe à 4096 octets (défaut : 512 Kio) pour capter les
petites allocations ; le profil coûte alors du temps et cette exécution ne sert pas de mesure de durée.

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
