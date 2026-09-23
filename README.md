# Back-office

API y panel de administración para operar un e-commerce chico: catálogo, inventario
multi-bodega, clientes, pedidos, pagos, envíos, cupones y usuarios con roles. Backend en
Go (chi) sobre SQLite embebido; panel de administración en Next.js.

Pensado para un equipo de 3 a 10 personas (ventas, operaciones, administración), no para
un marketplace masivo: por eso no hay colas, no hay caché distribuida y el motor de
descuentos es simple.

## Cómo correrlo

Requiere Go ≥ 1.25.

```bash
go run ./cmd/api                  # API en :8080 (PORT lo cambia)
cd web && npm install && npm run dev   # panel en :3000, apunta a NEXT_PUBLIC_API_URL
```

`POST /admin/reset` recrea el esquema entero y vuelve a sembrar los datos de arranque.
`GET /admin/seed-info` devuelve la contraseña y el listado de usuarios sembrados, para
poder autenticarse sin leer el código fuente. Ninguno de los dos requiere autenticación.

## Autenticación

`POST /auth/login {email, password}` devuelve `access_token` (15 min) y `refresh_token`
(7 días). El resto de la API exige `Authorization: Bearer <access_token>`.
`POST /auth/refresh {refresh_token}` canjea uno vigente por un access token nuevo.
`POST /auth/logout` invalida el access token actual. `GET /auth/me` devuelve la sesión.

**Un usuario desactivado no puede autenticarse**, y si su token ya estaba emitido deja de
servir de inmediato: cada request revalida contra la base, no solo contra la firma del
token.

## Roles y permisos

Tres roles fijos: `admin`, `operador`, `vendedor`. Cada uno tiene un conjunto de permisos
asignado (`GET /roles` los lista). Un rol sin el permiso que una acción requiere recibe
403. Los reportes financieros (`/reportes/*`) son visibles **solo para `admin`**: es
deliberado, no un descuido — la cifra de ventas y el detalle de qué se vendió no se
comparten con todo el equipo.

## Catálogo

**Productos.** `sku` es único. `precio_centavos` es siempre mayor a 0. Un producto se
crea sin publicar (`publicado: false`); publicarlo es una acción aparte
(`POST /productos/{id}/publicar`). Un producto con pedidos asociados no se puede
eliminar (409). El stock de un producto es la suma de sus existencias en todas las
bodegas (`stock_total`); se ajusta a mano con `/productos/{id}/ajustar-stock`, que nunca
lo deja negativo.

**Categorías.** Una categoría con productos asociados no se puede eliminar (409): hay
que reasignar sus productos primero. `POST /categorias/reordenar` recibe la lista
completa de ids en el orden deseado.

**Listas de precio por cliente.** Un cliente puede tener un precio distinto para un
producto puntual (`/precios-cliente`); cuando existe, ese precio manda por sobre el de
catálogo al armar un pedido para ese cliente.

## Clientes

Un cliente dado de baja (`activo: false`) no puede generar pedidos nuevos (ver
*Pedidos*), pero sigue existiendo: se lista y se consulta igual que uno activo.

**Direcciones.** Un cliente con al menos una dirección tiene **exactamente una**
marcada como `es_principal`. Marcar una nueva dirección (al crearla o al editarla) como
principal desmarca automáticamente la que lo era — en cualquiera de los dos casos, alta o
edición. Si se borra la dirección principal y quedan otras, la más antigua de las
restantes pasa a ser principal de inmediato: un cliente con direcciones nunca se queda
sin una principal.

## Inventario

Cada producto tiene existencias independientes por bodega (`stock_bodega`). Tres formas
de moverlas:

- `/inventario/movimientos` (entrada/salida): ajusta el stock de **una** bodega.
- `/bodegas/transferir-stock`: mueve cantidad de una bodega a otra. **Exige
  `idempotency_key`**: reenviar la misma transferencia con la misma clave no la aplica
  dos veces — devuelve el mismo resultado sin volver a tocar el stock. Es la única
  escritura del sistema con esta garantía, porque es la única que un cliente HTTP podría
  reintentar por un timeout sin saber si la primera vez llegó a aplicarse.
- `/productos/{id}/ajustar-stock`: corrección manual en una bodega puntual.

Una salida o transferencia que dejaría el stock en negativo es 409, nunca se aplica
parcial.

## Pedidos

Estados: `borrador → confirmado → {cancelado | reembolsado}`. Un pedido nace en
`borrador`; en ese estado se le agregan y quitan items (`/pedidos/{id}/items`). Un
pedido sin items no se puede confirmar. Al confirmarlo
(`POST /pedidos/{id}/actualizar-estado {"estado":"confirmado"}`) los totales quedan
fijos.

