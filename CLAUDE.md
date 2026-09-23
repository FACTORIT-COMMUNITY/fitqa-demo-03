# Back-office — contexto para agentes

API en Go (chi) + panel en Next.js. El README tiene la spec completa: endpoints, reglas
de negocio y notas de diseño. Leelo antes de tocar nada.

## Levantar el sistema

```bash
go run ./cmd/api           # API en :8080 (PORT la cambia)
```

`GET /health` responde `{"estado":"ok"}` cuando está listo. El panel (`web/`) es
opcional para probar la API: `cd web && npm install && npm run dev`, en :3000, apuntando
a `NEXT_PUBLIC_API_URL` (default `http://localhost:8080`).

## Autenticarse

`GET /admin/seed-info` (sin auth) devuelve la contraseña de siembra y el email de cada
usuario con su rol. `POST /auth/login` con esas credenciales da el `access_token` a usar
en `Authorization: Bearer <token>` para todo lo demás.

## Estado y aislamiento entre pruebas

La base es SQLite **en memoria**, compartida por todo el proceso.

- **Llamá `POST /admin/reset` antes de cada caso que escriba.** No requiere auth. Vuelve
  la base a la semilla documentada en el README.
- Reiniciar el proceso equivale a un reset.
- El reset es de **datos**, no de sesiones: un `access_token` emitido antes de un reset
  sigue sirviendo después, mientras no haya expirado (15 minutos) — los usuarios y roles
  vuelven a existir con el reset, así que el token vuelve a resolver contra algo válido.

## Convenciones del código

- Go 1.25, sin ORM: `database/sql` directo con `modernc.org/sqlite` (driver puro Go, sin
  CGO). Un archivo por recurso en `internal/handlers/`.
- Todas las cantidades de dinero son enteros en centavos (`*_centavos`), nunca floats.
- Los errores se responden como `{"error": "...", "detalles": [...]}"`. `detalles` solo
  aparece en validaciones (400).
- El panel Next (`web/`) es deliberadamente liviano: guarda el JWT en `localStorage` y
  consume la API por `fetch`, sin lógica de negocio propia.
- Español sin tildes en identificadores y comentarios del código; con tildes en el texto
  que ve el usuario y en la documentación.

## Qué hace falta para probar

El proceso de la API levantado y un cliente HTTP alcanza para toda la superficie de
negocio. El panel Next es un consumidor más de esa API, no una segunda fuente de verdad.
