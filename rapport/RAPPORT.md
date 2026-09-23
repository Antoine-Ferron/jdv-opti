# Rapport d'audit de performance — Simulation d'incendie

> **Équipe :** … — **Session :** E42 — **Dépôt :** … (commit final : `…`)
>
> Consigne de rédaction : chaque affirmation chiffrée renvoie à un fichier de `results/`.
> Rédiger chaque section **juste après** la mesure correspondante, jamais à la fin.
>
> Les règles du modèle simulé sont spécifiées dans `internal/fire/REGLES.md` ; les renvois `§n`
> ci-dessous pointent vers ses sections. Les défauts volontaires de la baseline sont numérotés
> `[F1]`–`[F7]` en tête de `internal/naive/naive.go`.

## Résumé exécutif

Trois phrases : point de départ (débit baseline en cases/s), point d'arrivée, facteur de gain global
et les deux leviers principaux.

---

## 1. Environnement & métrologie (baseline) — /3

### 1.1 Banc d'essai matériel

Source : `results/<commit>/<banc>/env.md`, généré par `make env` au début de chaque campagne.

**Deux bancs, deux rôles distincts.** Chaque étape d'optimisation est mesurée sur les deux
machines, pour répondre à une question que le barème ne pose pas mais qu'un ingénieur se pose :
*le gain tient-il quand l'architecture change ?*

| Banc | Rôle | Ce qu'on en tire |
|---|---|---|
| **A — MacBook Air M1** (ARM) | **référence** | Tous les chiffres cités dans ce rapport, sauf mention contraire explicite. |
| **B — Intel Core Ultra 9 / WSL2** (x86) | contrôle | Uniquement le *ratio* de gain de chaque étape, comparé à celui du banc A. |

Règle appliquée sans exception : **aucun tableau ne mélange les deux bancs**, et aucune valeur
absolue du banc B n'est citée comme résultat. Comparer 2,1 s sur M1 à 0,8 s sur x86 n'apprend rien ;
comparer un gain de ×4,1 à un gain de ×3,8 apprend que l'optimisation est portable.

#### Banc A — référence

> Valeurs issues d'un relevé `make env` antérieur, **à confirmer** par la première campagne
> officielle.

