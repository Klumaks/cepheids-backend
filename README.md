# cepheids-backend

## Лабораторная 3 — веб-сервис REST API

### Домен «Услуга» (спектральный класс цефеиды)

- `GET /api/cepheids` — список опубликованных услуг; необязательные query-фильтры `min_slope`, `max_slope`, `min_b`, `max_b` (диапазоны полей по теме). Ответ 200: массив объектов с признаками `is_mine` (1, если создатель — текущий пользователь) и `is_liked`.
- `GET /api/cepheids/feed` — лента: первая опубликованная услуга, id не указывается.
- `GET /api/cepheids/feed/:id` — опубликованная услуга по id; 404, если не существует или удалена.
- `GET /api/cepheids/feed/:id?next=true` — следующая опубликованная после id, в конце ленты заворот на первую.
- `GET /api/cepheids/draft` — единственный черновик текущего пользователя, id с клиента не передаётся; 404, если черновика нет.
- `POST /api/cepheids` — создание черновика; принимает `name` (текст) и файлы `image`, `video` (multipart/form-data, именно файлы, а не URL). Файлы сохраняются в MinIO под сгенерированными латинскими именами, имена пишутся в поля БД. Ответ 201 + объект черновика.
- `PUT /api/cepheids/:id/publish` — публикация своего черновика (смена статуса); JSON-тело: `description`, `pl_slope`, `pl_intercept`. Ответ 200 + обновлённый объект; 404, если черновик чужой или уже опубликован.
- `DELETE /api/cepheids/:id` — логическое удаление своей услуги (status = 'deleted', updated_at = NOW(), SQL UPDATE через курсор без ORM). Ответ 200; чужая услуга — 403.
- `POST /api/cepheids/:id/like` — лайк текущего пользователя; JSON `{"like": 1}` ставит, `{"like": 0}` отменяет. Ответ 200.

### Домен «Пользователь»

- `POST /api/users/register` — регистрация; JSON `{login, password}`. Ответ 201 + `{id, login}` (пароль не сериализуется); 409, если логин занят.
- `POST /api/users/login` — заглушка аутентификации для ЛР4. Ответ 200.
- `POST /api/users/logout` — заглушка деавторизации для ЛР4. Ответ 200.

### Соглашения ответов

- Набор полей JSON всегда одинаковый; незаполненные значения приходят как `null` / `""`.
- Статус услуги в ответах не передаётся (хранится только в БД); коды состояния в тело не дублируются.
- Успех: 200 / 201; ошибки: 400, 403, 404, 409 без текстовых сообщений.

## Таблицы базы данных

### users — зарегистрированные пользователи
| Поле | Тип | Описание |
|---|---|---|
| id | SERIAL, PK | первичный ключ |
| login | VARCHAR(25), UNIQUE, NOT NULL | логин |
| password | VARCHAR(100), NOT NULL | пароль (в ЛР4 — хэш) |

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
