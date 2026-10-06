.PHONY: up down run test keys flush generate mockery-install

up:
	docker compose -f deploy/compose.yaml up -d

down:
	docker compose -f deploy/compose.yaml down

run:
	go run ./cmd/app

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
