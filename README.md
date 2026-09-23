# wildfire — Audit de performance d'une simulation d'incendie en Go

TP noté *Optimisations & Performances Backend* (Sup de Vinci). Le livrable noté est `rapport/RAPPORT.md` ;
ce dépôt sert de preuve de reproductibilité.

Les règles du modèle sont spécifiées dans **[`internal/fire/REGLES.md`](internal/fire/REGLES.md)**.
Ce document fait foi : `internal/firetest` le traduit en tests, et toute implémentation doit les passer
avant d'être mesurée.

## Prérequis

- Linux ou macOS. **Sous Windows : WSL2 (Ubuntu)**, indispensable pour `make`, `lscpu` et les scripts bash.
- Go ≥ 1.22, Hyperfine (`sudo apt install hyperfine`), benchstat (`make tools`).

## Démarrage

```bash
make demo                 # un incendie animé en console
make web                  # la même chose dans le navigateur, en couleurs
make test                 # conformité de toutes les implémentations
make quick                # une exécution de la baseline
make profile IMPL=naive   # profils CPU + allocations -> results/profiles/
make flame IMPL=naive     # flamegraph dans le navigateur (View > Flame Graph)
make bench                # campagne complète -> results/<date>-<commit>/
make help                 # toutes les cibles
```

Sous WSL2, `make web` écoute sur la boucle locale de la distribution, que le navigateur Windows
ne voit pas toujours. Dans ce cas :

```bash
make web ADDR=0.0.0.0:8081      # puis ouvrir http://<ip-wsl>:8081
hostname -I | awk '{print $1}'  # l'adresse à utiliser
```

## Protocole à deux bancs

Chaque étape d'optimisation est mesurée sur **les deux machines de l'équipe**, pour vérifier que le
gain survit au changement d'architecture :

| Banc | Rôle |
|---|---|
| MacBook Air M1 (ARM) | **référence** — tous les chiffres du rapport |
| Intel Core Ultra 9 / WSL2 (x86) | contrôle — uniquement les *ratios* de gain, jamais les valeurs absolues |

```bash
make bench                  # campagne sur cette machine
make bench BANC=m1-air      # nom de banc explicite
```

Les résultats atterrissent dans `results/<commit>/<banc>/` : le même commit mesuré sur les deux
machines donne deux dossiers frères, directement comparables. Le pipeline **refuse de mesurer sur un
arbre de travail modifié** (`FORCE=1` pour passer outre, à éviter) — un dossier de résultats doit
toujours correspondre au code du commit qu'il nomme.

## La commande `wildfire`

Les cibles `make` appellent toutes le même binaire. En direct :

```bash
go run ./cmd/wildfire -list                          # implémentations enregistrées
go run ./cmd/wildfire -w 100 -h 35 -fires 2 -render  # démo animée
go run ./cmd/wildfire -quiet -size 512 -turns 200    # forme utilisée par Hyperfine
```

| Drapeau | Défaut | Rôle |
|---|---|---|
| `-impl` | `naive` | Implémentation à exécuter, parmi celles listées par `-list` |
| `-list` | `false` | Affiche les implémentations disponibles et quitte |
| `-w` / `-h` | `80` / `30` | Largeur / hauteur de la carte |
| `-size` | `0` | Carte carrée de ce côté ; prime sur `-w`/`-h` |
| `-turns` | `500` | Nombre **maximal** de tours |
| `-seed` | `42` | Graine de la carte **et** des foyers — à garder fixe entre deux mesures comparées |
| `-fires` | `1` | Foyers de départ |
| `-wind` | `0.01` | Proportion de cases vent |
| `-lakes` / `-rivers` | `0.08` / `0.04` | Proportion de lacs et étangs / de rivières |
| `-forest` | `0.45` | Proportion de forêt hors de l'eau |
| `-scale` | `4` | Passes de lissage du relief : plus haut = massifs plus vastes |
| `-render` | `false` | Anime la carte en console — **fausse les mesures** |
| `-delay` | `80ms` | Pause entre deux images |
| `-cpuprofile` | `""` | Écrit un profil CPU pprof dans ce fichier |
| `-memprofile` | `""` | Écrit un profil d'allocations pprof dans ce fichier |
| `-quiet` | `false` | N'affiche rien — évite que le coût de `stdout` pollue le chronométrage |

Sortie :

```
impl=naive carte=512x512 tours=200 fin=tours épuisés feu=18344 durée=2.481s débit=2.11e+07 cases/s
```

`tours` est le nombre de tours **réellement** calculés : l'incendie peut s'éteindre avant la limite,
et `fin=éteint` l'indique. Le débit est calculé sur ce nombre réel, pas sur `-turns`.

À l'écran : `~` eau, `,` plaine, `#` forêt, `@` feu, `%` cendres encore chaudes, et une flèche pour
les cases vent, qui indiquent leur direction.

### Ce qui est chronométré

Le chronomètre couvre la construction du moteur **et** la boucle de tours. La carte
(`fire.Generate`) est engendrée avant, hors périmètre : identique pour toutes les implémentations
à graine égale, elle n'a pas à peser dans la comparaison. La construction, elle, est comptée car
elle diffère d'une version à l'autre (allocation des grilles, des tampons, d'un pool de workers).

Avec `-memprofile`, `runtime.MemProfileRate` passe à 4096 octets (défaut : 512 Kio) pour capter les
petites allocations ; le profil coûte alors du temps et cette exécution ne sert pas de mesure de durée.

## Structure

| Chemin | Rôle |
|---|---|
| `internal/fire/REGLES.md` | **Les règles du modèle** — la référence |
| `internal/fire` | Contrat `Engine`, registre, carte et génération, boucle `Run`, rendu console |
| `internal/naive` | Baseline — **figée après les mesures de référence** |
| `internal/firetest` | Conformité : un test par règle, plus une référence indépendante |
| `internal/engines` | Liste des implémentations compilées (une ligne par version) |
| `internal/bench` | Benchmarks communs, nommés `impl=…/scenario=…/size=…` pour benchstat |
| `cmd/web` | Carte interactive dans le navigateur (démo, hors mesures) |
| `scripts/` | `env.sh` (banc d'essai), `run_benchmarks.sh` (pipeline) |
| `results/` | Mesures brutes versionnées (pièces à conviction) |
| `rapport/` | Rapport d'audit et figures |

## Deux scénarios de mesure

Une implémentation peut gagner sur l'un en perdant sur l'autre, et c'est ce qu'on veut voir :

- **`front`** — un seul foyer sur une grande carte : presque rien ne brûle, le balayage intégral de
  la grille domine tout le reste.
- **`embrasement`** — 64 foyers et 50 tours de mise en régime : la carte est saturée, ce sont la
  localité mémoire et la bande passante qui décident.

## Ajouter une version optimisée

1. Copier `internal/naive` vers `internal/<nom>`, renommer le package et l'appel `fire.Register("<nom>", …)`.
2. Ajouter `_ "gol-wildfire/internal/<nom>"` dans `internal/engines/engines.go`.
3. `make test`, puis `make bench` : la nouvelle version apparaît automatiquement dans toutes les mesures.
