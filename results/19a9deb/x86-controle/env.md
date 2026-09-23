# Banc d'essai — 2026-09-23T10:42:18+02:00
Commit : 19a9deb

## CPU
Architecture:                            x86_64
CPU(s):                                  24
Model name:                              Intel(R) Core(TM) Ultra 9 275HX
Thread(s) per core:                      1
Core(s) per socket:                      24
Socket(s):                               1
NUMA node(s):                            1
Extensions SIMD : avx avx2 sse4_2

## Hiérarchie de caches
NAME ONE-SIZE ALL-SIZE WAYS TYPE        LEVEL  SETS PHY-LINE COHERENCY-SIZE
L1d       48K     1.1M   12 Data            1    64        1             64
L1i       64K     1.5M   16 Instruction     1    64        1             64
L2         3M      72M   12 Unified         2  4096        1             64
L3        36M      36M   12 Unified         3 49152        1             64
Taille de ligne de cache (L1d) : 64 octets

## Mémoire
               total        used        free      shared  buff/cache   available
Mem:            31Gi       805Mi        30Gi        15Mi       378Mi        30Gi
Swap:          8.0Gi          0B       8.0Gi

## Système
PRETTY_NAME="Ubuntu 26.04 LTS"
Linux 6.6.114.1-microsoft-standard-WSL2 x86_64
ATTENTION : WSL détecté — le mentionner dans le rapport (virtualisation Hyper-V).

## Sources de bruit
Gouverneur CPU : non exposé (VM/WSL ?)
Charge moyenne : 0.02 0.21 0.23
Processus les plus gourmands :
%CPU COMMAND
 0.8 make
 0.0 wsl-pro-service
 0.0 systemd
 0.0 systemd-udevd
 0.0 polkitd

## Runtime
go version go1.27.1 linux/amd64
GOOS GOARCH GOAMD64 CGO_ENABLED : linux amd64 v1 0
GOMAXPROCS par défaut : nombre de CPU logiques (sauf variable GOMAXPROCS=non définie)
GOGC=défaut (100)
hyperfine 1.20.0
