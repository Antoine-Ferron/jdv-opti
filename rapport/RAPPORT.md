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

| Banc                                    | Rôle          | Ce qu'on en tire                                                           |
|-----------------------------------------|---------------|----------------------------------------------------------------------------|
| **A — MacBook Air M1** (ARM)            | **référence** | Tous les chiffres cités dans ce rapport, sauf mention contraire explicite. |
| **B — Intel Core Ultra 9 / WSL2** (x86) | contrôle      | Uniquement le *ratio* de gain de chaque étape, comparé à celui du banc A.  |

Règle appliquée sans exception : **aucun tableau ne mélange les deux bancs**, et aucune valeur
absolue du banc B n'est citée comme résultat. Comparer 2,1 s sur M1 à 0,8 s sur x86 n'apprend rien ;
comparer un gain de ×4,1 à un gain de ×3,8 apprend que l'optimisation est portable.

#### Banc A — référence

Source du relevé M1 : [env.md](../results/dcc3622/m1-air/env.md), campagne du 23 septembre 2026.
Les valeurs de cache sont celles exposées par sysctl, sans description exhaustive de la topologie.

| Élément                   | Valeur                                                                             |
|---------------------------|------------------------------------------------------------------------------------|
| CPU (modèle)              | Apple M1                                                                           |
| Cœurs physiques / threads | 8 / 8 — **4 performance + 4 efficiency**, pas de SMT                               |
| L1i / L1d (par cœur)      | 128 Kio / 64 Kio                                                                   |
| L2                        | 4 Mio                                                                              |
| L3                        | pas de L3 classique — *à préciser (System Level Cache)*                            |
| Ligne de cache            | **128 o**                                                                          |
| RAM                       | 8 Gio unifiée                                                                      |
| OS                        | macOS 15.5 (build 24F74)                                                           |
| Runtime                   | Go 1.27.1, darwin/arm64, CGO_ENABLED=1, GOGC=100 ; variable GOMAXPROCS non définie |
| SIMD disponibles          | NEON 128 bits (pas d'AVX : architecture ARM)                                       |
| Alimentation              | **Sur secteur**, mode économie d’énergie désactivé (déclaré par Cherif)            |

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
4. **Fréquence non verrouillée, et châssis sans ventilateur.** Apple Silicon ne laisse ni piloter le
   gouverneur ni figer le turbo, et le MacBook Air est refroidi passivement. Sur une charge de ~6,4 s
   répétée dix-huit fois, la machine chauffe et se bride de façon irrégulière : **le CV du banc A est
   de 3,1 %, contre 0,7 % sur le banc B** (§1.3). C'est la réserve la plus gênante de ce banc,
   puisque c'est lui qui fait foi — voir le §1.2 pour ce qu'on en déduit.

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

Source : [env.md](../results/dcc3622/x86-controle/env.md). Trois réserves, qui expliquent pourquoi ce
banc ne fournit que des ratios :

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
- Une campagne = `make bench BANC=<nom>`, qui écrit dans `results/<commit>/<banc>/`. Le commit retenu
  n'est pas `HEAD` mais **le dernier ayant touché le code ou le protocole** (`internal`, `cmd`,
  `go.mod`, `Makefile`, `scripts`) : un commit qui n'ajoute que des résultats ou du rapport ne
  déplace pas la référence, si bien que les deux machines rangent leurs mesures au même endroit sans
  avoir à se synchroniser sur un hash. `env.md` porte les deux, `HEAD` et le code mesuré.
- Le pipeline **refuse de mesurer sur un arbre de travail modifié** : un dossier de résultats doit
  toujours correspondre exactement au code qu'il nomme.

> Les deux campagnes de référence ci-dessous partagent le dossier `results/dcc3622/`, mesuré sur les
> deux bancs le même jour : `m1-air` et `x86-controle`.
- Isolation du bruit : navigateur/IDE fermés, machine sur secteur, charge système vérifiée avant
  chaque campagne (voir `env.md`), [pinning si utilisé].

**La dispersion du banc de référence est sa faiblesse, et passer sur secteur ne l'a pas corrigée.**
Une première campagne M1 sur batterie donnait un CV de 1,7 % ; la même sur secteur donne **3,1 %**,
au-dessus du seuil de 2 % qu'on s'était fixé — le refroidissement passif du MacBook Air le bride
d'autant plus qu'il monte plus haut en fréquence (§1.1, réserve 4). Ce qu'on en tire :

