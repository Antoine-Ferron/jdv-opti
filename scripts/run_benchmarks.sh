#!/usr/bin/env bash
# Pipeline de mesure complet, en une commande (axe 5 du barème) :
#   banc d'essai -> tests de conformité -> build -> go test -bench -> Hyperfine -> benchstat
#
# Usage : ./scripts/run_benchmarks.sh            (ou : make bench)
# Paramètres surchargeables par variables d'environnement :
#   IMPLS="naive flat"  SIZE=1024  TURNS=500  WARMUP=3  RUNS=15  COUNT=10  BENCH=.
#   BANC=m1-air   nom du banc d'essai (défaut : <système>-<architecture>)
#   FORCE=1       mesurer malgré un arbre de travail modifié (à éviter)
#
# Chaque exécution produit un dossier results/<commit>/<banc>/ à versionner : le
# même commit mesuré sur deux machines donne deux dossiers frères, directement
# comparables. Ce sont les pièces à conviction du rapport.
set -euo pipefail
cd "$(dirname "$0")/.."

SIZE=${SIZE:-1024}
TURNS=${TURNS:-500}
WARMUP=${WARMUP:-3}
RUNS=${RUNS:-15}
COUNT=${COUNT:-10}
BENCH=${BENCH:-.}
BANC=${BANC:-$(uname -s)-$(uname -m)}
BANC=$(echo "$BANC" | tr '[:upper:] ' '[:lower:]-')

need() { command -v "$1" >/dev/null || { echo "ERREUR : '$1' introuvable. $2" >&2; exit 1; }; }
need go "Installez Go : https://go.dev/dl/"
need hyperfine "Installez-le : 'sudo apt install hyperfine' ou 'cargo install hyperfine'."

# Une campagne lancée sur un arbre modifié n'est pas reproductible : le dossier
# results/ porterait un commit qui ne correspond pas au code mesuré.
if ! git diff --quiet HEAD 2>/dev/null && [ "${FORCE:-0}" != 1 ]; then
	echo "ERREUR : l'arbre de travail est modifié, la campagne ne serait pas reproductible." >&2
	echo "         Commitez d'abord, ou forcez avec FORCE=1 (le dossier sera marqué comme tel)." >&2
	git status --short >&2
	exit 1
fi

commit=$(git rev-parse --short HEAD 2>/dev/null || echo nogit)
out="results/${commit}/${BANC}"
if [ -d "$out" ]; then
	echo ">> Campagne existante pour ce commit et ce banc : elle sera remplacée."
fi
mkdir -p "$out"
echo ">> Banc : $BANC — résultats dans $out"

echo ">> [1/6] Description du banc d'essai"
./scripts/env.sh >"$out/env.md"

echo ">> [2/6] Tests de conformité (une version incorrecte n'est pas mesurée)"
go test ./... >"$out/tests.txt"

echo ">> [3/6] Compilation"
go build -o bin/wildfire ./cmd/wildfire
IMPLS=${IMPLS:-$(./bin/wildfire -list | tr '\n' ' ')}
echo "   Implémentations : $IMPLS"

echo ">> [4/6] Micro-benchmarks Go (count=$COUNT, -benchmem)"
go test ./internal/bench -run '^$' -bench "$BENCH" -benchmem -count "$COUNT" -timeout 0 | tee "$out/bench.txt"

echo ">> [5/6] Hyperfine (warmup=$WARMUP, runs=$RUNS, carte ${SIZE}x${SIZE}, $TURNS tours)"
cmds=()
for impl in $IMPLS; do
	cmds+=(-n "$impl" "./bin/wildfire -impl $impl -size $SIZE -turns $TURNS -fires 64 -quiet")
done
# -N : pas de shell intermédiaire (supprime le bruit du lancement de shell).
hyperfine -N --warmup "$WARMUP" --runs "$RUNS" \
	--export-json "$out/hyperfine.json" \
	--export-markdown "$out/hyperfine.md" \
	"${cmds[@]}"

# Statistiques complètes exigées par le barème (Hyperfine n'affiche pas la variance).
if command -v python3 >/dev/null; then
	python3 - "$out/hyperfine.json" >"$out/hyperfine-stats.md" <<'PY'
import json, statistics, sys
data = json.load(open(sys.argv[1]))["results"]
base = data[0]["mean"]
print("| Implémentation | Moyenne (s) | Médiane (s) | Écart-type (s) | Variance (s²) | CV | Min (s) | Max (s) | Accélération |")
print("|---|---|---|---|---|---|---|---|---|")
for r in data:
    t = r["times"]
    sd = statistics.stdev(t) if len(t) > 1 else 0.0
    print(f"| {r['command']} | {r['mean']:.4f} | {statistics.median(t):.4f} | {sd:.4f} | {sd*sd:.3e} "
          f"| {sd/r['mean']*100:.1f} % | {min(t):.4f} | {max(t):.4f} | x{base/r['mean']:.1f} |")
PY
	cat "$out/hyperfine-stats.md"
fi

echo ">> [6/6] benchstat"
if command -v benchstat >/dev/null; then
	benchstat -col /impl "$out/bench.txt" | tee "$out/benchstat.txt"
else
	echo "   benchstat absent : go install golang.org/x/perf/cmd/benchstat@latest" | tee "$out/benchstat.txt"
fi

echo ">> Terminé : $out"
