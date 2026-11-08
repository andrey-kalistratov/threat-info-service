OPENAPI_VALIDATOR_VERSION := 0.8.2
OGEN_VERSION := v1.24.0
MOCKERY_VERSION := v3.8.1
GOLANGCI_LINT_VERSION := v2.11.2

# Минимальное покрытие: целое число от 0 до 100.
MIN_COVERAGE := 60
COVERAGE_FOLDER := coverage
COVERAGE_EXCLUDE := /(generated|mocks)/

COMPOSE := docker compose -p threats-service --env-file docker/.env -f docker/compose.yaml

.PHONY: lint-openapi generate-http generate-mocks lint test test-cover db-up migrate up down

# Проверка спецификации OpenAPI.
lint-openapi:
	docker run --rm \
		--mount "type=bind,source=$(CURDIR)/api/openapi.yaml,target=/openapi.yaml,readonly" \
		pythonopenapi/openapi-spec-validator:$(OPENAPI_VALIDATOR_VERSION) /openapi.yaml

# Воспроизводимая генерация HTTP-контракта для разработчика и CI.
generate-http:
	go run github.com/ogen-go/ogen/cmd/ogen@$(OGEN_VERSION) \
		--config api/ogen.yaml --target internal/rest/generated --package generated --clean api/openapi.yaml

# Генерация моков зависимостей HTTP-тестов для разработчика и CI.
generate-mocks:
	go run github.com/vektra/mockery/v3@$(MOCKERY_VERSION) --config .mockery.yaml

# Проверка Go-кода линтерами.
lint:
	docker run --rm \
		--mount "type=bind,source=$(CURDIR),target=/app,readonly" \
		--workdir /app \
		golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run ./...

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

# Запуск PostgreSQL с ожиданием готовности.
db-up:
	$(COMPOSE) up -d --wait db

# Новый одноразовый запуск миграций, включая повторное развёртывание.
migrate:
	$(COMPOSE) run --build --rm migrate

# Перезапустить стек и применить миграции перед запуском API.
up:
	$(COMPOSE) rm --stop --force api
	$(COMPOSE) up -d --wait db
	$(COMPOSE) run --build --rm migrate
	$(COMPOSE) up -d --build --no-deps api

# Остановить сервисы, сохранив данные PostgreSQL в томе.
down:
	$(COMPOSE) down