| Élément                   | Valeur                                                        |
|---------------------------|---------------------------------------------------------------|
| CPU (modèle)              | Apple M1                                                      |
| Cœurs physiques / threads | 8 / 8 — **4 performance + 4 efficiency**, pas de SMT           |
| L1i / L1d (par cœur)      | 128 Kio / 64 Kio                                              |
| L2                        | 4 Mio                                                         |
| L3                        | pas de L3 classique — *à préciser (System Level Cache)*       |
| Ligne de cache            | **128 o**                                                     |
| RAM                       | 8 Gio unifiée                                                 |
| OS                        | macOS 15.5 (build 24F74)                                      |
| Runtime                   | *à relever* : `go version`, `GOARCH=arm64`, `CGO_ENABLED`, `GOGC`, `GOMAXPROCS` |
| SIMD disponibles          | NEON 128 bits (pas d'AVX : architecture ARM)                  |
| Alimentation              | *à préciser* : secteur ou batterie, et réglage d'économie d'énergie |

**Quatre réserves à porter au crédit de la métrologie, pas à sa charge :**

1. **Cœurs hétérogènes.** Le M1 mêle 4 cœurs *performance* et 4 cœurs *efficiency*. Conséquence
   directe pour le §3.2 : un worker pool dimensionné à `GOMAXPROCS` (8) répartit le travail sur des
   cœurs de puissances très inégales, et la bande la plus lente impose son rythme à la barrière de
   synchronisation. Attendre une scalabilité sous-linéaire dès que le nombre de workers dépasse 4,
   et dimensionner le pool aux cœurs *performance*.
2. **Ligne de cache de 128 octets**, le double du x86. Le *false sharing* du §4 se joue donc sur des
   zones deux fois plus larges, et le découpage en bandes du worker pool doit aligner ses frontières
   sur 128 o — une bande dont le bord partage une ligne avec la bande voisine invalide le cache des
   deux cœurs à chaque écriture.
3. **`perf` n'existe pas sur macOS.** Pas de compteur matériel `cache-misses` en ligne de commande.
   Les preuves du §2 reposent donc sur pprof (CPU et allocations) et, si un compteur matériel
   devient nécessaire, sur Instruments. À dire explicitement plutôt qu'à passer sous silence.
4. **Fréquence non verrouillée.** Apple Silicon ne laisse ni piloter le gouverneur ni figer le
   turbo. C'est le warmup Hyperfine et le coefficient de variation qui attestent de la stabilité,
   pas un réglage système.

#### Banc B — contrôle de portabilité

| Élément                   | Valeur                                                                                             |
|---------------------------|----------------------------------------------------------------------------------------------------|
| CPU (modèle)              | Intel Core Ultra 9 275HX (Arrow Lake-HX), base 2,7 GHz                                             |
| Cœurs physiques / threads | 24 / 24 — **pas de SMT** : 1 thread par cœur                                                       |
| L1d / L1i (par cœur)      | 48 Kio / 64 Kio                                                                                    |
| L2                        | 3 Mio privés par cœur — **40 Mio au total** côté hôte                                              |
| L3                        | 36 Mio partagés par les 24 cœurs                                                                   |
| Ligne de cache            | 64 o                                                                                               |
| RAM                       | 64 Gio DDR5-6400 — **31 Gio visibles depuis WSL2**                                                 |
| OS / noyau                | Windows 11 Famille 10.0.26200 → **WSL2** Ubuntu 26.04 LTS, noyau 6.6.114.1-microsoft-standard-WSL2 |
| Runtime                   | go1.27.1 linux/amd64, `GOAMD64=v1`, `CGO_ENABLED=0`, `GOGC=100`, `GOMAXPROCS=24`                   |
| SIMD disponibles          | sse4_2, avx, avx2 (pas d'AVX-512 sur Arrow Lake)                                                   |

Source : `results/env-2026-09-22.md`. Trois réserves, qui expliquent pourquoi ce banc ne fournit que
des ratios :

1. **Virtualisation Hyper-V.** Les mesures tournent dans WSL2, pas sur le métal. Le coût est
   constant entre les versions comparées — les ratios restent valides, les valeurs absolues sont
   minorées.
2. **Topologie hybride masquée.** L'Arrow Lake-HX mêle P-cores et E-cores, mais WSL2 présente 24
   cœurs homogènes (`lscpu` annonce même 72 Mio de L2, contre 40 Mio relevés côté hôte). Même
   conséquence qu'au banc A pour le §3.2, en pire : on ne sait même pas où sont les P-cores.
3. **Fréquence non verrouillée**, ni gouverneur ni turbo pilotables depuis WSL2.

### 1.2 Protocole de mesure

- Outil : Hyperfine `-N --warmup 3 --runs 15`, binaire exécuté sans shell intermédiaire.
- Justification du warmup : cache disque du binaire, caches CPU, stabilisation de la fréquence.
- Charge de travail : carte 1024×1024, graine 42, 64 foyers, `TURNS` tours. **Identique pour toutes
  les versions et pour les deux bancs** — elle est dimensionnée pour la plus petite des deux
  machines (8 Gio sur le banc A), ce qui exclut de promouvoir en charge officielle une carte au-delà
  de 4096².
- Une campagne = `make bench BANC=<nom>`, qui écrit dans `results/<commit>/<banc>/`. Le même commit
  mesuré sur les deux machines donne deux dossiers frères. Le pipeline **refuse de mesurer sur un
  arbre de travail modifié** : un dossier de résultats doit toujours correspondre exactement au code
  du commit qu'il nomme.
- Isolation du bruit : navigateur/IDE fermés, machine sur secteur, charge système vérifiée avant
  chaque campagne (voir `env.md`), [pinning si utilisé].
- Micro-benchmarks : `go test -bench -benchmem -count 10`, comparés avec benchstat (intervalle de
  confiance, test de significativité).

**Un Step isolé ne se mesure qu'en régime établi.** `go test` choisit lui-même son nombre
d'itérations ; or en scénario `front` l'incendie s'étend pendant la mesure, donc le coût moyen d'un
tour dépend de ce nombre. Première campagne à l'appui : `Step/front/size=2048` donnait **±33 %** de
variance, inexploitable pour comparer deux implémentations, quand `Run` — qui borne le travail par
itération — tenait ±2 %. `BenchmarkStep` ne mesure donc que le régime saturé, et le scénario `front`
est mesuré par `BenchmarkRun`.

**Deux périmètres de mesure, à ne pas confondre — et c'est ce qui a fixé la charge.** La carte est
engendrée par `fire.Generate` **avant** la boucle de tours : les micro-benchmarks l'excluent de leur
chronomètre, mais Hyperfine mesure le processus entier, génération comprise. Sur une charge trop
courte, la génération domine le temps total et **écrase le gain à mesurer**.

Mesuré sur le banc B, carte 1024², 64 foyers :

| Charge | Temps total | Dont simulation | Part de la génération |
|---|---|---|---|
| 50 tours | 0,84 s | 0,178 s | **79 %** |
| **500 tours** | 8,15 s | 7,49 s | **8 %** |

À 50 tours, une optimisation qui diviserait `Step` par deux n'aurait apparu que comme ~10 % sur la
ligne Hyperfine. **La charge est donc fixée à 500 tours**, ce qui ramène la génération sous 10 % du
temps mesuré.

Second argument, indépendant : le débit passe de 2,9 × 10⁸ à 7,0 × 10⁷ cases/s entre 50 et 500
tours. À 50 tours l'incendie n'a pas fini de s'étendre — on mesure un transitoire ; à 500 tours il
a atteint le régime entretenu décrit dans `REGLES.md` §4. Seule la seconde mesure est représentative
de ce que fait le programme.

### 1.3 Mesures de référence

#### Banc A — référence

*À produire* : coller `results/<commit>/<banc-A>/hyperfine-stats.md` (moyenne, médiane, écart-type,
variance, CV) et commenter le coefficient de variation.

#### Banc B — contrôle

Campagne de référence, commit `a47848f`, carte 1024², 64 foyers, 500 tours, Hyperfine `-N --warmup 3
--runs 15` (`results/a47848f/x86-controle/`) :

| Moyenne | Médiane | Écart-type | Variance | CV | Min | Max |
|---|---|---|---|---|---|---|
| 8,2574 s | 8,2408 s | 0,0893 s | 7,976 × 10⁻³ s² | **1,1 %** | 8,1320 s | 8,5106 s |

Coefficient de variation à 1,6 %, sous le seuil de 2 % : la mesure est stable malgré une fréquence
non verrouillée et la virtualisation WSL2 — c'est le warmup et le nombre de runs qui l'assurent.
Ces valeurs absolues ne sont pas des résultats du rapport ; seuls les ratios de gain mesurés sur ce
banc seront cités, en regard de ceux du banc A.

---

## 2. Diagnostic matériel & profiling réel — /5

> Les deux captures ci-dessous viennent du **banc B** (`results/a47848f/x86-controle/profiles/`),
> produites avec `make profile` puis `make flame` (menu *View > Flame Graph*). **À refaire sur le
> banc A**, qui fait foi, et **à annoter** avant rendu : le barème demande des captures annotées,
> pas brutes.

### 2.1 Profil CPU de la baseline

![Flamegraph CPU baseline](figures/flame-cpu-naive.png)

Annoter la capture : encadrer `Step` → `Map.At` → `fire.Mod`.

Relevé du banc B (`naive`, 1024², 64 foyers, 500 tours, 7,5 s de profil) :

| Fonction | CPU à plat | CPU cumulé |
|---|---|---|
| `naive.(*Sim).Step` | 49,7 % | 94,8 % |
| **`fire.Mod`** (le modulo torique) | **36,3 %** | 36,3 % |
| `fire.Map.At` | 3,5 % | 39,7 % |
| `fire.Map.WindAt` | 4,2 % | 10,3 % |
| `naive.(*Sim).Burning` | 2,7 % | 2,7 % |
| `runtime.gcBgMarkWorker` | — | 2,4 % |

**Le ramasse-miettes ne coûte que 2,4 %** alors que la baseline alloue 490 Mo sur l'exécution : les
allocations sont peu nombreuses mais énormes (1 Mio par tour), donc recyclées sans effort. `[F3]`
est un défaut de volume mémoire, pas de temps CPU — à ne pas présenter comme un goulot.

**`Fingerprint` n'apparaît pas dans ce profil**, et c'est normal : le binaire n'appelle pas
l'empreinte, faute de détection de cycles (REGLES.md §6). Son coût est établi par micro-benchmark
au §2.3, pas par ce profil.
Source : `results/<commit>/<banc>/profiles/naive-cpu-top.txt` et `naive-cpu-list.txt` (coût ligne par ligne, produits
par `make profile IMPL=naive`).

### 2.2 Profil d'allocations

![Flamegraph allocations baseline](figures/flame-alloc-naive.png)

Relevé du banc B, même exécution (596 Mo alloués au total) :

| Origine | Alloué | Part |
|---|---|---|
| `naive.(*Sim).Step` | **490 Mo** | 82,2 % |
| `fire.Generate` (hors chronomètre de la simulation) | 103 Mo | 17,3 % |

Les 490 Mo de `Step` sont exactement les 500 tampons d'ignition de 1 Mio alloués par les 500 tours
— `[F3]`, un tampon jeté et réalloué à chaque tour alors qu'un seul suffirait. À rapprocher des
`allocs/op` du §2.3 : **une** allocation par tour, mais d'un mégaoctet.

Les 103 Mo de `fire.Generate` sont hors du périmètre chronométré des micro-benchmarks ; ils
expliquent en revanche une part du temps mesuré par Hyperfine, et c'est ce qui a conduit à porter la
charge à 500 tours (§1.2).

### 2.3 Identification formelle du hot path

Pour chaque goulot, l'enchaînement **symptôme mesuré → cause mécanique → preuve** :

1. **`Fingerprint` coûte 1,7 fois un tour de simulation entier `[F6]`** — un `fmt.Sprintf` par case
   active, puis une conversion `string → []byte` et un SHA-256 : réflexion, allocation d'une chaîne
   à chaque appel. Mesuré à 1024², 64 foyers :

   | | `Step` (un tour) | `Fingerprint` (un appel) |
   |---|---|---|
   | Temps | 12,19 ms | **20,62 ms** |
   | Allocations | **1** | **451 200** |
   | Octets alloués | 1,000 Mio | 16,83 Mio |

   Le compte d'allocations et le volume alloué sont des propriétés du code, identiques sur les deux
   bancs ; seuls les temps sont à reconfirmer sur le banc A. Le rapport de 1,7 est le premier
   résultat exploitable de l'audit : **l'empreinte coûte plus cher que le calcul qu'elle
   accompagne**, alors qu'elle n'est qu'un moyen de détecter un état déjà vu.
2. **Le modulo torique, 36,3 % du CPU à lui seul `[F5]`** — et c'est le premier goulot en temps
   d'exécution, devant tout le reste. `fire.Mod` est appelé deux fois par voisin (une par axe) pour
   chaque case en feu, soit seize divisions entières par case, plus celles du saut du vent
   (REGLES.md §5). Coût ligne par ligne, relevé sur `internal/naive/naive.go` :

   | Ligne | Code | CPU cumulé |
   |---|---|---|
   | 104 | `ignite[s.carte.At(x+v[0], y+v[1])] = true` | **2,86 s / 7,5 s — 38 %** |
   | 97 | `d, vente := s.carte.WindAt(x, y)` | 0,85 s — 11 % |
   | 100 | `fire.Mod(j-int(d), 8)` (secteur amont) | 0,84 s — 11 % |

   Deux enseignements. D'abord, l'enroulement torique se paie par une division entière là où un
   masque suffirait pour des dimensions en puissance de deux, ou une bordure fantôme dans le cas
   général. Ensuite, `WindAt` coûte 11 % alors que **1 % des cases seulement portent du vent** : il
   est interrogé pour chaque case en feu et refait lui-même un `Map.At`, donc deux modulos, pour
   découvrir presque toujours qu'il n'y a pas de vent.
3. **Le balayage intégral `[F4]`** — `Step` parcourt toute la carte pour appliquer les transitions,
   alors que **seules les cases en feu propagent** (REGLES.md §4). C'est le gisement propre à ce
   modèle, celui qu'un automate à grille dense n'offre pas.
4. **Un tampon d'ignition alloué par tour `[F3]`** — 490 Mo sur l'exécution (§2.2), mais seulement
   2,4 % de CPU passé dans le GC : c'est un défaut de volume, pas de vitesse. `Burning()`, qui
   recompte toute la carte à chaque appel, pèse 2,7 %.

**Ordre de traitement qui en découle**, et qui n'est pas celui qu'on aurait supposé :

| Rang | Cible | Preuve | Gisement |
|---|---|---|---|
| 1 | `fire.Mod` `[F5]` | 36,3 % du CPU | masque ou bordure fantôme |
| 2 | `Fingerprint` `[F6]` | 1,7 × un tour, 451 200 allocs | hachage sans allocation |
| 3 | `WindAt` appelé partout | 11 % pour 1 % de cases ventées | index direct |
| 4 | balayage intégral `[F4]` | — | liste des cases actives |
| 5 | `[F1]`, `[F2]`, `[F3]` | GC à 2,4 % | gains de mémoire, pas de temps |

### 2.4 Deux pièges de lecture, à écarter avant d'interpréter

Ces deux observations ont été faites au cours de la mise au point, sur une autre machine que le banc
retenu : **elles sont méthodologiques, à reproduire ici avant d'être citées comme résultat.**

1. **Le débit en cases/s augmente avec la taille de la carte**, à nombre de foyers constant. Ce
   n'est pas une amélioration : plus la carte est grande, plus la fraction qui brûle est petite, et
   le débit compte des cases *balayées*, pas du travail utile. Pour comparer deux tailles, il faut
   garder la **densité de feu** constante — foyers proportionnels à la surface.
2. **Brider la machine ne change pas les ratios.** Réduire le nombre de cœurs, rendre le GC agressif
   ou contraindre la RAM laisse le temps d'exécution inchangé : la baseline est mono-thread et
   n'alloue qu'une fois par tour. Seule la contention CPU la ralentit, et linéairement. Corollaire
   utile : une machine plus lente ne fait pas apparaître un gain qui n'existe pas — c'est la
   mesure normalisée (cycles par case) qui démasque une implémentation coûteuse, pas le chronomètre.

**Une variance résiduelle, et sa cause.** Dans la campagne de référence, deux mesures restent
bruitées : `Step/embrasement/size=2048` (±13 %) et `Run/front/size=1024` (±19 %). Ce sont
précisément les deux qui allouent le plus — 4,000 Mio par tour et 52,03 Mio par itération — et le
GC y intervient à des moments variables. Autrement dit, **la baseline est instable parce qu'elle
alloue** : c'est `[F3]`, et ces deux lignes devraient se resserrer d'elles-mêmes dès la première
étape de l'axe mémoire. Conséquence à assumer d'ici là : sur ces deux lignes, un gain inférieur à
~20 % ne sera pas déclaré significatif par benchstat.

---

## 3. Journal d'optimisation — /5

Une entrée par étape, toujours au même format :

> **Étape N — titre** (commit `…`)
> - **Hypothèse d'impact matériel :** …
> - **Modification :** …
> - **Commande de vérification :** `…`
> - **Résultat (banc A, référence) :** avant → après (benchstat, avec p-value) ; allocs/op avant → après.
> - **Résultat (banc B, contrôle) :** le gain seul, en ratio.
> - **Portabilité :** l'écart entre les deux gains, et son explication. Un gain qui s'effondre d'un
>   banc à l'autre désigne une propriété matérielle précise — taille de cache, ligne de 64 contre
>   128 octets, nombre de cœurs réels, jeu d'instructions.
> - **Explication physique :** …

Chaque étape est un **package distinct**, enregistré dans le registre de `internal/fire` et ajouté à
`internal/engines` : la suite de conformité `internal/firetest` la rejoue automatiquement, et une
version qui casse une règle est rejetée avant d'être mesurée.

### 3.1 Mémoire & localité de cache

- Grille plate `[]uint8` + double tampon, et tampon d'ignition réutilisé : zéro allocation par tour.
- Compacité : `Cell` occupe 2 octets alors que l'état tient sur **4 bits** — `feu` ≤ 2 et `repos` ≤ 3
  tiennent chacun sur 2 bits (REGLES.md §2). Empreinte ÷ 4, puis ÷ 32 en bit-packé.
- Suppression des modulos : bordure fantôme ou traitement séparé des bords.
- **Bit-packing, sous la forme propre à ce modèle.** La contagion est un **OU logique** entre toutes
  les sources (REGLES.md §4) : il n'y a rien à *compter*. « Au moins un voisin en feu » s'écrit en
  huit décalages et sept `OR` sur des mots de 64 cases, là où un automate à comptage exigerait des
  demi-additionneurs SWAR. Les compteurs `feu` et `repos` se décrémentent en logique bit à bit sur
  des plans de bits séparés. Le vent reste traité à part, en boucle sur les seules cases ventées —
  ses cinq cibles et son saut dépendent d'une direction, donc ne se vectorisent pas.
- Struct padding : `unsafe.Sizeof(Cell{})` et `Sizeof(Sim{})` avant/après (`make layout`).
- Empreinte sans allocation : hachage FNV-1a direct sur les mots de la grille (`make escape` pour
  prouver l'absence d'échappement).
- **Liste des cases actives `[F4]`** : ne visiter que le front plutôt que toute la carte. À mesurer
  sur les *deux* scénarios — voir §4, le résultat n'est pas le même.

### 3.2 Concurrence & scalabilité CPU

- Worker pool : découpage en bandes horizontales, nombre de workers = cœurs **performance** (4), et
  non `GOMAXPROCS` (8) — justification au §1.1, réserve 1.
- Aucune synchronisation nécessaire *à l'intérieur* d'un tour : la contagion étant associative et
  commutative, deux bandes peuvent enflammer la même case sans verrou ni ordre imposé. Seule la
  barrière de fin de tour est requise (`sync.WaitGroup`), la mise à jour restant synchrone
  (REGLES.md §1).
- Compteurs (`Burning`) agrégés par `atomic.Int64` ou par réduction locale à chaque worker.
- Arrêt précoce : `context.WithCancel` / `WithTimeout`, annulation dès extinction (REGLES.md §6).
- Courbe de scalabilité : temps en fonction du nombre de workers (1, 2, 4, 8), comparée à la loi
  d'Amdahl, avec le décrochage attendu au-delà de 4.

### 3.3 I/O réseau & persistance

*Cet axe n'a encore aucun support dans le code : les snapshots, la base et le cache sont à écrire.
Plan de travail dans l'ordre ci-dessous, un commit par étape.*

- **Sérialisation** : snapshot de l'état d'un tour, d'abord en JSON naïf (baseline de l'axe : un
  tableau de structures par case), puis en **format binaire compact** — 4 bits par case, terrain et
  vent envoyés une seule fois puisqu'ils sont immuables. Mesurer les deux : taille du fichier et
  temps de sérialisation. Protobuf ou un encodage maison, à justifier.
- **Base de données** : historique des tours (tour, empreinte, cases en feu, surface brûlée). Une
  requête d'analyse — par exemple retrouver les tours dont l'empreinte se répète, ou la progression
  de la surface brûlée — exécutée **sans puis avec index**, avec `EXPLAIN ANALYZE` avant/après et le
  plan d'exécution commenté (`Seq Scan` → `Index Scan`).
- **Cache** : `sync.Pool` sur les tampons de sérialisation (les snapshots sont périodiques et de
  taille constante, c'est le cas d'usage idéal), et LRU des empreintes déjà vues.

---

## 4. Confrontation critique & échec constructif — /3

> **Tentative :** …
> - **Hypothèse initiale :** …
> - **Mesure :** régression de x % (benchstat, commit `…`)
> - **Explication mécanique chiffrée :** …
> - **Retour arrière :** commit `…` (git revert)

Trois pistes, par ordre d'intérêt :

1. **La liste des cases actives qui ne rapporte rien.** C'est le meilleur candidat, parce qu'elle
   n'échoue pas partout : elle écrase la baseline en scénario `front` (un foyer, presque rien ne
   brûle) et ne rapporte quasi rien en scénario `embrasement` (carte saturée), où le coût de tenir
   la liste à jour rejoint celui du balayage. Une optimisation dont le gain dépend du régime, chiffres
   à l'appui, vaut mieux qu'un échec franc.
