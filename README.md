# Threat Information Service

## О проекте

REST-сервис для хранения и обмена сведениями об угрозах по MD5.

[Репозиторий](https://github.com/andrey-kalistratov/threat-info-service) · [Закрытые pull request](https://github.com/andrey-kalistratov/threat-info-service/pulls?q=is%3Apr+is%3Aclosed)

### API

Контракт описан в [OpenAPI](api/openapi.yaml): спецификация последовательно развивалась и фиксировалась по версиям. 

Сервис предоставляет операции:

- `POST /info` — создать запись об угрозе;
- `GET /info/{md5}` — получить запись;
- `PUT /info/{md5}` — заменить сведения.

### Ключевые решения

- HTTP-типы и маршрутизация генерируются из OpenAPI с помощью ogen.
- MD5 принимается в любом допустимом регистре, но хранится и возвращается в нижнем.
- Данные хранятся в PostgreSQL, миграции запускаются Goose.

Подробности — в [решениях проекта](docs/decisions.md).

## Запуск

Нужны Docker Compose и Go версии, указанной в `go.mod`.

1. Создайте локальный файл настроек:

   ```sh
   cp docker/.env.example docker/.env
   ```

2. Укажите свой пароль PostgreSQL в `docker/.env` (`DB_POSTGRES_PASSWORD`).
3. Запустите сервис из корня проекта:

   ```sh
   make up
   ```

API будет доступен на `http://localhost:8080`. Для остановки выполните `make down`. Данные PostgreSQL сохраняются в Docker volume.

## Проверки и CI

Локальные проверки:

```sh
make lint-openapi  # проверка контракта OpenAPI
make lint          # линтеры Go
make test-cover    # тесты и минимальное покрытие
```

### CI

CI настроен на GitHub Actions и запускается при pull request в `main` и push в `main`.

Проверки CI:

- OpenAPI и актуальность сгенерированных HTTP-кода и моков;
- сборка Go и линтеры;
- тесты с проверкой покрытия.
