.PHONY: build
build:
	go build -o ./bin/api ./cmd/api

.PHONY: run
run: build
	./bin/api -config local/api-config.yml

.PHONY: test
test:
	go test -v ./...

.PHONY: run-docker
run-docker:
	@docker compose up

.PHONY: clean-docker
clean-docker:
	docker compose kill || true
	docker compose rm -f || true

.PHONY: build-docker
build-docker:
	docker compose build

