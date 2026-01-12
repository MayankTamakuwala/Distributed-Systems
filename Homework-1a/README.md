# Homework 1a: Albums API

This project is a small REST API written in Go using the Gin framework.
It supports full CRUD operations (Create, Read, Update, Delete) on albums.

## What `main.go` does

`Homework-1a/main.go` defines:
- A simple `album` struct that matches the JSON fields in requests and responses.
- An in-memory slice (`albums`) that acts like a tiny database.
- HTTP handlers for all CRUD operations:
  - `GET /albums` list all albums (Read)
  - `GET /albums/:id` fetch by id (Read)
  - `POST /albums` create a new album (Create)
  - `PUT /albums/:id` replace an album (Update)
  - `PATCH /albums/:id` update specific fields (Update)
  - `DELETE /albums/:id` remove an album (Delete)
- Helper functions for validation, normalization, and ID generation to keep data clean and consistent.

## Run

```bash
go run ./Homework-1a
```

## Curl examples (success + error cases)

### GET requests (Read)

GET all albums (success)
```bash
curl -s http://localhost:8080/albums
```

GET album by id (success)
```bash
curl -s http://localhost:8080/albums/2
```

GET album by id (error: not found)
```bash
curl -s http://localhost:8080/albums/999
```

### POST requests (Create)

POST create album (success, auto-ID)
```bash
curl -s -X POST http://localhost:8080/albums \
  -H "Content-Type: application/json" \
  -d '{"title":"Giant Steps","artist":"John Coltrane","price":42.5}'
```

POST create album (success, custom ID)
```bash
curl -s -X POST http://localhost:8080/albums \
  -H "Content-Type: application/json" \
  -d '{"id":"10","title":"Moanin","artist":"Art Blakey","price":19.5}'
```

POST create album (error: duplicate ID)
```bash
curl -s -X POST http://localhost:8080/albums \
  -H "Content-Type: application/json" \
  -d '{"id":"1","title":"Duplicate","artist":"Copy","price":10}'
```

POST create album (error: invalid JSON)
```bash
curl -s -X POST http://localhost:8080/albums \
  -H "Content-Type: application/json" \
  -d '{"title":"Bad JSON","artist":"Oops","price":}'
```

POST create album (error: missing required fields)
```bash
curl -s -X POST http://localhost:8080/albums \
  -H "Content-Type: application/json" \
  -d '{"artist":"Unknown","price":10}'
```

### PUT requests (Full update)

PUT replace album (success)
```bash
curl -s -X PUT http://localhost:8080/albums/2 \
  -H "Content-Type: application/json" \
  -d '{"title":"New Title","artist":"New Artist","price":12.75}'
```

PUT replace album (error: id mismatch)
```bash
curl -s -X PUT http://localhost:8080/albums/2 \
  -H "Content-Type: application/json" \
  -d '{"id":"3","title":"Bad","artist":"Mismatch","price":12.75}'
```

PUT replace album (error: not found)
```bash
curl -s -X PUT http://localhost:8080/albums/999 \
  -H "Content-Type: application/json" \
  -d '{"title":"Missing","artist":"Nobody","price":12.75}'
```

### PATCH requests (Partial update)

PATCH update album (success)
```bash
curl -s -X PATCH http://localhost:8080/albums/1 \
  -H "Content-Type: application/json" \
  -d '{"price":25.0}'
```

PATCH update album (error: invalid field values)
```bash
curl -s -X PATCH http://localhost:8080/albums/1 \
  -H "Content-Type: application/json" \
  -d '{"price":0}'
```

PATCH update album (error: not found)
```bash
curl -s -X PATCH http://localhost:8080/albums/999 \
  -H "Content-Type: application/json" \
  -d '{"price":15.0}'
```

### DELETE requests (Delete)

DELETE album (success)
```bash
curl -s -X DELETE http://localhost:8080/albums/3
```

DELETE album (error: not found)
```bash
curl -s -X DELETE http://localhost:8080/albums/999
```
