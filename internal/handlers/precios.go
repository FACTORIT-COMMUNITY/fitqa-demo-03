package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
)

type PreciosHandler struct {
	db *sql.DB
}

func NewPreciosHandler(db *sql.DB) *PreciosHandler {
	return &PreciosHandler{db: db}
}

type precioCliente struct {
	ID             int `json:"id"`
	ClienteID      int `json:"cliente_id"`
	ProductoID     int `json:"producto_id"`
	PrecioCentavos int `json:"precio_centavos"`
}

// GET /precios-cliente?cliente_id&producto_id&limite&desde
func (h *PreciosHandler) List(w http.ResponseWriter, r *http.Request) {
	pag, ok := httpx.LeerPaginacion(r.URL.Query(), 50)
	if !ok {
		httpx.BadRequest(w, "paginacion invalida")
		return
	}
	where := "1=1"
	args := []any{}
	if v := r.URL.Query().Get("cliente_id"); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil {
			httpx.BadRequest(w, "cliente_id invalido")
			return
		}
		where += " AND cliente_id = ?"
		args = append(args, id)
	}
	if v := r.URL.Query().Get("producto_id"); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil {
			httpx.BadRequest(w, "producto_id invalido")
			return
		}
		where += " AND producto_id = ?"
		args = append(args, id)
	}
	var total int
	_ = h.db.QueryRow("SELECT COUNT(*) FROM precios_cliente WHERE "+where, args...).Scan(&total)
	args = append(args, pag.Limite, pag.Desde)
	rows, err := h.db.Query("SELECT id, cliente_id, producto_id, precio_centavos FROM precios_cliente WHERE "+where+" ORDER BY id ASC LIMIT ? OFFSET ?", args...)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []precioCliente{}
	for rows.Next() {
		var p precioCliente
		if err := rows.Scan(&p.ID, &p.ClienteID, &p.ProductoID, &p.PrecioCentavos); err != nil {
			httpx.InternalError(w, err)
			return
		}
		items = append(items, p)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: total, Items: items})
}

// GET /precios-cliente/{id}
func (h *PreciosHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	p, err := h.buscar(id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "precio de cliente")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *PreciosHandler) buscar(id int) (precioCliente, error) {
	var p precioCliente
	err := h.db.QueryRow(`SELECT id, cliente_id, producto_id, precio_centavos FROM precios_cliente WHERE id = ?`, id).Scan(&p.ID, &p.ClienteID, &p.ProductoID, &p.PrecioCentavos)
	return p, err
}

type precioClienteBody struct {
	ClienteID      int `json:"cliente_id"`
	ProductoID     int `json:"producto_id"`
	PrecioCentavos int `json:"precio_centavos"`
}

func (b precioClienteBody) validar() []string {
	var det []string
	if b.ClienteID == 0 {
		det = append(det, "cliente_id es requerido")
	}
	if b.ProductoID == 0 {
		det = append(det, "producto_id es requerido")
	}
	if b.PrecioCentavos <= 0 {
		det = append(det, "precio_centavos debe ser mayor a 0")
	}
	return det
}

// POST /precios-cliente
func (h *PreciosHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var body precioClienteBody
	if err := httpx.Decode(r, &body); err != nil {
		httpx.BadRequest(w, "cuerpo invalido")
		return
	}
	if det := body.validar(); len(det) > 0 {
		httpx.BadRequest(w, "datos invalidos", det...)
		return
	}
	res, err := h.db.Exec(`INSERT INTO precios_cliente(cliente_id, producto_id, precio_centavos) VALUES (?, ?, ?)`, body.ClienteID, body.ProductoID, body.PrecioCentavos)
	if esViolacionUnicidad(err) {
		httpx.Conflict(w, "ya existe un precio para ese cliente y producto")
		return
	}
	if esViolacionFK(err) {
		httpx.BadRequest(w, "cliente_id o producto_id no existen")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	id, _ := res.LastInsertId()
	p, _ := h.buscar(int(id))
	httpx.JSON(w, http.StatusCreated, p)
}

// PUT /precios-cliente/{id}
func (h *PreciosHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "precio de cliente")
		return
	}
	var body precioClienteBody
	if err := httpx.Decode(r, &body); err != nil || body.PrecioCentavos <= 0 {
		httpx.BadRequest(w, "datos invalidos", "precio_centavos debe ser mayor a 0")
		return
	}
	if _, err := h.db.Exec(`UPDATE precios_cliente SET precio_centavos = ? WHERE id = ?`, body.PrecioCentavos, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	p, _ := h.buscar(id)
	httpx.JSON(w, http.StatusOK, p)
}

// DELETE /precios-cliente/{id}
func (h *PreciosHandler) Borrar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "precio de cliente")
		return
	}
	if _, err := h.db.Exec(`DELETE FROM precios_cliente WHERE id = ?`, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}
