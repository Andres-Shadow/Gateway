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

Variables disponibles:

```text
PORT=8080
SQLITE_PATH=gateway.db
DEBUG=false
```

## API administrativa

Crear ruta:

```powershell
Invoke-RestMethod -Method Post http://localhost:8080/admin/routes `
  -ContentType "application/json" `
  -Body '{"path":"/users/*","target_url":"http://localhost:8081","methods":"GET,POST","is_active":true}'
```

Listar rutas:

```powershell
Invoke-RestMethod http://localhost:8080/admin/routes
```

Actualizar ruta:

```powershell
Invoke-RestMethod -Method Put http://localhost:8080/admin/routes/1 `
  -ContentType "application/json" `
  -Body '{"methods":"GET,POST,PUT"}'
```

Eliminar ruta:

```powershell
Invoke-RestMethod -Method Delete http://localhost:8080/admin/routes/1
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
