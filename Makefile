.PHONY: run
run:
	@docker compose up

.PHONY: clean
clean:
	docker compose kill || true
	docker compose rm -f || true
