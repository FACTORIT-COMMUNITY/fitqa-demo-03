-- Esquema del back-office. Vive en memoria (sqlite) y se recrea entero en cada reset.

CREATE TABLE roles (
    nombre TEXT PRIMARY KEY
);

CREATE TABLE permisos (
    nombre TEXT PRIMARY KEY
);

CREATE TABLE rol_permisos (
    rol     TEXT NOT NULL REFERENCES roles(nombre),
    permiso TEXT NOT NULL REFERENCES permisos(nombre),
    PRIMARY KEY (rol, permiso)
);

CREATE TABLE usuarios (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre        TEXT NOT NULL,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    rol           TEXT NOT NULL REFERENCES roles(nombre),
    activo        INTEGER NOT NULL DEFAULT 1,
    creado_en     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE categorias (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre    TEXT NOT NULL,
    orden     INTEGER NOT NULL DEFAULT 0,
    creado_en TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE productos (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    sku              TEXT NOT NULL UNIQUE,
    nombre           TEXT NOT NULL,
    categoria_id     INTEGER,
    precio_centavos  INTEGER NOT NULL,
    publicado        INTEGER NOT NULL DEFAULT 0,
    creado_en        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE bodegas (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre    TEXT NOT NULL,
    direccion TEXT NOT NULL
);

CREATE TABLE stock_bodega (
    producto_id INTEGER NOT NULL REFERENCES productos(id),
    bodega_id   INTEGER NOT NULL REFERENCES bodegas(id),
    cantidad    INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (producto_id, bodega_id)
);

CREATE TABLE movimientos_stock (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    producto_id      INTEGER NOT NULL REFERENCES productos(id),
    bodega_id        INTEGER NOT NULL REFERENCES bodegas(id),
    bodega_destino_id INTEGER REFERENCES bodegas(id),
    tipo             TEXT NOT NULL, -- entrada | salida | transferencia | ajuste
    cantidad         INTEGER NOT NULL,
    referencia       TEXT,
    idempotency_key  TEXT,
    creado_en        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE proveedores (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre    TEXT NOT NULL,
    email     TEXT NOT NULL UNIQUE,
    telefono  TEXT
);

CREATE TABLE clientes (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre    TEXT NOT NULL,
    email     TEXT NOT NULL UNIQUE,
    activo    INTEGER NOT NULL DEFAULT 1,
    creado_en TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE direcciones (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    cliente_id    INTEGER NOT NULL REFERENCES clientes(id),
    calle         TEXT NOT NULL,
    ciudad        TEXT NOT NULL,
    es_principal  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE precios_cliente (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    cliente_id       INTEGER NOT NULL REFERENCES clientes(id),
    producto_id      INTEGER NOT NULL REFERENCES productos(id),
    precio_centavos  INTEGER NOT NULL,
    UNIQUE (cliente_id, producto_id)
);

CREATE TABLE cupones (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    codigo  TEXT NOT NULL UNIQUE,
    tipo    TEXT NOT NULL, -- porcentaje | fijo
    valor   INTEGER NOT NULL,
    activo  INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE pedidos (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    cliente_id          INTEGER NOT NULL REFERENCES clientes(id),
    vendedor_id         INTEGER NOT NULL REFERENCES usuarios(id),
    estado              TEXT NOT NULL DEFAULT 'borrador', -- borrador|confirmado|cancelado|reembolsado
    subtotal_centavos   INTEGER NOT NULL DEFAULT 0,
    descuento_centavos  INTEGER NOT NULL DEFAULT 0,
    impuestos_centavos  INTEGER NOT NULL DEFAULT 0,
    total_centavos      INTEGER NOT NULL DEFAULT 0,
    cupon_codigo        TEXT,
    creado_en           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE pedido_items (
    id                       INTEGER PRIMARY KEY AUTOINCREMENT,
    pedido_id                INTEGER NOT NULL REFERENCES pedidos(id),
    producto_id              INTEGER NOT NULL REFERENCES productos(id),
    cantidad                 INTEGER NOT NULL,
    precio_unitario_centavos INTEGER NOT NULL
);

CREATE TABLE pagos (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    pedido_id       INTEGER NOT NULL REFERENCES pedidos(id),
    monto_centavos  INTEGER NOT NULL,
    estado          TEXT NOT NULL DEFAULT 'pendiente', -- pendiente|confirmado|anulado
    metodo          TEXT NOT NULL,
    creado_en       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE envios (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    pedido_id     INTEGER NOT NULL REFERENCES pedidos(id),
    estado        TEXT NOT NULL DEFAULT 'pendiente', -- pendiente|en_transito|entregado
    transportista TEXT,
    tracking      TEXT,
    creado_en     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE resenas (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    producto_id   INTEGER NOT NULL REFERENCES productos(id),
    cliente_id    INTEGER NOT NULL REFERENCES clientes(id),
    calificacion  INTEGER NOT NULL,
    comentario    TEXT,
    estado        TEXT NOT NULL DEFAULT 'pendiente', -- pendiente|aprobada|rechazada
    creado_en     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE notificaciones (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    usuario_id  INTEGER NOT NULL REFERENCES usuarios(id),
    mensaje     TEXT NOT NULL,
    leida       INTEGER NOT NULL DEFAULT 0,
    creado_en   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE auditoria (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    usuario_id  INTEGER REFERENCES usuarios(id),
    accion      TEXT NOT NULL,
    recurso     TEXT NOT NULL,
    recurso_id  INTEGER,
    creado_en   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
