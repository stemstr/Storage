.PHONY: run
run:
	@docker compose up

.PHONY: clean
clean:
	docker compose kill || true
	docker compose rm -f || true

.PHONY: build
build:
	docker compose build

.PHONY: test
test:
	go test -v ./...

.PHONY: build-api
build-api:
	go build -o ./bin/api ./cmd/api

# run runs the server with a config file
.PHONY: run-api
run-api: build
	./bin/api -config config/api-config.yml
