# Product API (Homework 5)

## 1) Run Locally (Go)

```bash
go mod tidy
go run .
```

Server starts on `http://localhost:8080`.

## 2) Run via Docker

Build image:

```bash
docker build --platform linux/arm64 -t product-api:local .
```

Run container:

```bash
docker run --rm -p 8080:8080 product-api:local
```

Server is available at `http://localhost:8080`.

## 3) Postman Examples (All Status Codes)

### 200 OK

- Method: `GET`
- URL: `http://localhost:8080/products/1`

![GET 200](screenshots/GET%20200.png)

### 204 No Content

- Method: `POST`
- URL: `http://localhost:8080/products/1/details`
- Body (`raw` JSON):

```json
{
  "product_id": 1,
  "sku": "SKU-123",
  "manufacturer": "Acme",
  "category_id": 10,
  "weight": 50,
  "some_other_id": 77
}
```

![POST 204](screenshots/POST%20204.png)

### 400 Bad Request

- Method: `POST`
- URL: `http://localhost:8080/products/1/details`
- Body example (invalid):

```json
{
  "product_id": 0,
  "sku": "",
  "manufacturer": "",
  "category_id": 0,
  "weight": -1,
  "some_other_id": 0
}
```

![POST 400](screenshots/POST%20400.png)

### 404 Not Found

- Method: `GET`
- URL: `http://localhost:8080/products/999`

![GET 404](screenshots/GET%20404.png)

## 4) In-Memory Storage Choice

Using `map[int]Product` provides O(1) average lookup for common GET traffic.  
It keeps reads and updates simple and fast without adding external database setup.