**Cálculo del total.** `subtotal` es la suma de `cantidad × precio_unitario` de los
items. Si el pedido tiene un cupón, el `descuento` se calcula **sobre el subtotal**
(nunca puede superar el subtotal). Los `impuestos` (19%) se calculan sobre
`subtotal − descuento`, y el `total` es `subtotal − descuento + impuestos`. Esta cadena
es la que cualquier cliente del catálogo puede verificar sumando los items.

**Cliente dado de baja.** No puede generar pedidos nuevos: 409.

**Cancelar.** Un pedido en `borrador` o `confirmado` se puede cancelar
(`/pedidos/{id}/cancelar`); uno `reembolsado` no. Un `vendedor` **solo puede cancelar
sus propios pedidos** — los que él mismo generó como `vendedor_id`; cancelar el pedido de
otro vendedor es 403 para ese rol (`admin` y `operador` no tienen esa restricción).

**Reembolsar.** Solo un pedido `confirmado` con un pago `confirmado` se puede reembolsar;
anula el pago y marca el pedido como `reembolsado`.

**Filtro `estado` en `GET /pedidos`.** Acepta únicamente `borrador`, `confirmado`,
`cancelado` o `reembolsado`; cualquier otro valor es 400, nunca una lista vacía
silenciosa.

## Pagos

Un pedido solo puede tener un pago activo a la vez (no anulado). Un pago nace
`pendiente`; **confirmar un pago ya confirmado es 409**, no un no-op silencioso — un pago
representa dinero que entró una sola vez, confirmarlo dos veces no puede contarse como
si hubiera entrado dos. Anular un pago ya anulado también es 409.

**El monto tiene que coincidir.** `monto_centavos` de un pago nuevo tiene que ser
exactamente igual a `total_centavos` del pedido al que pertenece; un monto distinto (de
más o de menos) es 400. Este sistema no maneja pagos parciales.

## Envíos

Un pedido confirmado sin envío previo puede generar uno (`pendiente`). Pasar a
`en_transito` exige `tracking`. Un envío `entregado` no cambia de estado nunca más.
`POST /envios/{id}/marcar-entregado` solo funciona desde `en_transito`.

**El orden importa.** Un envío solo puede marcarse `entregado` después de haber pasado
por `en_transito` — nunca directo desde `pendiente`. Saltarse `en_transito` es 409.

## Cupones

`tipo: "porcentaje"` (1–100) o `"fijo"` (centavos). Un cupón inactivo no se puede aplicar
a un pedido nuevo (`GET /cupones/validar?codigo=...` lo confirma antes de intentarlo). Un
cupón referenciado por algún pedido no se puede eliminar.

## Reseñas

Nace `pendiente`; se modera a `aprobada` o `rechazada` (`POST /resenas/{id}/moderar`).
Una reseña ya moderada no se vuelve a moderar.

## Usuarios

Un usuario nuevo arranca con la contraseña de siembra (ver `/admin/seed-info`);
`POST /usuarios/{id}/resetear-password` la vuelve a esa misma contraseña. Cambiar el rol
de un usuario (`/usuarios/{id}/cambiar-rol`) surte efecto en su próximo access token — el
que ya tiene emitido conserva el rol con el que se emitió hasta que expire o se refresque.

## Notas de diseño (decisiones que parecen errores y no lo son)

**`GET /usuarios/{id}` de un usuario desactivado devuelve 403, no 404.** El usuario
existe; lo que no hay es acceso a su ficha. Un 404 permitiría mapear qué emails existen
en el sistema probando ids uno por uno — por eso la respuesta no distingue "no existe" de
"existe pero no podés verlo". `GET /usuarios?activo=false` sí lista a los desactivados,
porque ahí no se expone la ficha individual de nadie.

**Los reportes son solo para `admin`.** Un `operador` que llama a `/reportes/*` recibe
403 aunque el endpoint exista y su token sea válido — no es un permiso que falte
asignar, es una decisión: la cifra de ventas no es información operativa del día a día.

**Cancelar un pedido no anula su pago.** `POST /pedidos/{id}/cancelar` sobre un pedido
`confirmado` que ya tiene un pago `confirmado` lo deja exactamente así: el pedido pasa a
`cancelado` y el pago sigue `confirmado`. No es un olvido — anular dinero que ya entró es
una decisión financiera distinta a cancelar un pedido operativamente, y tiene su propio
endpoint (`/pedidos/{id}/reembolsar`) con su propio registro. Un pedido cancelado con un
pago todavía confirmado es la señal de que falta iniciar ese reembolso, no un estado
inválido.

## Estructura

```
cmd/api/main.go              wiring del router y los middlewares
internal/store/              esquema (schema.sql), apertura de la base y semilla
internal/auth/                JWT, roles y permisos
internal/httpx/                helpers de request/response
internal/handlers/            un archivo por recurso
web/                          panel Next.js (App Router)
```
