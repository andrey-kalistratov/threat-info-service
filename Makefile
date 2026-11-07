OPENAPI_VALIDATOR_IMAGE := pythonopenapi/openapi-spec-validator:0.8.2
GOLANGCI_LINT_IMAGE := golangci/golangci-lint:v2.11.2

# Минимальное покрытие: целое число от 0 до 100.
MIN_COVERAGE := 70
COVERAGE_FOLDER := coverage
COVERAGE_EXCLUDE :=

.PHONY: lint-openapi lint test test-cover

# Проверка спецификации OpenAPI.
lint-openapi:
	docker run --rm \
		--mount "type=bind,source=$(CURDIR)/api/openapi.yaml,target=/openapi.yaml,readonly" \
		$(OPENAPI_VALIDATOR_IMAGE) /openapi.yaml

# Проверка Go-кода линтерами.
lint:
	docker run --rm \
		--mount "type=bind,source=$(CURDIR),target=/app,readonly" \
		--workdir /app \
		$(GOLANGCI_LINT_IMAGE) golangci-lint run ./...

# Все тесты запускаются заново; в покрытие входит только internal.
test:
	@mkdir -p $(COVERAGE_FOLDER)
	@rm -f $(COVERAGE_FOLDER)/coverage.out $(COVERAGE_FOLDER)/index.html $(COVERAGE_FOLDER)/coverage-func.txt
	@go test -count=1 -coverpkg=./internal/... -coverprofile=$(COVERAGE_FOLDER)/coverage.out ./...

# Создание HTML-отчёта и проверка минимального покрытия после тестов.
test-cover: test
	@if [ -n '$(COVERAGE_EXCLUDE)' ]; then \
		grep -Ev '$(COVERAGE_EXCLUDE)' $(COVERAGE_FOLDER)/coverage.out > $(COVERAGE_FOLDER)/filtered.out && \
		mv $(COVERAGE_FOLDER)/filtered.out $(COVERAGE_FOLDER)/coverage.out; \
	fi
	@go tool cover -html=$(COVERAGE_FOLDER)/coverage.out -o $(COVERAGE_FOLDER)/index.html
	@go tool cover -func=$(COVERAGE_FOLDER)/coverage.out > $(COVERAGE_FOLDER)/coverage-func.txt
	@echo "HTML-отчёт: $(COVERAGE_FOLDER)/index.html"
	@cov=$$(grep '^total:' $(COVERAGE_FOLDER)/coverage-func.txt | awk '{print $$3}' | cut -d. -f1); \
	case "$$cov" in \
	    ''|*[!0-9]*) \
	        echo "❌ Не удалось определить покрытие: '$$cov'"; \
	        exit 1;; \
	esac; \
	if [ "$$cov" -ge "$(MIN_COVERAGE)" ]; then \
	    echo "✅ Покрытие кода ($$cov%) соответствует норме (>= $(MIN_COVERAGE)%)."; \
	else \
	    echo "❌ Покрытие кода ($$cov%) ниже допустимого порога ($(MIN_COVERAGE)%)!"; \
	    exit 1; \
	fi
