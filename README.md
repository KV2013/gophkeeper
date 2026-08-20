# GophKeeper

Менеджер паролей: CLI-клиент + сервер. Данные шифруются на клиенте
(end-to-end), сервер хранит только шифротекст (бинарные файлы — в S3/MinIO).

## Требования

- Go 1.26+
- Docker + Docker Compose (для PostgreSQL и MinIO)

## 1. Сборка исполняемых файлов

```bash
./build_server_and_client.sh
```

Создаёт в каталоге `bin/`:

- `server` — сервер (нативная платформа);
- `client-linux-amd64`, `client-windows-amd64.exe`, `client-darwin-amd64`, `client-darwin-arm64` — клиент под Linux/Windows/macOS.

## 2. Запуск сервера

```bash
./start_server.sh
```

Скрипт поднимает зависимости (`docker compose up -d db filestorage`),
при необходимости собирает бинарник и запускает `bin/server` (по умолчанию
HTTP на `:8080`, значения подключения — в самом скрипте).

## 3. Запуск клиента: вход

Первый запуск — регистрация, далее вход. После входа открывается TUI.

```bash
./bin/client-linux-amd64 register
> логин: alice
> пароль: 
./bin/client-linux-amd64 login
> логин: alice
> пароль: 
```

Логин и пароль запрашивается интерактивно. Адрес
сервера — `http://localhost:8080` по умолчанию (переопределяется `--server`
или сохраняется после первого входа).

## 4. Добавление логина и пароля

```bash
./bin/client-linux-amd64 add
```

Далее по подсказкам:

```
тип (login_password|text|card|binary): login_password
имя: GitHub
логин: user@example.com
пароль: ********
```

## 5. Добавление файла

```bash
./bin/client-linux-amd64 add -f /path/to/file.bin
```

Затем выбрать тип `binary` и указать имя:

```
тип (login_password|text|card|binary): binary
имя: backup
```

Файл шифруется на клиенте и загружается на сервер (S3) потоком.

## 6. Скачивание файла

Узнать идентификатор объекта можно командой `list` (или в TUI):

```bash
./bin/client-linux-amd64 list
```

Затем скачать:

```bash
./bin/client-linux-amd64 download <object-id>
```

Клиент спросит путь для сохранения, скачает и расшифрует файл.
