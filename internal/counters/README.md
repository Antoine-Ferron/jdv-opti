# Étape 2 — compteur incrémental

Macro-optimisation algorithmique : `Burning()` passe de O(N) à O(1), où N
est le nombre de cases. `Step()` et le tour complet restent O(N).
Le compteur augmente à chaque allumage et diminue à chaque extinction.
Les foyers initiaux dupliqués et les foyers sur l’eau sont traités sans surcomptage.

Hypothèse : supprimer le balayage de `Burning` réduit le trafic mémoire,
avec 0 allocation/tour conservée. Les profils M1 attribuent à ce balayage
21 % du CPU en front et 3–6 % en saturé. À coût restant inchangé, cela donne
des plafonds théoriques de ×1,27 et ×1,03–1,06 sur la portion profilée,
pas des gains acquis pour le programme complet. Les mises à jour du compteur
ajoutent du travail à `Step` et peuvent réduire ces gains.

Pour isoler cette étape, les modulos et le formatage de `Fingerprint` sont
hérités de `flat` sans optimisation supplémentaire.

Vérification avant commit :

```sh
make test
make escape IMPL=counters
make quick IMPL=counters
```

Ces trois contrôles passent. L’analyse d’échappement signale des allocations
à la construction, à l’enregistrement et dans `Fingerprint` (formatage hérité
de `flat`, conservé pour isoler le compteur). `Fingerprint` n’est pas appelé
par `fire.Run`. Aucune allocation n’est signalée dans `Step` ou `Burning` ;
le test `TestStepNoAllocations` confirme 0 allocation par tour.
La durée affichée par `quick` est un contrôle fonctionnel, pas une mesure de référence.
`make layout` ne s’applique pas ici : cette étape ne travaille pas sur le padding.

Après commit du code, sur arbre propre, Mac branché et disponible :

```sh
make bench BANC=m1-air
make profile IMPL=counters BANC=m1-air
make flame IMPL=counters BANC=m1-air
```

Conserver les captures dans `rapport/figures/m1-air/`. Le banc de contrôle
mesure le même commit ; le rapport y compare les ratios de gain.
Documenter les résultats (y compris les régressions et les différences non
significatives), puis commiter le rapport avec les résultats et pousser la
branche pour ouvrir une PR. Une régression abandonnée est annulée par revert,
sans effacer ses mesures de l’historique.

Comparer `counters` à `flat`, avec `benchstat -col /impl` sur le `bench.txt`
de cette campagne : `Step` en régime établi, `Run` dans les deux scénarios,
et allocations. Confirmer avec Hyperfine sur le binaire complet : la campagne
standard mesure 500 tours / 64 foyers ; compléter par 50 tours / 1 foyer :

```sh
hyperfine -N --warmup 3 --runs 12 \
  -n 'flat front' './bin/wildfire -impl flat -size 1024 -turns 50 -fires 1 -quiet' \
  -n 'counters front' './bin/wildfire -impl counters -size 1024 -turns 50 -fires 1 -quiet'
```

Archiver les sorties dans le dossier du commit mesuré et du banc concerné.
Résultats chronométriques : à mesurer sur M1, puis contrôle de portabilité x86.
