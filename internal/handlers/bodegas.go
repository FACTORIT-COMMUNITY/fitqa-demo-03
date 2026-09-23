package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
)

type BodegasHandler struct {
	db *sql.DB
}

func NewBodegasHandler(db *sql.DB) *BodegasHandler {
	return &BodegasHandler{db: db}
}

type bodega struct {
	ID        int    `json:"id"`
	Nombre    string `json:"nombre"`
	Direccion string `json:"direccion"`
}

// GET /bodegas
func (h *BodegasHandler) List(w http.ResponseWriter, r *http.Request) {
	pag, ok := httpx.LeerPaginacion(r.URL.Query(), 50)
	if !ok {
		httpx.BadRequest(w, "paginacion invalida")
		return
	}
	var total int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM bodegas`).Scan(&total)
	rows, err := h.db.Query(`SELECT id, nombre, direccion FROM bodegas ORDER BY id ASC LIMIT ? OFFSET ?`, pag.Limite, pag.Desde)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []bodega{}
	for rows.Next() {
		var b bodega
		if err := rows.Scan(&b.ID, &b.Nombre, &b.Direccion); err != nil {
			httpx.InternalError(w, err)
			return
		}
		items = append(items, b)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: total, Items: items})
}

// GET /bodegas/{id}
func (h *BodegasHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	b, err := h.buscar(id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "bodega")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, b)
}

func (h *BodegasHandler) buscar(id int) (bodega, error) {
	var b bodega
	err := h.db.QueryRow(`SELECT id, nombre, direccion FROM bodegas WHERE id = ?`, id).Scan(&b.ID, &b.Nombre, &b.Direccion)
	return b, err
}

type bodegaBody struct {
	Nombre    string `json:"nombre"`
	Direccion string `json:"direccion"`
}

// POST /bodegas
func (h *BodegasHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var body bodegaBody
	if err := httpx.Decode(r, &body); err != nil || body.Nombre == "" || body.Direccion == "" {
		httpx.BadRequest(w, "datos invalidos", "nombre y direccion son requeridos")
		return
	}
	res, err := h.db.Exec(`INSERT INTO bodegas(nombre, direccion) VALUES (?, ?)`, body.Nombre, body.Direccion)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	id, _ := res.LastInsertId()
	b, _ := h.buscar(int(id))
	httpx.JSON(w, http.StatusCreated, b)
}

// PUT /bodegas/{id}
func (h *BodegasHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "bodega")
		return
	}
	var body bodegaBody
	if err := httpx.Decode(r, &body); err != nil || body.Nombre == "" || body.Direccion == "" {
		httpx.BadRequest(w, "datos invalidos", "nombre y direccion son requeridos")
		return
	}
	if _, err := h.db.Exec(`UPDATE bodegas SET nombre = ?, direccion = ? WHERE id = ?`, body.Nombre, body.Direccion, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	b, _ := h.buscar(id)
	httpx.JSON(w, http.StatusOK, b)
}

// DELETE /bodegas/{id}
func (h *BodegasHandler) Borrar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "bodega")
		return
	}
	var conStock int
	_ = h.db.QueryRow(`SELECT COALESCE(SUM(cantidad), 0) FROM stock_bodega WHERE bodega_id = ?`, id).Scan(&conStock)
	if conStock > 0 {
		httpx.Conflict(w, "la bodega tiene existencias y no se puede eliminar")
		return
	}
	if _, err := h.db.Exec(`DELETE FROM stock_bodega WHERE bodega_id = ?`, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	if _, err := h.db.Exec(`DELETE FROM bodegas WHERE id = ?`, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

type transferirStockBody struct {
	ProductoID        int    `json:"producto_id"`
	BodegaOrigenID    int    `json:"bodega_origen_id"`
	BodegaDestinoID   int    `json:"bodega_destino_id"`
	Cantidad          int    `json:"cantidad"`
	IdempotencyKey    string `json:"idempotency_key"`
}

// POST /bodegas/transferir-stock — mueve existencias de una bodega a otra. Es la unica
// escritura de este SUT que exige idempotency_key: sin ella, reintentar una transferencia
// (por un timeout de red, por ejemplo) la duplicaria.
func (h *BodegasHandler) TransferirStock(w http.ResponseWriter, r *http.Request) {
	var body transferirStockBody
	if err := httpx.Decode(r, &body); err != nil || body.Cantidad <= 0 || body.IdempotencyKey == "" || body.BodegaOrigenID == body.BodegaDestinoID {
		httpx.BadRequest(w, "datos invalidos", "se requiere producto_id, bodega_origen_id != bodega_destino_id, cantidad > 0 e idempotency_key")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer tx.Rollback()

	var origen int
	err = tx.QueryRow(`SELECT cantidad FROM stock_bodega WHERE producto_id = ? AND bodega_id = ?`, body.ProductoID, body.BodegaOrigenID).Scan(&origen)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.BadRequest(w, "el producto no tiene stock registrado en la bodega de origen")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	if origen < body.Cantidad {
		httpx.Conflict(w, "no hay suficiente stock en la bodega de origen")
		return
	}

	if _, err := tx.Exec(`UPDATE stock_bodega SET cantidad = cantidad - ? WHERE producto_id = ? AND bodega_id = ?`, body.Cantidad, body.ProductoID, body.BodegaOrigenID); err != nil {
		httpx.InternalError(w, err)
		return
	}
	// upsert manual: modernc.org/sqlite soporta ON CONFLICT, se usa directo.
	if _, err := tx.Exec(
		`INSERT INTO stock_bodega(producto_id, bodega_id, cantidad) VALUES (?, ?, ?)
		 ON CONFLICT(producto_id, bodega_id) DO UPDATE SET cantidad = cantidad + excluded.cantidad`,
		body.ProductoID, body.BodegaDestinoID, body.Cantidad,
	); err != nil {
		httpx.InternalError(w, err)
		return
	}
	if _, err := tx.Exec(
		`INSERT INTO movimientos_stock(producto_id, bodega_id, bodega_destino_id, tipo, cantidad, idempotency_key) VALUES (?, ?, ?, 'transferencia', ?, ?)`,
		body.ProductoID, body.BodegaOrigenID, body.BodegaDestinoID, body.Cantidad, body.IdempotencyKey,
	); err != nil {
		httpx.InternalError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"estado": "transferencia aplicada"})
}
