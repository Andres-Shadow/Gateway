# Lightweight Dynamic API Gateway

API Gateway dinamico y liviano en Go con rutas persistidas en SQLite, cache en memoria y reverse proxy con hot reload simple.

## Stack

- Go
- chi
- GORM
- SQLite
- `net/http/httputil.ReverseProxy`
- configuracion por variables de entorno

## Ejecutar

```powershell
go mod tidy
go run ./cmd/gateway
```

Luego abre el panel web:

```text
http://localhost:8080/dashboard/
```

El frontend esta embebido en el binario Go, asi que no requiere Node, build separado ni servidor adicional.

Desde el dashboard puedes:

- iniciar sesion con JWT admin
- listar, buscar, crear, editar, activar/desactivar y eliminar rutas
- configurar health path, rewrites, headers y transforms simples
- abrir en una nueva pestana el endpoint health registrado para cada ruta

Variables disponibles:

```text
PORT=8080
SQLITE_PATH=gateway.db
DEBUG=false
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin
ADMIN_JWT_SECRET=change-me-in-production
ADMIN_TOKEN_TTL=12h
CORS_ALLOWED_ORIGINS=http://localhost:3000
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=120
PROXY_MAX_RETRIES=2
PROXY_RETRY_BACKOFF=100ms
CIRCUIT_FAILURE_LIMIT=3
CIRCUIT_COOLDOWN=30s
HEALTH_CHECK_ENABLED=true
HEALTH_CHECK_INTERVAL=30s
HEALTH_CHECK_TIMEOUT=2s
```

## API administrativa

Obtener token admin:

```powershell
$token = (Invoke-RestMethod -Method Post http://localhost:8080/admin/auth/token `
  -ContentType "application/json" `
  -Body '{"username":"admin","password":"admin"}').token
```

Crear ruta:

```powershell
Invoke-RestMethod -Method Post http://localhost:8080/admin/routes `
  -Headers @{ Authorization = "Bearer $token" } `
  -ContentType "application/json" `
  -Body '{"path":"/users/*","target_url":"http://localhost:8081","methods":"GET,POST","is_active":true,"health_check_path":"/healthz"}'
```

Listar rutas:

```powershell
Invoke-RestMethod http://localhost:8080/admin/routes `
  -Headers @{ Authorization = "Bearer $token" }
```

Actualizar ruta:

```powershell
Invoke-RestMethod -Method Put http://localhost:8080/admin/routes/1 `
  -Headers @{ Authorization = "Bearer $token" } `
  -ContentType "application/json" `
  -Body '{"methods":"GET,POST,PUT"}'
```

Eliminar ruta:

```powershell
Invoke-RestMethod -Method Delete http://localhost:8080/admin/routes/1 `
  -Headers @{ Authorization = "Bearer $token" }
```

Campos opcionales por ruta:

```json
{
  "rewrite_prefix_from": "/api/v1/users",
  "rewrite_prefix_to": "/users",
  "request_headers_set": { "X-Gateway": "lightweight" },
  "request_headers_remove": ["X-Debug"],
  "response_headers_set": { "X-Proxied-By": "gateway" },
  "response_headers_remove": ["Server"],
  "request_body_transform": "uppercase",
  "response_body_transform": "lowercase"
}
```

## Proxy

Si existe una ruta activa:

```json
{
  "path": "/users/*",
  "target_url": "http://localhost:8081",
  "methods": "GET,POST"
}
```

Entonces una request a:

```text
GET http://localhost:8080/users/123
```

se reenvia a:

```text
GET http://localhost:8081/users/123
```

## Arquitectura

```text
cmd/gateway              entrypoint
internal/config          variables de entorno
internal/database        conexion y migraciones GORM
internal/models          modelos persistentes
internal/repositories    acceso a datos
internal/services        casos de uso y hot reload de cache
internal/proxy           registry thread-safe y reverse proxy
internal/handlers        API REST administrativa
internal/router          wiring HTTP
internal/middleware      middlewares reutilizables
configs                  ejemplos de configuracion
```