- une moyenne sur 15 exécutions a une erreur standard de 3,1 % / √15 ≈ **0,8 %**, ce qui reste très
  inférieur aux gains attendus (plusieurs dizaines de pour cent) : les comparaisons d'étapes restent
  exploitables ;
- en revanche, **aucun écart inférieur à ~3 % ne sera déclaré significatif sur ce banc**, et les
  micro-benchmarks `go test -count 10` comparés par benchstat, qui donnent une p-value, primeront sur
  la ligne Hyperfine pour trancher les cas serrés ;
- piste d'amélioration si un cas serré se présente : augmenter `RUNS`, ou intercaler un temps de
  repos entre les exécutions (`hyperfine --prepare 'sleep 2'`) pour laisser le châssis refroidir.
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

| Charge        | Temps total | Dont simulation | Part de la génération |
|---------------|-------------|-----------------|-----------------------|
| 50 tours      | 0,84 s      | 0,178 s         | **79 %**              |
| **500 tours** | 8,15 s      | 7,49 s          | **8 %**               |

À 50 tours, une optimisation qui diviserait `Step` par deux n'aurait apparu que comme ~10 % sur la
ligne Hyperfine. **La charge est donc fixée à 500 tours**, ce qui ramène la génération sous 10 % du
temps mesuré.

Second argument, indépendant : le débit passe de 2,9 × 10⁸ à 7,0 × 10⁷ cases/s entre 50 et 500
tours. À 50 tours l'incendie n'a pas fini de s'étendre — on mesure un transitoire ; à 500 tours il
a atteint le régime entretenu décrit dans `REGLES.md` §4. Seule la seconde mesure est représentative
de ce que fait le programme.

### 1.3 Mesures de référence

#### Banc A — référence

Campagne sur le commit `dcc3622`, **machine sur secteur**, carte 1024², 64 foyers, 500 tours
demandés, avec 3 échauffements puis 15 exécutions Hyperfine.
Source : [statistiques](../results/dcc3622/m1-air/hyperfine-stats.md).

| Moyenne  | Médiane  | Écart-type | Variance   | CV        | Min      | Max      |
|----------|----------|------------|------------|-----------|----------|----------|
| 6,4229 s | 6,4606 s | 0,1968 s   | 0,03872 s² | **3,1 %** | 6,1872 s | 6,9199 s |

La dispersion dépasse le seuil de 2 %, et le passage sur secteur l'a **aggravée** : une première
campagne sur batterie, [conservée pour comparaison](../results/a47848f/m1-air/hyperfine-stats.md),
donnait 1,7 % (6,3767 s de moyenne). L'explication la plus probable est le refroidissement passif du
châssis (§1.1, réserve 4), qui bride la machine d'autant plus qu'elle monte plus haut en fréquence.
Le §1.2 précise ce qu'on en déduit : les comparaisons d'étapes restent exploitables, mais aucun écart
inférieur à ~3 % ne sera déclaré significatif sur cette seule ligne.

Le temps inclut le démarrage du processus et la génération de carte. Les micro-benchmarks `Run`
restent fixés à 50 tours ; leurs temps ne sont pas directement comparables à cette mesure globale.

#### Banc B — contrôle

Campagne de référence, commit `dcc3622`, carte 1024², 64 foyers, 500 tours, Hyperfine `-N --warmup 3
--runs 15` (`results/dcc3622/x86-controle/`) :

| Moyenne  | Médiane  | Écart-type | Variance        | CV        | Min      | Max      |
|----------|----------|------------|-----------------|-----------|----------|----------|
| 8,4021 s | 8,4104 s | 0,0611 s   | 3,729 × 10⁻³ s² | **0,7 %** | 8,2926 s | 8,4825 s |

Coefficient de variation à 0,7 %, bien sous le seuil de 2 % : la mesure est stable malgré une
fréquence non verrouillée et la virtualisation WSL2 — c'est le warmup et le nombre de runs qui
l'assurent. Fait notable : **le banc de contrôle est quatre fois plus stable que le banc de
référence**, dont le châssis se bride (§1.1, réserve 4).
Ces valeurs absolues ne sont pas des résultats du rapport ; seuls les ratios de gain mesurés sur ce
banc seront cités, en regard de ceux du banc A.