2. **Une goroutine par case** : coût d'ordonnancement (~µs) contre coût de calcul d'une case (~ns).
3. **Des bandes trop fines** → *false sharing*, aggravé ici par la ligne de cache de **128 o** : deux
   workers qui écrivent à moins de 128 octets l'un de l'autre s'invalident mutuellement le cache.

---

## 5. Reproductibilité & synthèse comparative — /4

### 5.1 Reproduire toutes les mesures

```bash
git clone https://github.com/Antoine-Ferron/jdv-opti.git && cd jdv-opti
make tools   # benchstat
make bench   # env + tests + go bench + hyperfine + benchstat -> results/<date>-<commit>/
```

### 5.2 Tableau de synthèse

Coller `results/<run>/benchstat.txt` (benchstat `-col /impl`) et le tableau Hyperfine.

**Scénario `embrasement`** (64 foyers, carte saturée) :

| Version | Temps (1024², N tours) | Débit (cases/s) | allocs/tour | Gain cumulé |
|---|---|---|---|---|
| naive (baseline) | | | | ×1 |
| flat + double tampon | | | 0 | |
| bitpack | | | 0 | |
| parallel (N workers) | | | | |

**Scénario `front`** (1 foyer, carte creuse) :

| Version | Temps (1024², N tours) | Débit (cases/s) | allocs/tour | Gain cumulé |
|---|---|---|---|---|
| naive (baseline) | | | | ×1 |
| liste des cases actives | | | | |
| … | | | | |

