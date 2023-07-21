-include secrets

LDFLAGS="-X main.commit=$(shell git rev-parse --short HEAD) -X main.buildDate=$(shell date +"%Y-%m-%dT%H:%M:%S%z")"

.PHONY: build
build:
	CGO_ENABLED=1 go build -ldflags=$(LDFLAGS) -o ./bin/api .

.PHONY: run
run: build
	AWS_REGION=$(AWS_REGION) \
	AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) \
	AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) \
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

