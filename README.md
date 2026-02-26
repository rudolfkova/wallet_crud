# Wallet Service

REST API сервис для управления балансом кошельков.

## Stack

- **Go 1.25** - стандартная библиотека `net/http` без фреймворков
- **PostgreSQL 16** - основное хранилище
- **pgx/v5** - драйвер PostgreSQL с пулом соединений
- **Docker + docker-compose** - запуск сервиса

## API

### POST /api/v1/wallet
Пополнение или списание средств.

```json
{
  "valletId": "11111111-1111-1111-1111-111111111111",
  "operationType": "DEPOSIT",
  "amount": 1000
}
```

`operationType`: `DEPOSIT` или `WITHDRAW`

**Responses:**
- `200 OK` - операция выполнена
- `400 Bad Request` - невалидные данные
- `404 Not Found` - кошелёк не найден
- `409 Conflict` - недостаточно средств

### GET /api/v1/wallets/{uuid}
Получить баланс кошелька.

```json
{
  "walletId": "11111111-1111-1111-1111-111111111111",
  "balance": 1000
}
```

## Запуск

```bash
docker-compose up --build
```

Сервис поднимается на `http://localhost:8080`. PostgreSQL стартует первым, применяет миграции из папки `init/`, затем стартует приложение.

**Тестовый кошелёк** создаётся автоматически при первом запуске:
```
ID:      11111111-1111-1111-1111-111111111111
Balance: 100000
```

## Конфигурация

Переменные окружения читаются из `config.env`. Пример в `config.env.example`:

```env
BIND_ADDR=:8080
DATABASE_URL=host=localhost port=5432 user=postgres password=postgres dbname=wallet sslmode=disable
LOG_LEVEL=DEBUG
TEST_DATABASE_URL=host=localhost port=5432 user=postgres password=postgres dbname=wallet_test sslmode=disable
```

## Архитектура

Чистая архитектура с разделением на слои:

```
handler - usecase - repository - PostgreSQL
```

- **handler** - HTTP, валидация входящих данных, маппинг ошибок в HTTP-коды
- **usecase** - бизнес-логика, управление транзакциями через `TxManager`
- **repository** - SQL-запросы, абстракция над pgx

Зависимости направлены внутрь: usecase не знает о pgx, repository не знает о HTTP.

## Тесты

```bash
# юнит-тесты (без БД)
make test

# интеграционные тесты (нужна TEST_DATABASE_URL в config.env)
make test
```

## Нагрузочное тестирование (k6)

```bash
make bench
# или напрямую
k6 run bench.js
```

Сценарий гоняет **100 виртуальных пользователей в течение 10 секунд** по одному кошельку (`11111111-1111-1111-1111-111111111111`):
- 40% запросов — `DEPOSIT` на 10
- 40% запросов — `WITHDRAW` на 1
- 20% запросов — `GET balance`

Пороги (thresholds):
- `p(99) < 500ms` — 99% запросов должны укладываться в 500ms
- `rate == 0` — ноль ошибок

> Перед запуском убедись что `LOG_LEVEL=ERROR` в `config.env` — логирование на DEBUG заметно снижает RPS.
```

## Debug UI

В корне проекта лежит `wallet_debug.html` - открыть в браузере при запущенном сервисе. Позволяет делать запросы к API и смотреть ответы.
