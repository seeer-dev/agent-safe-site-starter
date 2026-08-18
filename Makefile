.PHONY: dev migrate seed theme site render preview verify verify-contracts archcheck scopecheck

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

# verify-contracts runs the two frontend contract checks that guard the admin
# resource config and the public theme against contracts/openapi.yaml. It is
# separate from `verify` on purpose: `verify` stays Go-only, so a contributor
# without Node is never blocked by the repository verifier.
#
# Both scripts import only Node standard-library modules, so there is no
# install step. check-resource-contracts.mjs uses import.meta.dirname, which
# needs Node 20.11 or newer; an older Node fails with a confusing undefined
# path instead, so the version is checked first and reported plainly.
verify-contracts:
	@node -e "const v=process.versions.node.split('.').map(Number); if(v[0]<20||(v[0]===20&&v[1]<11)){console.error('verify-contracts: Node '+process.versions.node+' is too old. Node 20.11 or newer is required because check-resource-contracts.mjs uses import.meta.dirname. Install a newer Node and retry.');process.exit(1)}"
	node admin/scripts/check-resource-contracts.mjs
	node site/themes/minimal-cart/scripts/check-openapi-contracts.mjs