---

## 2. Diagnostic matériel & profiling réel — /5

> **Captures : `rapport/figures/<banc>/<vue>-<implémentation>.png`.** Carte 1024 × 1024, 64 foyers,
> 500 tours demandés (`make profile BANC=<nom>`). **Aucune capture n'est retouchée** : ce sont les
> sorties brutes de pprof, et l'annotation tient dans la légende, qui nomme les zones à lire et les
> chiffre. Une campagne rejouée ne demande ainsi que la mise à jour du texte — d'autant que les
> pourcentages d'un profil ont une incertitude de plusieurs points d'une exécution à l'autre, à code
> identique.

### 2.1 Profil CPU de la baseline

![Flamegraph CPU baseline sur M1](figures/m1-air/flame-cpu-naive.png)
*Figure 1 — **Banc A** (Apple M1, macOS), celui qui fait foi. `naive`, carte 1024², graine 42,
64 foyers, 500 tours, commit `dcc3622`, machine sur secteur. Profil :
`results/dcc3622/m1-air/profiles/naive-cpu.prof`, 5,34 s d'échantillons sur 6,16 s. Le calcul dans
Step domine le CPU ; les indices toriques Map.At → Mod, le comptage Burning et `runtime.madvise` —
le retour de mémoire au système — sont également visibles.*  
Sources : [profil CPU](../results/dcc3622/m1-air/profiles/naive-cpu-top.txt)
et [détail par ligne](../results/dcc3622/m1-air/profiles/naive-cpu-list.txt).

Chiffres correspondants :  

| Fonction | Part directe (flat) | Part appels inclus (cum) |
|----------|--------------------:|-------------------------:|
| Step     |             73,78 % |                  87,45 % |
| Burning  |              6,74 % |                   6,74 % |
| `runtime.madvise` |     5,43 % |                   5,43 % |
| Mod      |              5,24 % |                   5,24 % |
| Terrain.Combustion |    2,62 % |                   2,62 % |
| Map.At   |              2,43 % |                   7,30 % |

Le profil couvre 6,16 s, avec 5,34 s d'échantillons CPU. Les pourcentages
portent sur ces échantillons, pas sur le temps Hyperfine. Les coûts cumulés
s'incluent : Map.At et Mod sont notamment compris dans Step et ne doivent pas
être ajoutés à ses 87,45 %. Le profil CPU commence après la génération de carte.
Fingerprint n'est pas appelé par fire.Run et n'apparaît donc pas dans ce profil.

Deux observations propres à ce banc. **`Burning` y coûte plus cher que `Mod`** — 6,74 % contre
5,24 % : recompter toute la carte à chaque appel pèse davantage, ici, que l'enroulement torique.
Et `runtime.madvise`, à 5,43 %, est la trace des tampons jetés à chaque tour : le noyau rend la
mémoire au système puis la redemande. C'est un coût CPU imputable aux allocations, que le seul
`gcBgMarkWorker` du profil x86 ne laissait pas voir.

#### Observation complémentaire — banc B (x86, hors référence M1)

![Flamegraph CPU baseline sur x86](figures/x86-controle/flame-cpu-naive.png)
*Figure 2 — **Banc B** (Intel Core Ultra 9 275HX, WSL2), contrôle de portabilité. Même code et même
charge que la figure 1. Profil : `results/dcc3622/x86-controle/profiles/naive-cpu.prof`, 7,94 s
d'échantillons sur 7,71 s. Deux zones à lire :*
- *le large bloc central `fire.Map.At → fire.Mod`, directement sous Step : **3,31 s, soit 42 % du
  CPU** — c'est `naive.go:104`, deux divisions entières par voisin et huit voisins par case en feu ;*
- *la pile de droite `fire.Map.WindAt → fire.Map.At → fire.Mod` : **0,82 s, 10 % du CPU**, alors
  qu'1 % seulement des cases portent du vent — c'est `naive.go:97`, où WindAt refait lui-même un
  Map.At pour découvrir presque toujours qu'il n'y a pas de vent.*

