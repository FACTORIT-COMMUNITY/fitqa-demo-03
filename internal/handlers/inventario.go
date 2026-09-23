package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
)

type InventarioHandler struct {
	db *sql.DB
}

func NewInventarioHandler(db *sql.DB) *InventarioHandler {
	return &InventarioHandler{db: db}
}

type movimiento struct {
	ID               int    `json:"id"`
	ProductoID       int    `json:"producto_id"`
	BodegaID         int    `json:"bodega_id"`
	BodegaDestinoID  *int   `json:"bodega_destino_id,omitempty"`
	Tipo             string `json:"tipo"`
	Cantidad         int    `json:"cantidad"`
	Referencia       string `json:"referencia"`
	CreadoEn         string `json:"creado_en"`
}

// GET /inventario/movimientos?producto_id&bodega_id&tipo&limite&desde
func (h *InventarioHandler) List(w http.ResponseWriter, r *http.Request) {
	pag, ok := httpx.LeerPaginacion(r.URL.Query(), 20)
	if !ok {
		httpx.BadRequest(w, "paginacion invalida")
		return
	}
	where := "1=1"
	args := []any{}
	if v := r.URL.Query().Get("producto_id"); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil {
			httpx.BadRequest(w, "producto_id invalido")
			return
		}
		where += " AND producto_id = ?"
		args = append(args, id)
	}
	if v := r.URL.Query().Get("bodega_id"); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil {
			httpx.BadRequest(w, "bodega_id invalido")
			return
		}
		where += " AND bodega_id = ?"
		args = append(args, id)
	}
	if v := r.URL.Query().Get("tipo"); v != "" {
		where += " AND tipo = ?"
		args = append(args, v)
	}
	var total int
	_ = h.db.QueryRow("SELECT COUNT(*) FROM movimientos_stock WHERE "+where, args...).Scan(&total)
	args = append(args, pag.Limite, pag.Desde)
	rows, err := h.db.Query(
		"SELECT id, producto_id, bodega_id, bodega_destino_id, tipo, cantidad, COALESCE(referencia,''), creado_en FROM movimientos_stock WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		args...,
	)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []movimiento{}
	for rows.Next() {
		var m movimiento
		var destino sql.NullInt64
		if err := rows.Scan(&m.ID, &m.ProductoID, &m.BodegaID, &destino, &m.Tipo, &m.Cantidad, &m.Referencia, &m.CreadoEn); err != nil {
			httpx.InternalError(w, err)
			return
		}
		if destino.Valid {
			d := int(destino.Int64)
			m.BodegaDestinoID = &d
		}
		items = append(items, m)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: total, Items: items})
}

type crearMovimientoBody struct {
	ProductoID int    `json:"producto_id"`
	BodegaID   int    `json:"bodega_id"`
	Tipo       string `json:"tipo"` // entrada | salida
	Cantidad   int    `json:"cantidad"`
	Referencia string `json:"referencia"`
}

var tiposMovimientoDirectos = map[string]bool{"entrada": true, "salida": true}

// POST /inventario/movimientos — entrada o salida de stock en una sola bodega. Para
// mover entre bodegas existe /bodegas/transferir-stock.
func (h *InventarioHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var body crearMovimientoBody
	if err := httpx.Decode(r, &body); err != nil || !tiposMovimientoDirectos[body.Tipo] || body.Cantidad <= 0 {
		httpx.BadRequest(w, "datos invalidos", "tipo debe ser 'entrada' o 'salida', cantidad > 0")
		return
	}
	var actual int
	err := h.db.QueryRow(`SELECT cantidad FROM stock_bodega WHERE producto_id = ? AND bodega_id = ?`, body.ProductoID, body.BodegaID).Scan(&actual)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.BadRequest(w, "el producto no tiene stock registrado en esa bodega")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	delta := body.Cantidad
	if body.Tipo == "salida" {
		delta = -body.Cantidad
	}
	nuevo := actual + delta
	if nuevo < 0 {
		httpx.Conflict(w, "la salida dejaria el stock en negativo")
		return
	}
	tx, err := h.db.Begin()
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE stock_bodega SET cantidad = ? WHERE producto_id = ? AND bodega_id = ?`, nuevo, body.ProductoID, body.BodegaID); err != nil {
		httpx.InternalError(w, err)
		return
	}
	res, err := tx.Exec(
		`INSERT INTO movimientos_stock(producto_id, bodega_id, tipo, cantidad, referencia) VALUES (?, ?, ?, ?, ?)`,
		body.ProductoID, body.BodegaID, body.Tipo, body.Cantidad, body.Referencia,
	)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		httpx.InternalError(w, err)
		return
	}
	id, _ := res.LastInsertId()
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id, "producto_id": body.ProductoID, "bodega_id": body.BodegaID, "tipo": body.Tipo, "cantidad": body.Cantidad, "stock_resultante": nuevo})
}

// GET /inventario/productos/{producto_id}/historial
func (h *InventarioHandler) HistorialPorProducto(w http.ResponseWriter, r *http.Request) {
	productoID, err := strconv.Atoi(chi.URLParam(r, "producto_id"))
	if err != nil {
		httpx.BadRequest(w, "producto_id invalido")
		return
	}
	var existe int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM productos WHERE id = ?`, productoID).Scan(&existe)
	if existe == 0 {
		httpx.NotFound(w, "producto")
		return
	}
	rows, err := h.db.Query(
		`SELECT id, producto_id, bodega_id, bodega_destino_id, tipo, cantidad, COALESCE(referencia,''), creado_en
		 FROM movimientos_stock WHERE producto_id = ? ORDER BY id DESC`, productoID,
	)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []movimiento{}
	for rows.Next() {
		var m movimiento
		var destino sql.NullInt64
		if err := rows.Scan(&m.ID, &m.ProductoID, &m.BodegaID, &destino, &m.Tipo, &m.Cantidad, &m.Referencia, &m.CreadoEn); err != nil {
			httpx.InternalError(w, err)
			return
		}
		if destino.Valid {
			d := int(destino.Int64)
			m.BodegaDestinoID = &d
		}
		items = append(items, m)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: len(items), Items: items})
}
