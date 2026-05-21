.PHONY: run build docker-up docker-down test tidy migrate

# Запуск локально (нужны PostgreSQL и Redis)
run:
	go run ./cmd/server

# Сборка бинарника
build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

# Загрузка зависимостей
tidy:
	go mod tidy

# Запуск через Docker Compose
docker-up:
	docker-compose up --build

# Запуск в фоне
docker-up-d:
	docker-compose up --build -d

# Остановка
docker-down:
	docker-compose down

# Остановка + удаление данных
docker-clean:
	docker-compose down -v

# Применить миграцию вручную
migrate:
	docker exec -it $$(docker-compose ps -q db) \
	  psql -U messenger_user -d messenger_db -f /docker-entrypoint-initdb.d/init.sql

# Логи приложения
logs:
	docker-compose logs -f app

# Тесты
test:
	go test -race ./...

# Race detector
test-race:
	go test -race -count=1 ./...

# Проверить что всё компилируется
check:
	go build ./...
