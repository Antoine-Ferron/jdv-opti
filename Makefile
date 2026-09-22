# Points d'entrée du projet. Tout est reproductible en une commande : make bench
IMPL  ?= naive
SIZE  ?= 1024
GENS  ?= 50
PROF  := results/profiles

.PHONY: help build test env bench quick profile flame layout escape tools clean

help: ## Affiche cette aide
	@grep -E '^[a-z]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  make %-9s %s\n", $$1, $$2}'

build: ## Compile bin/gol
	go build -o bin/gol ./cmd/gol

test: ## Tests de conformité de toutes les implémentations
	go test ./...

env: ## Décrit le banc d'essai (axe 1)
	@./scripts/env.sh

bench: ## Pipeline complet : env + tests + go bench + hyperfine + benchstat (axe 5)
	./scripts/run_benchmarks.sh

quick: build ## Exécution rapide d'une implémentation (IMPL=, SIZE=, GENS=)
	./bin/gol -impl $(IMPL) -size $(SIZE) -gens $(GENS)

profile: build ## Profils CPU + allocations de IMPL (axe 2)
	@mkdir -p $(PROF)
	./bin/gol -impl $(IMPL) -size $(SIZE) -gens $(GENS) \
		-cpuprofile $(PROF)/$(IMPL)-cpu.prof -memprofile $(PROF)/$(IMPL)-mem.prof
	go tool pprof -top -nodecount=15 bin/gol $(PROF)/$(IMPL)-cpu.prof | tee $(PROF)/$(IMPL)-cpu-top.txt
	go tool pprof -sample_index=alloc_space -top -nodecount=15 bin/gol $(PROF)/$(IMPL)-mem.prof | tee $(PROF)/$(IMPL)-mem-top.txt
	go tool pprof -list 'neighbors|Fingerprint|Step' bin/gol $(PROF)/$(IMPL)-cpu.prof > $(PROF)/$(IMPL)-cpu-list.txt

flame: ## Ouvre pprof dans le navigateur (menu View > Flame Graph) pour les captures
	go tool pprof -http=localhost:8080 bin/gol $(PROF)/$(IMPL)-cpu.prof

layout: ## Taille des structs (axe padding)
	go test ./internal/... -run Layout -v | grep -i sizeof

escape: ## Analyse d'échappement : quelles variables partent sur le tas (axe zero-allocation)
	go build -gcflags=-m ./internal/$(IMPL) 2>&1 | grep -E 'escapes|moved to heap'

tools: ## Installe benchstat
	go install golang.org/x/perf/cmd/benchstat@latest

clean: ## Supprime binaires et snapshots (les résultats sont conservés)
	rm -rf bin snapshots

.PHONY: web
web: ## Lance la page interactive sur http://127.0.0.1:8081
	go run ./cmd/web
