# Règles de la simulation d'incendie

Ce document est la référence. Chaque règle numérotée ici a son test dans
`internal/firetest`, et toute implémentation doit passer cette suite avant d'être mesurée.
En cas de désaccord entre le code et ce document, c'est le document qui a raison.

## 1. Le monde

- Grille rectangulaire **torique** : les bords se recollent (gauche↔droite, haut↔bas).
  Toute case a exactement 8 voisins, y compris dans les coins.
- **Voisinage de Moore** : les 8 cases adjacentes, diagonales comprises.
- **Mise à jour synchrone** : la propagation d'un tour est intégralement calculée à partir
  de l'état du tour précédent, puis appliquée. Aucune case ne voit le nouvel état d'une
  autre pendant le même tour.
- À graine égale, la carte **et** les foyers de départ sont identiques : deux exécutions
  produisent exactement la même partie.

## 2. Le terrain — immuable

| Terrain | Combustible | Combustion | Repos |
|---|---|---|---|
| **Eau** | non | — | — |
| **Plaine** | oui | 1 tour | 2 tours |
| **Forêt** | oui | 2 tours | 3 tours |

Une durée de combustion nulle **est** la définition de l'incombustible : c'est le seul test
à faire pour l'eau, il n'y a pas de cas particulier ailleurs dans les règles.

Le **vent** n'est pas un terrain mais un **attribut** posé sur une case, portant une
direction parmi 8. Une case vent est combustible si et seulement si le terrain en dessous
l'est : du vent sur de l'eau ne brûle pas, donc ne propage rien. Le générateur ne pose donc
du vent que sur des cases combustibles.

## 3. L'état d'une case — deux compteurs

- `feu` : nombre de tours de combustion restants. `0` = pas en feu.
- `repos` : nombre de tours avant de pouvoir brûler à nouveau. `0` = disponible.

## 4. Transition, pour chaque case, à chaque tour

1. **Si `feu > 0`** → la case enflamme ses cibles (§5), puis `feu -= 1`. Si `feu` atteint 0,
   elle passe en repos : `repos = repos(terrain)`.
2. **Sinon si `repos > 0`** → `repos -= 1`. Elle ignore toute contagion reçue.
3. **Sinon, si elle a été enflammée ce tour-ci et que son terrain est combustible** →
   `feu = combustion(terrain)`.

L'ordre de ces trois branches encode la règle **« une case en feu n'est jamais remise à
zéro »** : une case aux cas 1 ou 2 ignore la contagion, et cette contagion est **perdue** —
elle n'est pas mise en attente pour un tour ultérieur.

Conséquence recherchée : la contagion est un simple **OU logique** entre toutes les sources.
Commutatif, associatif, indépendant de l'ordre d'évaluation — donc décomposable par blocs
sans synchronisation.

Conséquence sur la durée d'un incendie : une plaine embrasée redevient inflammable 3 tours
plus tard (1 de combustion, 2 de repos), une forêt 5 tours. Ce que devient l'incendie dépend
alors du terrain — et cela se mesure, sur une carte 200×200 :

- **Terrain homogène, ou hétérogène mais sans vent** : le front s'éloigne d'une case par tour
  et ne revient jamais. L'incendie s'éteint après avoir parcouru la carte, en une centaine de
  tours, qu'il y ait des lacs, des massifs forestiers, ou les deux.
- **Vent et terrain hétérogène** — le réglage par défaut : le saut du vent allume des cases
  avec un tour d'avance, les obstacles et les durées de combustion inégales entretiennent ce
  décalage, et des fronts désynchronisés finissent par se rallumer mutuellement. L'incendie
  atteint un **régime entretenu** : après 4000 tours, un quart de la carte brûle encore.

Ni le vent seul sur de la plaine pure, ni l'hétérogénéité seule ne suffisent : il faut les
deux. Pour les mesures de performance, le réglage par défaut donne donc une charge de calcul
stable, ce qui est exactement ce qu'on veut.
## 5. Les cibles d'une case en feu

