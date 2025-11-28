# Swagger API Documentation

## Viewing the Documentation

Once the Go API server is running, visit:
- **Swagger UI**: http://localhost:8000/swagger/index.html
- **OpenAPI JSON**: http://localhost:8000/swagger/doc.json

## Regenerating Documentation

If you modify API handlers or add new endpoints, regenerate the Swagger docs:

```bash
cd go-api
~/go/bin/swag init -g cmd/api/main.go --output docs
```

Or use the Makefile:
```bash
make swagger
```

## Adding Documentation to Endpoints

Swagger annotations are in `internal/handlers/swagger.go`. Keep them minimal:

```go
// FunctionName godoc
// @Summary Short description
// @Tags tag-name
// @Param name type datatype required "description"
// @Success 200 {object} ModelName
// @Router /endpoint [method]
```

## Available Endpoints

- `POST /api/v1/documents` - Create document
- `GET /api/v1/documents/{id}` - Get document
- `GET /api/v1/documents` - List documents
- `PUT /api/v1/documents/{id}` - Update document
- `DELETE /api/v1/documents/{id}` - Delete document
- `GET /api/v1/search` - Search documents
