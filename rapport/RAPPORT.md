# Rapport d'audit de performance — Jeu de la vie

> **Équipe :** … — **Session :** E42 — **Dépôt :** … (commit final : `…`)
>
> Consigne de rédaction : chaque affirmation chiffrée renvoie à un fichier de `results/`.
> Rédiger chaque section **juste après** la mesure correspondante, jamais à la fin.

## Résumé exécutif

Trois phrases : point de départ (débit baseline en cellules/s), point d'arrivée, facteur de gain global et les deux leviers principaux.

---

## 1. Environnement & métrologie (baseline) — /3

### 1.1 Banc d'essai matériel

Source : `results/<run>/env.md` (généré par `make env`).

| Élément | Valeur |
|---|---|
| CPU (modèle) | |
| Cœurs physiques / threads | |
| L1d / L1i (par cœur) | |
| L2 | |
| L3 | |
| Ligne de cache | 64 o |
| RAM | |
| OS / noyau | (préciser si WSL2) |
| Runtime | go 1.xx.x, GOAMD64=… |
| Alimentation / gouverneur | secteur, `performance` |

### 1.2 Protocole de mesure

- Outil : Hyperfine `-N --warmup 3 --runs 15`, binaire exécuté sans shell intermédiaire.
- Justification du warmup : cache disque du binaire, caches CPU, stabilisation de la fréquence.
- Charge de travail : grille 1024×1024, graine 42, densité 0,3, 50 générations. **Identique pour toutes les versions.**
- Isolation du bruit : navigateur/IDE fermés, machine sur secteur, charge système vérifiée avant chaque campagne (voir `env.md`), [pinning `taskset` si utilisé].
- Micro-benchmarks : `go test -bench -benchmem -count 10`, comparés avec benchstat (intervalle de confiance, test de significativité).

### 1.3 Mesures de référence

Coller `results/<run>/hyperfine-stats.md` (moyenne, médiane, écart-type, variance, CV).
Commenter le coefficient de variation : < 2 % = mesure stable.

---

## 2. Diagnostic matériel & profiling réel — /5

### 2.1 Profil CPU de la baseline

![Flamegraph CPU baseline](figures/flame-cpu-naive.png)

Annoter la capture : encadrer `Fingerprint` → `fmt.Sprintf`, et `Step` → `neighbors`.
Source : `results/profiles/naive-cpu-top.txt`, `naive-cpu-list.txt` (coût ligne par ligne).

### 2.2 Profil d'allocations

![Flamegraph allocations baseline](figures/flame-alloc-naive.png)

Chiffres attendus : allocations par génération (`allocs/op` de `BenchmarkStep` et `BenchmarkFingerprint`), octets alloués, part du temps passée dans `runtime.mallocgc` et dans le GC (`GODEBUG=gctrace=1`).

### 2.3 Identification formelle du hot path

Pour chaque goulot, l'enchaînement **symptôme mesuré → cause mécanique → preuve** :

1. **`Fingerprint`, xx % du CPU** — un `fmt.Sprintf` par cellule vivante (~300 000 par génération en 1024²) : réflexion, conversion d'entiers, allocation d'une chaîne à chaque appel, puis copie `string → []byte` → pression GC.
2. **`neighbors`, xx % du CPU** — 8 modulos (division entière : ~20-40 cycles chacune) et 8 branchements par cellule ; double indirection `[][]bool` : chaque ligne est un bloc distinct du tas, la ligne y-1, y et y+1 ne sont pas contiguës.
3. **`Step`, une grille complète allouée par génération** — 1024 slices + 1 Mo par génération, collectés par le GC.

---

## 3. Journal d'optimisation — /5

Une entrée par étape, toujours au même format :

> **Étape N — titre** (commit `…`)
> - **Hypothèse d'impact matériel :** …
> - **Modification :** …
> - **Commande de vérification :** `…`
> - **Résultat :** avant → après (benchstat, avec p-value) ; allocs/op avant → après.
> - **Explication physique :** …

### 3.1 Mémoire & localité de cache

- Grille plate `[]uint8` + double buffering (zéro allocation par génération).
- Suppression des modulos : lignes fantômes ou traitement séparé des bords.
- Bit-packing : 64 cellules par `uint64`, comptage des voisins par opérations bit à bit ; empreinte mémoire ÷ 8, une ligne de 1024 cellules = 128 o = 2 lignes de cache.
- Struct padding : `unsafe.Sizeof(Sim{})` 72 o → 56 o (`make layout`).
- Empreinte sans allocation : hachage FNV-1a direct sur les mots de la grille (`make escape` pour prouver l'absence d'échappement).

### 3.2 Concurrence & scalabilité CPU

- Worker pool : découpage en bandes horizontales, nombre de workers = cœurs **physiques** (justifier contre les threads SMT).
- Synchronisation : `sync.WaitGroup` par génération ou barrière ; population comptée avec `atomic.Int64`.
- Arrêt précoce : `context.WithCancel` / `WithTimeout`, annulation dès détection d'un cycle.
- Courbe de scalabilité : temps en fonction du nombre de workers (1, 2, 4, …), comparée à la loi d'Amdahl.

### 3.3 I/O réseau & persistance

- Snapshots : JSON `[][]bool` → Protobuf / format bit-packé (taille fichier et temps de sérialisation).
- PostgreSQL : historique (génération, empreinte, population) ; requête de détection de cycle sans puis avec index (`EXPLAIN ANALYZE` avant/après).
- Cache : LRU des empreintes déjà vues ; `sync.Pool` pour les buffers de sérialisation.

---

## 4. Confrontation critique & échec constructif — /3

> **Tentative :** … (ex. une goroutine par cellule, ou bandes trop fines → *false sharing*)
> - **Hypothèse initiale :** …
> - **Mesure :** régression de x % (benchstat, commit `…`)
> - **Explication mécanique chiffrée :** coût d'ordonnancement d'une goroutine (~µs) × 1 M cellules vs coût du calcul d'une cellule (~ns) ; ou invalidations de lignes de cache entre cœurs (compteurs `perf stat -e cache-misses` si disponible).
> - **Retour arrière :** commit `…` (git revert)

---

## 5. Reproductibilité & synthèse comparative — /4

### 5.1 Reproduire toutes les mesures

```bash
git clone … && cd gol
make tools   # benchstat
make bench   # env + tests + go bench + hyperfine + benchstat -> results/<date>-<commit>/
```

### 5.2 Tableau de synthèse

Coller `results/<run>/benchstat.txt` (benchstat `-col /impl`) et le tableau Hyperfine.

| Version | Temps (1024², 50 gén.) | Débit (cellules/s) | allocs/génération | Gain cumulé |
|---|---|---|---|---|
| naive (baseline) | | | | ×1 |
| flat + double buffer | | | 0 | |
| bitpack | | | 0 | |
| parallel (N workers) | | | | |

Conclure en ordres de grandeur, et sur la limite atteinte (calcul ou bande passante mémoire ?).

---

## 6. Bonus — gouvernance IA

Voir `constitution.md` à la racine du dépôt. Expliquer en quelques lignes comment il a été utilisé pendant le TP.
