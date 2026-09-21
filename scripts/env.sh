#!/usr/bin/env bash
# Décrit le banc d'essai (axe 1 du barème) : CPU, cœurs/threads, caches L1/L2/L3,
# RAM, OS, runtime Go, et état des sources de bruit. Sortie : stdout (à rediriger).
set -uo pipefail

section() { printf '\n## %s\n' "$1"; }

echo "# Banc d'essai — $(date -Iseconds)"
if git rev-parse --git-dir >/dev/null 2>&1; then
	echo "Commit : $(git rev-parse --short HEAD 2>/dev/null || echo 'aucun')$(git diff --quiet HEAD 2>/dev/null || echo ' (ATTENTION : modifications non commitées)')"
else
	echo "Commit : hors dépôt git"
fi

case "$(uname -s)" in
Linux)
	section "CPU"
	lscpu | grep -E '^(Architecture|Model name|Socket|Core|Thread|CPU\(s\)|CPU max MHz|CPU min MHz|NUMA node\(s\))'
	echo "Extensions SIMD : $(grep -o -w -E 'sse4_2|avx|avx2|avx512f' /proc/cpuinfo | sort -u | paste -sd' ' -)"
	section "Hiérarchie de caches"
	if lscpu -C >/dev/null 2>&1; then
		lscpu -C
	else
		lscpu | grep -i cache
	fi
	echo "Taille de ligne de cache (L1d) : $(getconf LEVEL1_DCACHE_LINESIZE 2>/dev/null || echo '?') octets"
	section "Mémoire"
	free -h
	section "Système"
	grep PRETTY_NAME /etc/os-release 2>/dev/null
	uname -srm
	grep -qi microsoft /proc/version 2>/dev/null && echo "ATTENTION : WSL détecté — le mentionner dans le rapport (virtualisation Hyper-V)."
	section "Sources de bruit"
	echo "Gouverneur CPU : $(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null || echo 'non exposé (VM/WSL ?)')"
	if [ -r /sys/devices/system/cpu/intel_pstate/no_turbo ]; then
		echo "Turbo désactivé (intel_pstate/no_turbo) : $(cat /sys/devices/system/cpu/intel_pstate/no_turbo)"
	elif [ -r /sys/devices/system/cpu/cpufreq/boost ]; then
		echo "Boost actif (cpufreq/boost) : $(cat /sys/devices/system/cpu/cpufreq/boost)"
	fi
	echo "Charge moyenne : $(cut -d' ' -f1-3 /proc/loadavg)"
	echo "Processus les plus gourmands :"
	ps -eo pcpu,comm --sort=-pcpu | head -6
	;;
Darwin)
	section "CPU"
	sysctl -n machdep.cpu.brand_string
	echo "Cœurs physiques : $(sysctl -n hw.physicalcpu) / logiques : $(sysctl -n hw.logicalcpu)"
	section "Hiérarchie de caches"
	sysctl hw.l1icachesize hw.l1dcachesize hw.l2cachesize hw.l3cachesize hw.cachelinesize 2>/dev/null
	section "Mémoire"
	echo "RAM : $(($(sysctl -n hw.memsize) / 1024 / 1024 / 1024)) Gio"
	section "Système"
	sw_vers
	;;
*)
	echo "OS non géré ($(uname -s)) : utilisez WSL2 sous Windows."
	;;
esac

section "Runtime"
go version
go env GOOS GOARCH GOAMD64 CGO_ENABLED | paste -sd' ' - | sed 's/^/GOOS GOARCH GOAMD64 CGO_ENABLED : /'
echo "GOMAXPROCS par défaut : nombre de CPU logiques (sauf variable GOMAXPROCS=${GOMAXPROCS:-non définie})"
echo "GOGC=${GOGC:-défaut (100)}"
command -v hyperfine >/dev/null && hyperfine --version
