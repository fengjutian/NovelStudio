.PHONY: api test web

api:
	go run ./cmd/api

test:
	go test ./...

web:
	cd web && npm run dev
