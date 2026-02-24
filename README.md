# Blog API

API блог-платформы. JWT аутентификация. Круд по блогам и комментам. Функционал отлаженной публикации, многопоточная обработка.

## Доступ
- Апи: `http://localhost:8080`
- Адимнка: `http://localhost:8081`
- Свагер: `http://localhost:8082`


## Сервисы
- **Postgres** — база данных
- **Redis** — троттлинг
- **Swagger UI** — документация
- **Adminer** — админка БД

## Эндпоинты

### Auth
- `POST /api/register` — регистрация пользователя
```
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"username":"john","email":"john@example.com","password":"StrongPassword12345?"}'
```

- `POST /api/login` — вход пользователя
```
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email":"john@example.com","password":"StrongPassword12345?"}'
```

- `POST /api/refresh` — обновление токена
```
curl -X POST http://localhost:8080/api/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh-token>"}'
```


### Users
- `GET /api/users/{userID}` — получение профиля пользователя (auth)
```
curl -X GET http://localhost:8080/api/users/1 \
  -H "Authorization: Bearer <access-token>"
```

### Posts
- `GET /api/posts` — список постов с пагинацией
```
curl -X GET "http://localhost:8080/api/posts?limit=10&offset=0"
```
- `POST /api/posts` — создание поста (auth)
```
curl -X POST http://localhost:8080/api/posts \
  -H "Authorization: Bearer <access-token>" \
  -H "Content-Type: application/json" \
  -d '{"title":"New Post","content":"Hello world"}'
```
- `GET /api/posts/{postID}` — получение поста 
```
curl -X GET http://localhost:8080/api/posts/1
```
- `PUT /api/posts/{postID}` — обновление поста (auth)
```
curl -X PUT http://localhost:8080/api/posts/1 \
  -H "Authorization: Bearer <access-token>" \
  -H "Content-Type: application/json" \
  -d '{"title":"Updated","content":"Updated content"}'
```
- `DELETE /api/posts/{postID}` — удаление поста (auth)
```
curl -X DELETE http://localhost:8080/api/posts/1 \
  -H "Authorization: Bearer <access-token>"
```
#### Посты с отложенной публикацией
Такие посты не отображаются в ответах публичных эндпойнтов (`GET /api/posts` и `GET /api/posts/{post_id}`).
Для получения собственных отложенных постов выведен отдельный домен `delayed`.
Поддерживает получение списком и одиночных записей. Требует аутентификации.
Отдаёт только собственные отложенные записи.  
Для операций записи и редактирования применяются эндпойнты `/api/posts`

- `POST /api/delayed` — создание отложенного поста (auth)
```
curl -X POST http://localhost:8080/api/posts \
  -H "Authorization: Bearer <access-token>" \
  -H "Content-Type: application/json" \
  -d '{"title":"Future Post","content":"This will be published later","publish_at":"2026-02-25T10:00:00+03:00"}'
```

- `GET /api/delayed` — список собственных отложенных постов с пагинацией (auth)
```
curl -X GET "http://localhost:8080/api/delayed?limit=10&offset=0" \
  -H "Authorization: Bearer <access-token>"
```
`GET /api/delayed/{postID}` — получение собственного отложенного поста по ID (auth)
```
curl -X GET http://localhost:8080/api/delayed/1 \
  -H "Authorization: Bearer <access-token>"
```

### Comments
- `GET /api/posts/{postID}/comments` — список комментариев с пагинацией
```
curl -X GET "http://localhost:8080/api/posts/1/comments?limit=10&offset=0"
```
- `POST /api/posts/{postID}/comments` — создание комментария (auth)
```
curl -X POST http://localhost:8080/api/posts/1/comments \
  -H "Authorization: Bearer <access-token>" \
  -H "Content-Type: application/json" \
  -d '{"content":"Nice post!"}'
```
- `PUT /api/posts/{postID}/comments/{commentID}` — обновление комментария (auth)
```
curl -X PUT http://localhost:8080/api/posts/1/comments/2 \
  -H "Authorization: Bearer <access-token>" \
  -H "Content-Type: application/json" \
  -d '{"content":"Updated comment"}'
```
- `DELETE /api/posts/{postID}/comments/{commentID}` — удаление комментария (auth)
```
curl -X DELETE http://localhost:8080/api/posts/1/comments/2 \
  -H "Authorization: Bearer <access-token>"
```


## Пагинация
- Параметры: `limit` и `offset`

## Миграции
- SQL скрипт: `migrations/001_init_schema.sql`
- Автоматически применяются при поднятии контейнера Postgres через Docker

## Запуск
```bash
docker-compose up -d
go run cmd/api/main.go
```

# Запуск тестов
```
go test ./...
```