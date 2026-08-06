.PHONY: openapi-generate openapi-lint openapi-check

openapi-generate:
	go generate ./internal/openapi

openapi-lint:
	npx @redocly/cli@latest lint cmd/api/openapi.yaml

openapi-check:
	$(MAKE) openapi-generate
	git diff --exit-code -- internal/openapi