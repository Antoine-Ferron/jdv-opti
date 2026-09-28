# Étape 5 — listes actives

Hypothèse préalable : supprimer les parcours de toutes les cases et ne visiter
que les feux, les repos et les nouvelles ignitions réduit le travail en front.
Objectif expérimental : Run/front au moins 20 % plus rapide que counters,
0 allocation/Step. Comparer aussi à bitpack, meilleur moteur mesuré : aucun
gain garanti face à lui, ni en saturé. Vérifier le programme complet.

Macro-optimisation algorithmique : Step O(F+R+I) avec F cases en feu, R en repos
et I nouvelles ignitions (au plus 8F), au lieu du balayage systématique O(N).
Le pire cas reste O(N). Burning utilise la longueur de la liste, en O(1).
Cette variante scalaire dérive du modèle de counters, pas des plans de bitpack :
la comparaison avec bitpack porte donc sur deux stratégies, pas sur un ajout
isolé à bitpack. Le tore utilise des corrections aux bords sans modulo.

Les listes sont préallouées à N pour éviter les réallocations même en saturé.
Sur une architecture 64 bits, hors décor/structs, elles réservent environ
40N octets (deux listes de coordonnées et une d'indices), plus 3N pour états
et drapeaux. Ce compromis mémoire doit être mesuré et documenté.
Les listes sont compactées en place ; seules les ignitions touchées sont effacées.
L'éligibilité à l'allumage est testée avant les transitions, pour ne pas allumer
une case dont le repos vient d'atteindre zéro dans ce même tour.
Fingerprint conserve le formatage de counters pour isoler la stratégie de Step.

Avant commit :

```sh
make test
make escape IMPL=front
make quick IMPL=front
```

Après commit propre, sur M1 branché :

```sh
make bench-cpu BANC=m1-air
make profile IMPL=front BANC=m1-air
```

Compléter Hyperfine par 50 tours / 1 foyer, sur les mêmes cartes que bitpack et
counters. Utiliser le même commit et filtre sur x86. Documenter les régressions
avant de décider de conserver ou d'annuler la variante par revert.

## Contrôles réalisés

La suite complète passe, avec comparaison à counters et vérification des listes
sur 1 000 tours. Le contrôle quick termine 500 tours. L'analyse d'échappement
signale les append de Step/mark : ils pourraient allouer si la capacité était
dépassée. Ici chaque liste contient au plus N indices/coordonnées distincts,
et sa capacité N est réservée à New ; les tests AllocsPerRun mesurent zéro
allocation par tour. Ce résultat doit être confirmé par benchmem lors de la
campagne. Les allocations de Fingerprint sont héritées et hors de fire.Run.
