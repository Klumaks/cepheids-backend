# Cepheids Backend — ЛР4

Go-бэкенд определения расстояний методом цефеид. Спектральный класс содержит коэффициенты зависимости период–светимость: `M = pl_slope × log10(P) + pl_intercept`. В этой ветке реализованы каталог классов, черновики, публикация, медиа, лайки и авторизация наблюдателей. Расчётные заявки не добавляются в рамках ЛР4.

Основной источник требований: [README курса, ЛР4](https://github.com/iu5git/Web/blob/main/README.md#лабораторная-4). [Методичка Go](https://github.com/iu5git/Web/blob/main/tutorials/lab4-go/README.md) используется как пример. Ветка: `cepheids-jwt-swagger`, основана на `cepheids-rest-api`. HTML-шаблоны ЛР1–2 остаются в предыдущих ветках; фронтенд здесь не изменяется.

## Запуск

Проверено на Go 1.26.0, Gin 1.12.0, GORM 1.31.2, PostgreSQL 16 и Redis 7. Версии библиотек закреплены в `go.mod`.

Из корня репозитория в Linux / WSL:

```bash
docker compose up -d postgres minio redis adminer
go mod download
cp .env.example .env
```

Если `.env` уже существует, сохраните его настройки БД и добавьте переменные из `.env.example`. Для старого Compose в WSL используйте `docker-compose` вместо `docker compose`. Окружение процесса имеет приоритет над `.env`.

| Переменная | Назначение |
|---|---|
| `DATABASE_URL` | PostgreSQL DSN для сервера и обеих команд миграции |
| `JWT_SECRET` | Обязательный секрет HS256, минимум 32 байта, без значения по умолчанию |
| `JWT_TTL` | Срок JWT, по умолчанию `1h`, формат Go duration, минимум `1s` |
| `REDIS_ADDR` | По умолчанию `localhost:6379` |
| `REDIS_PASSWORD` | Пароль Redis, если сервер его требует; локальный Compose использует пустой |
| `CEPHEIDS_ADDR` | Адрес API, по умолчанию `:8080` |

Сгенерируйте секрет командой `openssl rand -hex 32` и запишите результат в `JWT_SECRET` вашего `.env`. Не включайте `.env` в Git.

```bash
go run ./cmd/migrate
go run ./cmd/hash-cepheid-passwords
go run ./cmd/cepheids
```

Первая команда создаёт/обновляет таблицы и частичный уникальный индекс одного черновика на наблюдателя. При существующих дубликатах миграция завершается ошибкой, записи автоматически не удаляются. Найти дубликаты:

```sql
SELECT creator_id, COUNT(*)
FROM spectral_classes
WHERE status = 'draft'
GROUP BY creator_id
HAVING COUNT(*) > 1;
```

Вторая команда преобразует открытые пароли ЛР3 в bcrypt, сохраняя логины, ID пользователей и все связи. Выполняется в транзакции; повторный запуск оставляет готовые хэши неизменными. Пароли длиннее 72 байт приводят к откату, а не обрезаются. Новая регистрация сразу сохраняет bcrypt; вход с открытым паролем в БД не поддерживается.

MinIO использует существующие настройки проекта: `localhost:9000`, бакет `cepheids`, локальные ключи `minioadmin`. Изображения и видео передаются файлами, имена генерируются на латинице. Поля `image_key` и `video_key` содержат ключи объектов; `image_url` и `video_url` формируются сервером.

- API: [http://localhost:8080/api/cepheids](http://localhost:8080/api/cepheids).
- Swagger: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html).
- MinIO Console: [http://localhost:9001](http://localhost:9001).
- Adminer: [http://localhost:8081](http://localhost:8081).

## JWT, blacklist и права доступа

Вход возвращает `access_token`, `token_type: "Bearer"`, `expires_in` в секундах и `user: {id, login}`. Передавайте `Authorization: Bearer <access_token>`. JWT содержит обязательные `sub` (ID наблюдателя), `jti` (уникальный ID токена), `iat`, `exp`; проверяются подпись HS256, срок и blacklist.

Выход записывает в Redis `cepheids:jwt:blacklist:<jti>` со значением `revoked` и TTL до `exp`. Сами JWT и пароли в Redis не хранятся. Redis использует AOF и отдельный Docker volume. Выход отзывает только переданный токен; новый вход выдаёт другой JWT.

Список опубликованных классов и лента доступны без токена, с неверным, просроченным или отозванным токеном. При действующем JWT сервер добавляет персональные признаки. При недоступном Redis публичное чтение продолжает работать как для гостя, защищённые операции с корректно подписанным токеном возвращают 503.

Черновик, создание, публикация, удаление, лайки и выход требуют авторизации. Создатель берётся из JWT; клиентские ID пользователя, статусы и даты не используются для изменения системных полей. Чужие черновики не выдаются, чужую запись нельзя публиковать или удалять.

## HTTP API

В таблице все URL начинаются с `/api`.

| Метод | URL | Доступ | Вход | Успех |
|---|---|---|---|---|
| GET | `/cepheids` | Публичный | Query: `min_slope`, `max_slope`, `min_b`, `max_b` | 200, массив опубликованных классов |
| GET | `/cepheids/feed` | Публичный | Без ID | 200, первый опубликованный класс |
| GET | `/cepheids/feed/:id` | Публичный | ID; `next=true` для следующего | 200, один класс; в конце ленты переход к первому |
| GET | `/cepheids/draft` | Наблюдатель | Без ID | 200, собственный черновик |
| POST | `/cepheids` | Наблюдатель | Multipart: `name`, файлы `image` и `video` | 201, созданный черновик |
| PUT | `/cepheids/:id/publish` | Владелец | JSON: `description`, `pl_slope`, `pl_intercept` | 200, обновлённый класс |
| DELETE | `/cepheids/:id` | Владелец | ID | 200, пустое тело; soft delete через SQL UPDATE |
| POST | `/cepheids/:id/like` | Наблюдатель | JSON: `{"like": 1}` или `{"like": 0}` | 200, пустое тело |
| POST | `/users/register` | Публичный | JSON: `login`, `password` | 201, `{id, login}` |
| POST | `/users/login` | Публичный | JSON: `login`, `password` | 200, JWT и пользователь |
| POST | `/users/logout` | Наблюдатель | Bearer JWT | 200 после успешного отзыва |

Файлы при создании необязательны, но для демонстрации требуется передать изображение и видео. Только опубликованные классы доступны в списке/ленте и для лайков. Единственный переход публикации — `draft → published`; удаление — `draft/published → deleted`. Обратного перехода в черновик нет.

Логин после удаления пробелов по краям: 1–25 символов; пароль: 1–72 байта. Название класса: 1–100 символов; описание: до 500 символов. Оба коэффициента при публикации обязательны, нулевые значения допускаются. Фильтры должны быть конечными числами, минимум не должен превышать максимум. Лайк требует явно указанное целое 0 или 1; повторная установка не создаёт дубликат.

### Ответы и ошибки

Объект спектрального класса: `id`, `name`, `description`, `image_key`, `video_key`, `pl_slope`, `pl_intercept`, `creator_id`, `created_at`, `updated_at`, `image_url`, `video_url`, `likes_count`, `is_mine`, `is_liked`.

Статус хранится только в БД и не сериализуется. Признаки `is_mine` и `is_liked` — числа **0/1**, как требует README курса; у гостя оба равны 0. Незаданные коэффициенты — `null`, пустые строки остаются строками, пустой список — `[]`. Пароль/хэш не сериализуется. Коды состояния в JSON не дублируются, ошибки возвращаются без текстового тела.

| Код | Причина |
|---|---|
| 400 | Некорректный JSON, ID, фильтр, поле или значение лайка |
| 401 | Нет действующего JWT для защищённого метода; неверные данные входа |
| 403 | Публикация/удаление чужого класса |
| 404 | Нет записи/черновика, класс удалён или недоступен по статусу |
| 409 | Логин занят или у наблюдателя уже есть черновик |
| 500 | Ошибка PostgreSQL, MinIO или сервера |
| 503 | Redis недоступен для защищённой операции |

## Swagger и демонстрация ЛР4

Swagger UI и спецификация обслуживаются бэкендом. Генерация после изменения аннотаций:

```bash
go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/cepheids/main.go --parseInternal
```

Результат: `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`; они включаются в ветку. В **Authorize** введите полностью `Bearer <access_token>`. У публичных GET нет обязательной схемы безопасности; чтобы проверить персональные признаки, передайте Authorization через Postman/Insomnia. Регистрация и вход доступны гостю как необходимые точки получения учётной записи и JWT.

Порядок показа по README курса:

1. **Скриншоты 1–3:** открыть Swagger в инкогнито, зарегистрировать наблюдателя при необходимости, выполнить вход, получить список классов; скопировать JWT из ответа в Authorization запроса Postman/Insomnia.
2. **Скриншоты 4–5:** получить список классов без Authorization и с неверным JWT — оба запроса возвращают 200.
3. **Скриншоты 6–8:** создать свой черновик с изображением и видео; выполнить PUT публикации гостем — 401; повторить с JWT владельца — 200 и сохранённые коэффициенты. Выполнить выход, показать blacklist в Redis; повторное использование JWT для изменения возвращает 401.
4. **Скриншот 9:** в UML переиспользовать `signature` сообщения из диаграммы классов. UML и Word-документация остаются отдельной частью сдачи.

Команды просмотра Redis:

```bash
docker compose exec -T redis redis-cli --scan --pattern 'cepheids:jwt:blacklist:*'
docker compose exec -T redis redis-cli GET 'cepheids:jwt:blacklist:<jti>'
docker compose exec -T redis redis-cli TTL 'cepheids:jwt:blacklist:<jti>'
```

До выхода blacklist может быть пустым — это нормально для JWT. TTL должен быть положительным и не превышать оставшийся срок токена.

## Проверки

```bash
go test ./...
go vet ./...
go build ./cmd/cepheids
```

По умолчанию выполняются автономные тесты JWT и middleware; интеграционные тесты явно пропускаются без тестовой БД. Для полного сценария нужны запущенные PostgreSQL, MinIO и Redis и отдельная БД с `cepheids_lab4_test` в имени:

```bash
docker exec postgres createdb -U myuser cepheids_lab4_test
export CEPHEIDS_TEST_DATABASE_URL='postgres://myuser:mypassword@localhost:5433/cepheids_lab4_test?sslmode=disable'
go test ./cmd/hash-cepheid-passwords -v
go test -race ./internal/app/auth ./internal/app/handler -v
```

Тесты проверяют реальную загрузку медиа, двух наблюдателей, владельца из JWT, один черновик при конкурентном создании, публикацию/удаление, фильтры, лайки, TTL blacklist и повторный вход после выхода. Отдельный тест проверяет преобразование открытого пароля и повторный запуск без изменения хэша/ID. Тестовые данные создаются только в отдельной БД; загруженные тестом медиа удаляются из MinIO. Автономный тест проверяет гостевое чтение и 503 при ошибке blacklist.

## Таблицы базы данных

### users — зарегистрированные пользователи
| Поле | Тип | Описание |
|---|---|---|
| id | SERIAL, PK | первичный ключ |
| login | VARCHAR(25), UNIQUE, NOT NULL | логин |
| password | VARCHAR(100), NOT NULL | bcrypt-хэш пароля |

### spectral_classes — черновики, опубликованные и удалённые услуги
| Поле | Тип | Описание |
|---|---|---|
| id | SERIAL, PK | первичный ключ |
| status | VARCHAR(15), NOT NULL, DEFAULT 'draft' | draft / published / deleted |
| name | VARCHAR(100), NOT NULL | название |
| description | VARCHAR(500), NULL | краткое описание |
| image_key | VARCHAR(100), NULL | имя изображения в MinIO |
| video_key | VARCHAR(100), NULL | имя видео в MinIO |
| pl_slope | NUMERIC, NULL | наклон a (поле по теме) |
| pl_intercept | NUMERIC, NULL | свободный член b (поле по теме) |
| creator_id | INTEGER, NOT NULL, FK → users.id | создатель (ON DELETE RESTRICT) |
| created_at | TIMESTAMP | дата создания |
| updated_at | TIMESTAMP | дата формирования / изменения |

### likes — связь многие-ко-многим пользователи ↔ услуги
| Поле | Тип | Описание |
|---|---|---|
| id | SERIAL, PK | первичный ключ |
| user_id | INTEGER, FK → users.id | кто лайкнул (ON DELETE RESTRICT) |
| spectral_class_id | INTEGER, FK → spectral_classes.id | что лайкнули (ON DELETE RESTRICT) |
| — | UNIQUE(user_id, spectral_class_id) | один лайк на пару |


Частичный уникальный индекс `idx_spectral_class_observer_draft` на `spectral_classes(creator_id)` применяется только к `status = 'draft'`. Он защищает ограничение одного черновика при одновременных запросах. Каскадное удаление не используется.
