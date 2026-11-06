OPENAPI_VALIDATOR_IMAGE := pythonopenapi/openapi-spec-validator:0.8.2

.PHONY: lint-openapi

lint-openapi:
	docker run --rm \
		--mount "type=bind,source=$(CURDIR)/api/openapi.yaml,target=/openapi.yaml,readonly" \
		$(OPENAPI_VALIDATOR_IMAGE) /openapi.yaml
