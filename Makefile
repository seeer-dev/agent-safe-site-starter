.PHONY: dev migrate seed render preview verify archcheck scopecheck

dev:
	go run ./server/tools/dev

migrate:
	go run ./server/tools/migrate

seed:
	go run ./server/tools/seed

render:
	go run ./server/tools/render

preview:
	go run ./server/tools/preview

archcheck:
	go run ./server/tools/archcheck

scopecheck:
	go run ./server/tools/scopecheck

verify:
	go run ./server/tools/verify
