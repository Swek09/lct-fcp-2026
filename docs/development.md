# Запуск и разработка

## Полный стенд

Установите Docker с Compose и выполните из корня:

```sh
docker compose up --build -d
docker compose ps
```

Адрес приложения — http://localhost:8080. API — /api; OpenAPI —
/api/openapi.json; интерфейс документации — /api/docs. Письма регистрации
и восстановления доступа доступны в http://localhost:8025.

Для настройки скопируйте .env.example в .env и измените значения.
API запускает встроенные SQL-миграции до обслуживания запросов.
DEMO_SEED=true добавляет синтетические аккаунты и профили; повторный запуск
сохраняет изменения в существующих записях.

```sh
docker compose logs api --tail 100
docker compose logs web --tail 100
docker compose down
```

Остановка сохраняет volumes. Не удаляйте volumes, если нужны результаты,
аккаунты или статусы приглашений.

## Локальная разработка

Используются Go 1.26.8 и Node.js 24. Параметры Go закреплены в go.mod;
зависимости npm — в lock-файле. Docker собирает API и статический фронтенд
независимо от установленных на хосте языковых сред.

```sh
docker compose -f compose.yaml -f compose.dev.yaml up -d postgres redis mailpit
```

Ресурсы: PostgreSQL на localhost:15432, Redis на localhost:16379,
SMTP на localhost:1025. Compose.dev также публикует API на localhost:8000.

Для API на хосте в PowerShell:

```powershell
$env:DATABASE_URL='postgres://fsp:fsp_local_change_me@localhost:15432/fsp?sslmode=disable'
$env:REDIS_URL='redis://localhost:16379/0'
$env:SMTP_HOST='localhost'
$env:SMTP_PORT='1025'
$env:APP_URL='http://localhost:5173'
$env:DEMO_SEED='true'
cd backend
go run ./cmd/api
```

Если API уже запущен Compose с портом 8000, остановите только этот сервис
перед запуском API на хосте: docker compose stop api.

В другом терминале:

```sh
cd frontend
npm ci
npm run dev
```

Vite проксирует /api на localhost:8000. APP_URL задаёт origin фронтенда:
это проверяется при изменяющих запросах. В APP_ENV=development локальные
адреса localhost, 127.0.0.1 и [::1] взаимозаменяемы при совпадении протокола
и порта с APP_URL. Поэтому полный стенд также работает на
http://127.0.0.1:8080. В остальных режимах Origin должен точно совпадать
с APP_URL.

## Проверки с реальными БД

Запустите PostgreSQL и Redis с compose.dev.yaml. В PowerShell из backend:

```powershell
$env:TEST_DATABASE_URL='postgres://fsp:fsp_local_change_me@localhost:15432/fsp?sslmode=disable'
$env:TEST_REDIS_URL='redis://localhost:16379/1'
go test ./... -count=1
go vet ./...
```

Интеграционный тест создаёт случайную отдельную схему PostgreSQL и удаляет её
после проверки. Он не очищает базу приложения и не делает FLUSHALL в Redis.
Сессии интеграционного теста используют отдельный Redis DB 1.

Для Linux/macOS можно использовать переменные окружения перед командой:

```sh
TEST_DATABASE_URL='postgres://fsp:fsp_local_change_me@localhost:15432/fsp?sslmode=disable' TEST_REDIS_URL='redis://localhost:16379/1' go test ./... -count=1
```

Эксперимент с переносом готовых ответов:

```sh
go run ./cmd/evaluate
go run ./cmd/evaluate-matching
```

Вторая команда читает docs/matching-dataset.json и автономно проверяет
парсер и ранжирование. Проверка этих описаний через работающий API:

```powershell
./scripts/benchmark-matching.ps1
```

Запускайте скрипт из корня. Он использует синтетический аккаунт работодателя,
создаёт отдельную сессию и завершает её после чтения выдачи.

Экспорт OpenAPI не требует запущенной БД:

```powershell
cd backend
$env:EXPORT_OPENAPI='../docs/openapi.json'
go run ./cmd/api
Remove-Item Env:EXPORT_OPENAPI
```

Если путь заканчивается на .json, экспортируется JSON; иначе YAML.

## Изменение схемы

Миграции хранятся в backend/migrations. Добавляйте новую миграцию с директивами
goose; существующие применённые миграции не редактируйте. SQL-запросы sqlc —
backend/queries; сгенерированный Go-код сохраняется в репозитории.

```sh
cd backend
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate
```

## Параметры размещения

Текущий Compose публикует приложение и почту только на loopback. Для удалённого
стенда нужен домен и HTTPS на внешнем reverse proxy. APP_URL задаётся точным
HTTPS-origin, APP_ENV=production, DEMO_SEED=false. Production-режим требует
HTTPS и STARTTLS у SMTP. Для реальных аккаунтов нужна отдельная база без
демонстрационных пользователей и паролей.

POSTGRES_PASSWORD задаётся через окружение. SMTP_HOST, SMTP_PORT, SMTP_FROM,
SMTP_USER и SMTP_PASSWORD настраивают отправку почты. Реальные значения
не добавляются в Git. Контейнеры PostgreSQL и Redis в основном Compose
не публикуют порты наружу.

## Материалы сдачи

Готовые файлы находятся в docs/delivery. PDF собирается скриптом
scripts/build-documentation.py с Python и reportlab. На Windows он
использует Arial; в других средах задайте FSP_PDF_FONT_REGULAR
и FSP_PDF_FONT_BOLD путями к шрифтам с кириллицей.

```sh
python scripts/build-documentation.py
```

Новый PDF создаётся в output/pdf. Скопируйте проверенный результат
в docs/delivery. Пересборка презентации использует JavaScript-скрипт
scripts/build-presentation.mjs и пакет @oai/artifact-tool из среды Codex.
Готовый PPTX можно редактировать обычным редактором презентаций.
Пакет Artifact Tool не является зависимостью приложения.

Для запуска скрипта презентации создайте отдельную рабочую папку,
ссылку node_modules на пакеты этой среды и скопируйте туда скрипт.
Задайте WORKSPACE_DIR, SKILL_DIR, RUNTIME_PYTHON и RUNTIME_NODE_MODULES
абсолютными путями. Для новой ревизии FINAL_PPTX должен указывать
на ещё не существующий файл. Внутренний отчёт проверки остаётся
в output/presentation-build.
