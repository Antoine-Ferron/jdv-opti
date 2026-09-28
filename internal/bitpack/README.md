# Étape 4 — bitpack

Hypothèse avant implémentation : quatre plans de bits représentent feu et repos
sur 4 bits/case au lieu de 2 octets/case par grille Cell (division par quatre,
hors arrondi de chaque ligne à 64 cases, décor et buffers auxiliaires).
Propagation sans vent par décalages et OU de mots de 64 bits, transitions par
opérations booléennes. Objectif expérimental : Step saturé au moins 20 % plus
rapide que ghost, zéro allocation/tour. Aucun gain garanti avant mesure.

Optimisation mixte : macro pour la représentation et le traitement par blocs,
micro pour les décalages, masques et popcount. Complexité O(H*ceil(W/64)+V),
où V est le nombre de cases ventées traitées séparément (donc toujours O(N)
à densité de vent fixe). Burning reste O(1).

Le tore est traité par retenues entre mots et bouclage des lignes ; le dernier
mot est masqué pour les largeurs non multiples de 64. Le halo de booléens de
ghost est remplacé par ces opérations sur les mots. Les cibles du vent sont
précalculées à la construction ; cela ajoute du stockage proportionnel à V.
Fingerprint conserve le format des versions précédentes pour isoler cette étape.

Vérification avant commit :

```sh
make test
make escape IMPL=bitpack
make quick IMPL=bitpack
```

Après commit du code, sur arbre propre, banc M1 branché :

```sh
make bench-cpu BANC=m1-air
make profile IMPL=bitpack BANC=m1-air
make flame IMPL=bitpack BANC=m1-air
```

Comparer bitpack à ghost dans la même campagne, sans masquer les régressions.
Confirmer sur le binaire complet, en saturé et en front (50 tours, 1 foyer),
puis mesurer le même commit sur x86. Résultats chronométriques en attente.