*Ni l'une ni l'autre n'a d'équivalent sur la figure 1 : Mod y pèse 5,2 % contre 41,1 % ici.*  
Source : [profil CPU x86](../results/dcc3622/x86-controle/profiles/naive-cpu-top.txt).
Ce profil couvre 7,71 s et totalise 7,94 s d'échantillons CPU.

| Fonction               | CPU direct | CPU cumulé |
|------------------------|-----------:|-----------:|
| Step                   |    46,22 % |    94,84 % |
| Mod                    |    41,06 % |    41,18 % |
| Map.At                 |     2,52 % |    43,70 % |
| Map.WindAt             |     3,78 % |    10,33 % |
| Burning                |     2,27 % |     2,27 % |
| runtime.gcBgMarkWorker |        0 % |     2,35 % |

Le modulo apparaît proportionnellement plus coûteux sur ce profil x86 que sur
M1. Cette observation motive une vérification de portabilité, sans établir
à elle seule la cause matérielle. Les 2,35 % de gcBgMarkWorker ne représentent
pas tout le coût des allocations ou du GC : ils ne prouvent pas que supprimer
le tampon alloué par tour serait sans effet sur le temps CPU.
Fingerprint est absent des deux profils car fire.Run ne l'appelle pas.

### 2.2 Profil d'allocations

![Flamegraph allocations baseline sur M1](figures/m1-air/flame-alloc-naive.png)
*Figure 3 — **Banc A** (Apple M1, macOS). Même exécution que la figure 1, commit `dcc3622`. Profil :
`results/dcc3622/m1-air/profiles/naive-mem.prof`, échantillonné à `MemProfileRate = 4096` octets.
Sur cette exécution, le tampon d'allumage créé par Step domine les allocations ; la génération
initiale de la carte contribue également au volume total.*  
Source : [profil alloc_space](../results/dcc3622/m1-air/profiles/naive-mem-top.txt).

Le volume cumulé estimé est de 595,28 MB dans les unités affichées par pprof :
Step représente 82,15 % et Generate, appels inclus, 17,33 %.
Il ne s'agit ni du pic de mémoire ni de la mémoire conservée en fin d'exécution.
Contrairement au profil CPU, le profil cumulatif d'allocations inclut la génération
initiale. Les estimations échantillonnées ne remplacent pas les mesures par opération.

Les [micro-benchmarks](../results/dcc3622/m1-air/benchstat.txt) mesurent
1 allocation de 1 Mio par Step à 1024². Le benchmark isolé de Fingerprint
mesure environ 451 200 allocations et 16,82 Mio par appel ; cette opération
ne fait pas partie de l'exécution normale profilée ici.

Une mesure spécifique des
pauses et du temps GC reste à effectuer ; elle ne se déduit pas du volume alloué.

#### Observation complémentaire — allocations du banc B

![Flamegraph allocations baseline sur x86](figures/x86-controle/flame-alloc-naive.png)
*Figure 4 — **Banc B** (Intel Core Ultra 9 275HX, WSL2). Même code, même charge, même commit que la
figure 3. Profil : `results/dcc3622/x86-controle/profiles/naive-mem.prof`. La figure est
superposable à la figure 3 : c'est le résultat à retenir, et il se vérifie ici plutôt que de
s'affirmer.*  
Source : [profil alloc_space x86](../results/dcc3622/x86-controle/profiles/naive-mem-top.txt).

Mise en regard des deux bancs :

|                           |         Banc A (M1) |        Banc B (x86) |
|---------------------------|--------------------:|--------------------:|
| Total estimé              |           595,34 MB |           596,38 MB |
| `Step`                    |    489 MB — 82,14 % |    491 MB — 82,33 % |
| `Generate`, appels inclus | 103,14 MB — 17,32 % | 102,22 MB — 17,14 % |
| `champLisse`              |      32 MB — 5,38 % |      32 MB — 5,37 % |
| `quantile`                |   31,05 MB — 5,21 % |   31,05 MB — 5,21 % |

**Les deux profils coïncident à 0,2 point près**, écart imputable à l'échantillonnage de pprof. C'est
la vérification du principe utilisé ailleurs dans ce rapport : **le volume alloué est une propriété
du code, pas de la machine**, et un compte d'allocations mesuré sur un banc vaut pour l'autre.