**Case sans vent** → ses 8 voisins de Moore, à distance 1. L'eau n'étant jamais combustible,
elle bloque : le feu la contourne, il ne la traverse pas.

**Case portant un vent de direction `k`** → deux modifications, valables **pour cette case
seulement** :

- **Secteur amont épargné** : les 3 voisins situés à l'opposé du vent ne sont pas enflammés.
  Les 8 directions étant numérotées dans l'ordre trigonométrique, ce sont les voisins `j`
  tels que `(j - k) mod 8 ∈ {3, 4, 5}` — la direction diamétralement opposée et les deux
  diagonales qui l'encadrent. Restent 5 voisins enflammés.
- **Saut** : la case à **distance 2** dans la direction `k` est enflammée en plus, sans
  considération de ce qui se trouve entre les deux. C'est le seul mécanisme permettant de
  franchir une rivière d'une case de large. La case intermédiaire (distance 1) brûle aussi,
  normalement, si elle est combustible.

```
   ·  x  x  .          x = enflammée          vent vers l'est
   ·  F  x  X          · = amont, épargné
   ·  x  x  .          X = saut à distance 2
```

**L'effet du vent ne se transmet pas.** Les cases allumées par une case vent sont des cases
ordinaires : dès le tour suivant elles repropagent dans leurs 8 directions, y compris vers
l'amont. Le vent est un coup de pouce local, pas un front directionnel.

## 6. Allumage et fin de partie

- Les foyers de départ sont tirés parmi les cases combustibles avec **la même graine** que
  le terrain. Leur nombre est réglable (défaut : 1).
- Pas de foudre en cours de partie : l'allumage est uniquement initial.
- La simulation s'arrête quand plus aucune case n'est en feu, quand le nombre de tours
  demandé est atteint, ou quand un état déjà vu réapparaît (arrêt précoce sur cycle).

## 7. Génération de la carte

Deux champs de bruit aléatoire, lissés indépendamment sur le tore par moyennes 3×3
successives : le nombre de passes fixe l'échelle des motifs, ce qui donne des massifs de
tailles variées sans bibliothèque externe.

- **Champ « altitude »** → l'eau. Les cases les plus basses forment **lacs et étangs** ;
  celles proches de la ligne de niveau médiane forment des **rivières**, rubans sinueux d'un
  à deux pas de large.
  *Réserve assumée : ces rivières suivent une ligne de niveau, elles ne s'écoulent pas vers
  le bas. Le rendu visuel est celui attendu, l'hydrologie ne l'est pas.*
- **Champ « végétation »** → sur ce qui n'est pas de l'eau : **forêt** au-dessus d'un seuil,
  **plaine** en dessous.
- **Vent** : faible proportion des cases combustibles (défaut 1 %), direction tirée parmi
  les 8.

Les seuils sont des **quantiles** des champs : demander 8 % d'eau en donne 8 %, quelle que
soit la graine.

## 8. Paramètres et valeurs par défaut

| Drapeau | Défaut | Rôle |
|---|---|---|
| `-w` / `-h` | 80 / 30 | Dimensions de la carte (`-size` pour une carte carrée) |
| `-seed` | 42 | Graine : carte **et** foyers |
| `-turns` | 500 | Nombre maximal de tours |
| `-fires` | 1 | Foyers de départ |
| `-wind` | 0.01 | Proportion de cases vent |
| `-lakes` | 0.08 | Proportion de lacs et étangs |
| `-rivers` | 0.04 | Proportion de rivières |
| `-forest` | 0.45 | Proportion de forêt hors de l'eau |
| `-scale` | 4 | Passes de lissage : plus haut = massifs plus vastes |

## Hors périmètre, décidé

- Pas de repousse ni de foudre en cours de partie : l'extinction est une fin normale.
- L'effet du vent ne se transmet pas aux cases qu'il enflamme (§5).
- Les rivières ne s'écoulent pas : ce sont des lignes de niveau (§7).
