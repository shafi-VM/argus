.PHONY: demo demo-signoz up up-signoz down logs test build fmt help
COMPOSE_SIGNOZ := docker compose -f compose.yaml -f compose.signoz.yaml

help: ## show this help
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | sed -E 's/:.*##/ —/' | sort

demo: up ## zero-dep: bring up Argus and drive the PREVENT money moment
	@printf "waiting for argusd"; until curl -sf localhost:8088/healthz >/dev/null 2>&1; do printf .; sleep 1; done; echo " ready"
	python3 demo/drive.py

demo-signoz: up-signoz ## full arc: PREVENT + LEARN through SigNoz (needs .env + SigNoz running)
	@printf "waiting for argusd"; until curl -sf localhost:8088/healthz >/dev/null 2>&1; do printf .; sleep 1; done; echo " ready"
	python3 demo/drive.py

up: ## bring up the zero-dep stack (gateway + replay engine)
	docker compose up -d --build

up-signoz: ## bring up Argus wired to SigNoz (joins signoz-network, provisions dashboards)
	$(COMPOSE_SIGNOZ) up -d --build

down: ## stop and remove the app containers (both tiers)
	$(COMPOSE_SIGNOZ) down

logs: ## tail argusd logs
	docker compose logs -f argusd

test: ## run the Go test suite (race detector)
	go test -race ./...

build: ## build the argusd binary locally
	go build ./cmd/argusd

fmt: ## format the Go code
	gofmt -w .