Le contraste avec le §2.1 en est d'autant plus instructif : *le même volume alloué* se paie
différemment selon le système — `runtime.madvise` à 5,43 % du CPU sur macOS, `gcBgMarkWorker` à
2,35 % sur Linux. Volume identique, coût CPU différent : ces deux profils ne mesurent pas la même
chose, et seul celui du CPU dit ce que les allocations coûtent réellement.

Ces estimations cumulées ne sont pas un comptage exact des 500 tampons ; le micro-benchmark confirme
séparément 1 Mio par `Step`. `Generate` est hors chronométrage des micro-benchmarks de simulation,
mais fait partie du processus mesuré par Hyperfine.

### 2.3 Identification formelle du hot path

1. **Step : 90,48 % du CPU, appels inclus.** La propagation puis les transitions
   balayent la grille. Map.At et Mod contribuent au calcul des positions toriques.
   Source : profil CPU ci-dessus et `internal/naive/naive.go`.
2. **Tampon d'allumage : 1 Mio alloué par tour en 1024².** Step construit un
   nouveau tableau booléen à chaque appel. Le réutiliser est une hypothèse
   mesurable de réduction des allocations, pas encore une preuve de gain CPU.
3. **Burning : 6,74 % du CPU**, soit davantage que le modulo torique sur ce banc. Le nombre de cases
   en feu est recalculé par un balayage complet à chaque appel.
4. **Fingerprint : coût isolé, hors du chemin de fire.Run.** Sur le banc M1,
   sa médiane est de 24,05 ms contre 10,03 ms pour Step dans le scénario
   embrasement, soit environ 2,40 fois. Ces benchmarks utilisent des états et
   protocoles différents (empreinte sur état fixe, Step sur état évolutif) :
   ce ratio n'est pas une part du temps de simulation. Le formatage des coordonnées
   explique ses nombreuses allocations, mais son optimisation seule n'accélérera
   pas fire.Run tant que celui-ci ne l'appelle pas.

Sources des micro-mesures : [benchstat](../results/dcc3622/m1-air/benchstat.txt)
et [résultats bruts](../results/dcc3622/m1-air/bench.txt). Les comptes d'allocations
sont ceux mesurés sur M1 ; leur égalité sur un autre environnement doit être vérifiée.

#### Le même filtre sur les deux bancs

Les deux captures suivantes sont les mêmes vues que les figures 1 et 2, avec `Mod` saisi dans le
champ *Search regexp* de pprof : l'outil encadre lui-même les cadres correspondants, sans retouche
d'image, et le champ reste visible dans la capture — n'importe qui peut la reproduire.

![Filtre Mod sur le profil M1](figures/m1-air/flame-cpu-mod-naive.png)  
*Figure 5 — **Banc A** (M1). Le cadre `fire.Mod` encadré occupe environ 5 % de la largeur du
graphe ; `Burning`, à sa droite, est visiblement plus large — il coûte effectivement plus cher
(6,74 % contre 5,24 %).*

![Filtre Mod sur le profil x86](figures/x86-controle/flame-cpu-mod-naive.png)
*Figure 6 — **Banc B** (x86). Même filtre, même code, même charge, même commit : le cadre `fire.Mod`
occupe environ 35 % de la largeur, soit sept fois plus qu'à la figure 5 (41,1 % du CPU contre 5,2 %).*

**C'est le résultat central du diagnostic**, et il n'aurait pas été visible avec un seul banc : la
division entière du modulo torique domine le profil x86 et reste marginale sur ARM. Une optimisation
qui la supprime doit donc être attendue comme un gain majeur sur le banc B et modeste sur le banc A
— hypothèse à vérifier par la mesure, l'écart de coût d'instruction entre les deux jeux
d'instructions n'étant pas établi par ces seuls profils.

#### Pistes de portabilité issues du diagnostic x86
Le [profil par ligne x86](../results/dcc3622/x86-controle/profiles/naive-cpu-list.txt)
met en évidence le calcul des cibles via Map.At, la lecture du vent via WindAt
et le calcul du secteur amont. WindAt est interrogé pour chaque case en feu,
même si elle ne porte pas de vent ; il recalcule des indices toriques.

