# Banc d'essai — contrôle d'environnement, hôte Windows natif
Commit : 8a71072
Code mesuré : 8a71072 (dernier commit touchant le code ou le protocole)

Ce dossier n'est **pas** un banc du protocole. C'est un contrôle ponctuel, limité à
l'axe I/O (§3.3), destiné à mesurer ce que la frontière WSL coûte sur un
aller-retour vers PostgreSQL. Les campagnes du protocole restent sous
`x86-controle` (WSL2), seul banc comparable à l'historique.

## Matériel
CPU : Intel(R) Core(TM) Ultra 9 275HX — 24 cœurs physiques / 24 logiques
RAM : 63,4 Gio (l'hôte entier ; WSL n'en voit que 31 Gio)
Alimentation secteur branchée

## Système
Microsoft Windows 11 Famille 10.0.26200
Pas de virtualisation entre le programme et le système.

## Runtime
go version go1.27.1 windows/amd64
GOOS GOARCH GOAMD64 CGO_ENABLED : windows amd64 v1 0
GOMAXPROCS par défaut : nombre de CPU logiques (24)
GOGC=défaut (100)

## Base
PostgreSQL 17.11 (Alpine), **le même conteneur** que pour le banc `x86-controle` :
seul le chemin client → serveur change, jamais le serveur.
Réglages : fsync=off, synchronous_commit=off, full_page_writes=off (docker-compose.yml)

## Réserve
Les deux campagnes n'ont pas tourné au même moment. L'état du cache du serveur
(shared_buffers, pages chaudes après TRUNCATE/COPY) n'est donc pas contrôlé entre
elles : seules les mesures dominées par le transport sont comparables, pas celles
dominées par le travail du serveur. Voir §3.3.
