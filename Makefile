.PHONY: up up-dev down down-dev run run-dev test keys flush generate mockery-install

DEV_COMPOSE = docker compose --env-file deploy/.env.dev -f deploy/devcompose.yaml

up:
	docker compose -f deploy/compose.yaml up -d

up-dev:
	$(DEV_COMPOSE) up -d

down:
	docker compose -f deploy/compose.yaml down

down-dev:
	$(DEV_COMPOSE) down

run:
	go run ./cmd/app

run-dev:
	ENV_FILE=deploy/.env.dev go run ./cmd/app

test:
	go test ./... -race

keys:
	@docker exec booking-redis redis-cli --scan --pattern '*' | sort | while read -r k; do \
		printf '%-32s %-8s ttl=%s\n' "$$k" "$$(docker exec booking-redis redis-cli type $$k)" "$$(docker exec booking-redis redis-cli ttl $$k)"; \
	done

flush:
	docker exec booking-redis redis-cli flushall

mockery-install:
	go install github.com/vektra/mockery/v3@v3.2.5

generate:
	go generate ./...