Les pistes à comparer sur les deux bancs sont :

- Enroulement torique : masque pour les dimensions en puissance de deux,
  traitement général pour les autres dimensions.
- Lecture du vent par indice direct, sans recalcul de coordonnées valides.
- Réutilisation du tampon et stockage contigu, pour mesurer le gain réel en
  allocations et en temps, sans l'inférer du seul profil GC.
- Liste de cases actives, à valider selon la densité de feu.
- Empreinte sans formatage, pour ses usages propres ; elle n'accélère pas
  l'exécution actuelle de fire.Run.

Pour mémoire, les valeurs correspondantes du banc B sur la campagne courante
(son [benchstat](../results/dcc3622/x86-controle/benchstat.txt)) : 12,81 ms ± 1 %
pour Step/embrasement/1024 et 20,60 ms ± 4 % pour Fingerprint/1024, soit un rapport
de 1,61 contre 2,40 sur M1. Ces données restent un diagnostic du banc B, pas la référence.

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

**Dispersion, et elle n'est pas où on l'attendrait.** Sur la campagne `dcc3622`, les
micro-benchmarks du banc B affichent ±12 % pour Run/front/512 et **±18 %** pour Run/front/1024,
quand les mêmes mesures sur le banc A tiennent ±1 %. C'est l'inverse d'Hyperfine, où le banc A est
le moins stable (CV 3,1 % contre 0,7 %) : les micro-benchmarks sont assez courts pour échapper au
bridage thermique du MacBook Air, tandis que le scénario `front` a peu d'itérations et subit les
allocations. Les allocations et le GC restent des hypothèses d'explication ; une attribution causale
exigerait des mesures complémentaires. Aucun seuil universel de gain ne peut en être déduit : la
significativité sera évaluée sur les échantillons comparés, banc par banc.

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

### Étape 1 — `flat` : stockage contigu et tampons réutilisés

**Nature : mixte, selon les définitions du cours.** Macro : remplacement de
`[][]Cell` par deux grilles contiguës `[]Cell`, modification de la structure des
données. Micro : suppression du tampon d'allumage temporaire, désormais alloué
une fois et réinitialisé avec `clear` à chaque tour.

**Hypothèse :** passer de 1 allocation de 1 Mio par Step à 0 allocation et
0 octet par tour sur 1024². Cette hypothèse porte sur les allocations ;
le gain CPU éventuel doit être établi séparément.

**Implémentation :** la propagation lit la grille courante et marque un tampon
`[]bool`. Les transitions écrivent toutes les cellules dans la seconde grille,
puis les deux grilles sont échangées. Réécrire chaque cellule et remettre les
marques à zéro évite les états périmés. Le double tampon suit le plan convenu,
mais n'est pas indispensable à la synchronisation de cette baseline déjà en
deux phases ; son coût en mémoire et en écritures doit donc être évalué.

**Complexité :** O(N) par Step et O(N) en mémoire avant et après, N étant le
nombre de cases et le nombre de voisins étant borné. Burning reste un balayage
O(N). Avec Cell de 2 octets, les deux grilles et le tampon représentent 5N octets
de données persistantes (5 Mio en 1024²), hors carte et en-têtes. Ce changement
réduit le volume alloué par tour, pas nécessairement la mémoire résidente.

**Périmètre contrôlé :** règles, modulos, lecture du vent, recomptage Burning
et formatage/SHA-256 de Fingerprint sont conservés. Le maintien du formatage et
des modulos est une exception expérimentale aux interdits de constitution.md
pour isoler cette étape. New et Fingerprint allouent toujours.
La baseline naive et la référence firetest restent intactes.

**Validation réalisée :** `go test ./... -count=1` passe. La suite commune est
complétée par une comparaison cellule par cellule et des empreintes sur 100 tours
avec vent et dimensions impaires, un test d'extinction sans contagion résiduelle,
et un test `AllocsPerRun` qui vérifie zéro allocation par Step sur 67 × 45.
L'analyse `go build -gcflags=-m ./internal/flat` documente les allocations du
constructeur et de l'empreinte ; Step ne contient pas d'allocation de tampon.

**Commande de campagne exécutée sur le banc M1, commit de code `1d4a064` :**

```bash
make bench BANC=m1-air
```

