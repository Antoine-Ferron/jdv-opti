# Ancien relevé Intel/WSL — à vérifier séparément

Texte conservé depuis le rapport avant documentation de la campagne Mac.
Les interprétations ci-dessous ne sont pas validées par la campagne Mac et
ne doivent pas servir à justifier ses résultats.


Source : `results/env-2026-09-22.md` (généré par `make env`, complété des relevés hôte Windows).

| Élément                   | Valeur                                                                                             |
|---------------------------|----------------------------------------------------------------------------------------------------|
| CPU (modèle)              | Intel Core Ultra 9 275HX (Arrow Lake-HX), base 2,7 GHz                                             |
| Cœurs physiques / threads | 24 / 24 — **pas de SMT** : 1 thread par cœur                                                       |
| L1d / L1i (par cœur)      | 48 Kio / 64 Kio                                                                                    |
| L2                        | 3 Mio privés par cœur — **40 Mio au total** côté hôte                                              |
| L3                        | 36 Mio partagés par les 24 cœurs                                                                   |
| Ligne de cache            | 64 o                                                                                               |
| RAM                       | 64 Gio DDR5-6400 (2×32 Gio SK Hynix) — **31 Gio visibles depuis WSL2**                             |
| OS / noyau                | Windows 11 Famille 10.0.26200 → **WSL2** Ubuntu 26.04 LTS, noyau 6.6.114.1-microsoft-standard-WSL2 |
| Runtime                   | go1.27.1 linux/amd64, `GOAMD64=v1`, `CGO_ENABLED=0`, `GOGC=100`, `GOMAXPROCS=24`                   |
| SIMD disponibles          | sse4_2, avx, avx2 (pas d'AVX-512 sur Arrow Lake)                                                   |
| Alimentation / gouverneur | secteur, profil Acer `ed2c8e98-…` ; gouverneur **non exposé** sous WSL2                            |

**Trois réserves à porter au crédit de la métrologie, pas à sa charge :**

1. **Virtualisation Hyper-V.** Les mesures tournent dans WSL2, pas sur le métal. Le
   coût est constant entre toutes les versions comparées — les *rapports* de gain
   restent valides, les valeurs absolues de débit sont minorées.
2. **Topologie hybride masquée.** L'Arrow Lake-HX mêle P-cores et E-cores, mais WSL2
   présente 24 cœurs homogènes (`lscpu` annonce même 72 Mio de L2, contre 40 Mio
   relevés côté hôte). Conséquence directe pour le §3.2 : un worker pool dimensionné
   à `GOMAXPROCS` répartit le travail sur des cœurs de puissances inégales, et la
   génération la plus lente impose son rythme à la barrière de synchronisation.
   Attendre une scalabilité sous-linéaire dès que le nombre de workers dépasse le
   nombre de P-cores.
3. **Fréquence non verrouillée.** Ni gouverneur ni état turbo ne sont pilotables
   depuis WSL2 : c'est le warmup Hyperfine et le coefficient de variation qui
   attestent de la stabilité, pas un réglage système.