**Portabilité des gains** — la seule table qui met les deux bancs en regard, et uniquement en ratios :

| Version | Gain cumulé — banc A (M1, ARM) | Gain cumulé — banc B (x86) | Écart, et pourquoi |
|---|---|---|---|
| flat + double tampon | | | |
| bitpack | | | |
| liste des cases actives | | | |
| parallel | | | |

Conclure en ordres de grandeur, sur la limite atteinte (calcul ou bande passante mémoire ?), sur le
fait qu'aucune version n'est la meilleure dans les deux régimes, et sur les gains qui ne survivent
pas au changement d'architecture — ce sont eux qui en disent le plus sur le matériel.

---

## 6. Bonus — gouvernance IA

Voir `constitution.md` à la racine du dépôt. Montrer qu'il répond aux quatre directives du barème :

1. **Rôle et posture** — §1 : ingénieur système raisonnant en cycles, lignes de cache et octets
   alloués, interdiction de proposer du code sans hypothèse mesurable.
2. **Contraintes négatives explicites** — §2 : interdits sur le hot path (`fmt.Sprintf`, conversions
   `string ↔ []byte`, allocations non justifiées, goroutines non bornées, `%` dans la boucle
   interne, `[][]T`, `map`, verrous par cellule).
3. **Justification empirique** — §3 : tout gain proposé sous la forme « Hypothèse → Vérification »,
   avec la commande exacte, et un gain non significatif (p > 0,05) déclaré comme tel.
4. **Format impératif** — §4 : injonctions vérifiables, prose limitée, chiffres avec unités.

Expliquer en quelques lignes comment il a été utilisé pendant le TP : ce qu'il a fait refuser, et ce
qu'il a fait mesurer avant d'accepter.