**Résultats banc A — Apple M1.** Sources :
[benchstat](../results/1d4a064/m1-air/benchstat.txt),
[mesures brutes](../results/1d4a064/m1-air/bench.txt) et
[tests](../results/1d4a064/m1-air/tests.txt).

| Mesure | naive | flat | Conclusion |
|---|---:|---:|---|
| Allocations par Step, 1024² | 1 | 0 | Objectif atteint |
| Octets alloués par Step, 1024² | 1 Mio | 0 | Tampon temporaire supprimé |
| Temps médian Step/embrasement, 1024² | 9,982 ms | 9,973 ms | Pas de différence significative, p = 0,393 |
| Temps médian Step/embrasement, 2048² | 23,62 ms | 24,07 ms | flat environ 1,9 % plus lent, p = 0,004 |

Sur BenchmarkRun en 1024², le volume cumulé alloué passe de 52,03 Mio à
5 Mio, soit environ 90,4 % de réduction ; les allocations passent de 1 076
à 4 par exécution, construction du moteur comprise. Il ne s'agit pas du pic de
mémoire occupée. Step n'alloue plus pour les trois tailles testées.

Les résultats CPU sont mixtes : Step en 256² et Run/front en 512² s'améliorent,
mais Run/front en 1024², Run/embrasement aux deux tailles et Fingerprint
présentent de petites régressions significatives dans cette campagne.
Benchstat prend flat comme référence : un pourcentage négatif dans la colonne
naive indique que naive est plus rapide. Les p-values affichées `0.000` sont
arrondies, pas nulles.

**Temps global — 500 tours demandés.** Source :
[statistiques Hyperfine](../results/1d4a064/m1-air/hyperfine-stats.md).

| Moteur | Temps moyen | Écart-type | CV |
|---|---:|---:|---:|
| naive | 6,3787 s | 0,1496 s | 2,3 % |
| flat | 6,3188 s | 0,0333 s | 0,5 % |

La diminution observée de la moyenne est d'environ 0,9 %. Ce rapport de
moyennes ne suffit pas à établir un gain global statistiquement significatif.
Hyperfine inclut la génération de carte et l'exécution du processus ; les
micro-benchmarks Run de cette version ne calculent que 50 tours maximum.

**Interprétation mémoire–CPU.** La réduction des allocations est démontrée,
mais elle ne produit pas une accélération systématique. Le double tampon ajoute
une grille persistante de 2 Mio en 1024² et modifie les accès mémoire. Son rôle
dans les régressions est une hypothèse à tester avec une variante sans second
tampon de cellules ; aucun profil matériel ne prouve ici la cause des écarts.

**Banc B et portabilité :** en attente de mesures x86 sur le même commit de
code `1d4a064`. Aucun gain de portabilité n'est revendiqué à ce stade.

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

### Résultat mitigé observé — double tampon de flat

**Tentative :** grille contiguë, double tampon de cellules et tampon d'allumage
réutilisé, commit `1d4a064` ; optimisation mixte macro/micro (§3, étape 1).

**Hypothèse initiale :** supprimer les allocations temporaires par tour pour
réduire leur coût, tout en améliorant le stockage des cellules.

**Mesure :** zéro allocation par Step est atteint. Pourtant, en scénario
embrasement 2048², Step passe de 23,62 ms à 24,07 ms, soit une régression
d'environ 1,9 % (p = 0,004). En 1024², aucun gain significatif n'est établi
(p = 0,393). Source : [benchstat](../results/1d4a064/m1-air/benchstat.txt).

**Explication à vérifier :** le second tampon ajoute 2N octets de stockage
persistant et change les accès mémoire. Cette cause n'est pas démontrée ; une
variante à une seule grille permettra d'isoler son effet en conservant la
réutilisation du tampon d'allumage. La mise à jour en deux phases autorise cette
variante sans modifier les règles de propagation.

**Enseignement :** réduire le volume cumulé alloué ne garantit pas une réduction
du temps CPU ni du pic de mémoire.

**Retour arrière :** aucun revert effectué à ce stade. La version et ses
résultats sont conservés pour la comparaison ; la variante reste à implémenter
et à mesurer avant de décider de la version à retenir.

### Autres pistes à explorer

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
