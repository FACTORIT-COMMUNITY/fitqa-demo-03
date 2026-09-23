// Package store abre la base en memoria y sabe reconstruirla desde cero: el reset
// determinístico es lo que permite que la corrida N no herede lo que dejó la N-1.
package store

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// DB envuelve el *sql.DB con el reset como método de primera clase.
type DB struct {
	*sql.DB
}

// Open crea una base sqlite en memoria compartida por todas las conexiones del proceso
// (":memory:" con cache=shared) y aplica el esquema. La app entera comparte una sola DB.
func Open() (*DB, error) {
	sqlDB, err := sql.Open("sqlite", "file:fitqa03?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("abrir sqlite: %w", err)
	}
	// sqlite en memoria con cache compartido necesita una sola conexión activa a la vez
	// para no perder tablas cuando la última conexión del pool se cierra.
	sqlDB.SetMaxOpenConns(1)

	db := &DB{sqlDB}
	if err := db.Reset(); err != nil {
		return nil, err
	}
	return db, nil
}

// Reset recrea el esquema entero y vuelve a sembrar los datos deterministicos.
// Es lo que expone POST /admin/reset.
func (db *DB) Reset() error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(dropAllSQL); err != nil {
		return fmt.Errorf("drop esquema: %w", err)
	}
	if _, err := tx.Exec(schemaSQL); err != nil {
		return fmt.Errorf("crear esquema: %w", err)
	}
	if err := seed(tx); err != nil {
		return fmt.Errorf("sembrar: %w", err)
	}
	return tx.Commit()
}

const dropAllSQL = `
DROP TABLE IF EXISTS auditoria;
DROP TABLE IF EXISTS notificaciones;
DROP TABLE IF EXISTS resenas;
DROP TABLE IF EXISTS envios;
DROP TABLE IF EXISTS pagos;
DROP TABLE IF EXISTS pedido_items;
DROP TABLE IF EXISTS pedidos;
DROP TABLE IF EXISTS cupones;
DROP TABLE IF EXISTS precios_cliente;
DROP TABLE IF EXISTS direcciones;
DROP TABLE IF EXISTS clientes;
DROP TABLE IF EXISTS proveedores;
DROP TABLE IF EXISTS movimientos_stock;
DROP TABLE IF EXISTS stock_bodega;
DROP TABLE IF EXISTS bodegas;
DROP TABLE IF EXISTS productos;
DROP TABLE IF EXISTS categorias;
DROP TABLE IF EXISTS usuarios;
DROP TABLE IF EXISTS rol_permisos;
DROP TABLE IF EXISTS permisos;
DROP TABLE IF EXISTS roles;
`
