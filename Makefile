# Points d'entrée du projet. Tout est reproductible en une commande : make bench
IMPL  ?= naive
SIZE  ?= 1024
TURNS ?= 500
FIRES ?= 64
ADDR  ?= 127.0.0.1:8081
BANC  ?= $(shell uname -s)-$(shell uname -m)
COMMIT := $(shell git rev-parse --short HEAD)
PROF  := results/$(COMMIT)/$(BANC)/profiles

.PHONY: help build test env bench quick demo web profile flame layout escape tools clean

help: ## Affiche cette aide
	@grep -E '^[a-z]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  make %-9s %s\n", $$1, $$2}'

build: ## Compile bin/wildfire
	go build -o bin/wildfire ./cmd/wildfire

test: ## Conformité de toutes les implémentations (internal/firetest)
	go test ./...

env: ## Décrit le banc d'essai (axe 1)
	@./scripts/env.sh

bench: ## Campagne complète sur ce banc (BANC=, SIZE=, TURNS=) -> results/<commit>/<banc>/
	BANC=$(BANC) SIZE=$(SIZE) TURNS=$(TURNS) ./scripts/run_benchmarks.sh

quick: build ## Exécution rapide d'une implémentation (IMPL=, SIZE=, TURNS=, FIRES=)
	./bin/wildfire -impl $(IMPL) -size $(SIZE) -turns $(TURNS) -fires $(FIRES)

demo: build ## Incendie animé en console, à taille d'écran
	./bin/wildfire -w 100 -h 35 -turns 400 -fires 2 -render

profile: build ## Profils CPU + allocations de IMPL -> results/<commit>/<banc>/profiles/
	@mkdir -p $(PROF)
	./bin/wildfire -impl $(IMPL) -size $(SIZE) -turns $(TURNS) -fires $(FIRES) -quiet \
		-cpuprofile $(PROF)/$(IMPL)-cpu.prof -memprofile $(PROF)/$(IMPL)-mem.prof
	go tool pprof -top -nodecount=15 bin/wildfire $(PROF)/$(IMPL)-cpu.prof | tee $(PROF)/$(IMPL)-cpu-top.txt
	go tool pprof -sample_index=alloc_space -top -nodecount=15 bin/wildfire $(PROF)/$(IMPL)-mem.prof | tee $(PROF)/$(IMPL)-mem-top.txt
	go tool pprof -list 'Step|Fingerprint|Burning' bin/wildfire $(PROF)/$(IMPL)-cpu.prof > $(PROF)/$(IMPL)-cpu-list.txt

flame: ## pprof dans le navigateur (View > Flame Graph) -> rapport/figures/<banc>/
	go tool pprof -http=localhost:8080 bin/wildfire $(PROF)/$(IMPL)-cpu.prof

layout: ## Taille des structs (axe padding)
	go test ./internal/... -run Layout -v | grep -i sizeof

escape: ## Analyse d'échappement : quelles variables partent sur le tas (axe zero-allocation)
	go build -gcflags=-m ./internal/$(IMPL) 2>&1 | grep -E 'escapes|moved to heap'

tools: ## Installe benchstat
	go install golang.org/x/perf/cmd/benchstat@latest

clean: ## Supprime binaires et snapshots (les résultats sont conservés)
	rm -rf bin snapshots

web: ## Lance la carte interactive (ADDR=0.0.0.0:8081 pour y accéder depuis Windows)
	go run ./cmd/web -addr $(ADDR)
