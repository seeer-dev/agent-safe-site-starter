.PHONY: dev migrate seed theme site render preview verify archcheck scopecheck

dev:
	go run ./server/tools/dev

migrate:
	go run ./server/tools/migrate

seed:
	go run ./server/tools/seed

# theme builds the Vue islands bundle the renderer requires. The bundle is
# git-ignored, so a fresh clone has none and the renderer fails closed until
# this runs.
theme:
	npm --prefix site/themes/minimal-cart run build

# site is the one command that takes a fresh clone to a rendered site. Use it
# as the deployment build command. The two steps remain separately callable.
site: theme render

# render renders only. It fails closed when the theme bundle is absent, which
# is correct; run `make site` or `make theme` first.
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
