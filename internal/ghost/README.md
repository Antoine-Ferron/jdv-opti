# Étape 3 — bordure fantôme

Hypothèse préalable : supprimer les modulos de propagation réduit le travail CPU.
Leur part de 7–9 % sur M1 suggère un plafond de ×1,08–1,10 si seul ce coût
était supprimé, avant le surcoût de bordure. Objectif à vérifier : quelques
pourcents de temps en moins en saturé, 0 allocation par Step. Aucun gain garanti
en front ; contrôle x86 nécessaire, sans additionner des coûts CPU imbriqués.

Nature mixte : macro (organisation du tampon d’ignition avec halo), micro
(adressage par décalages et masque de direction). Step reste O(N).

Un halo de deux cases reçoit les ignitions, y compris les sauts de vent.
Après propagation, ses cases sont fusionnées par OU vers leurs destinations
toriques. Les correspondances sont préparées à la construction, y compris
pour les dimensions 1 et 2. Aucun modulo ni allocation dans Step.
Les grilles d’état et le compteur sont hérités de counters. Fingerprint conserve
son formatage pour isoler cette étape ; ses allocations ne sont pas celles de Run.
Le halo ajoute 4W+4H+16 booléens et une table de correspondances de bordure.

Vérification avant commit :

```sh
make test
make escape IMPL=ghost
make quick IMPL=ghost
```

Après commit, arbre propre et machine disponible :

```sh
make bench-cpu BANC=m1-air
make profile IMPL=ghost BANC=m1-air
make flame IMPL=ghost BANC=m1-air
```

Comparer ghost à counters avec benchstat (Step, Run, allocations), puis le
binaire complet avec Hyperfine, en saturé et en front (50 tours, un foyer).
Mesurer le même commit et les mêmes charges sur x86. Résultats en attente.
