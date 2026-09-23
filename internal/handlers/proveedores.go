package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
)

type ProveedoresHandler struct {
	db *sql.DB
}

func NewProveedoresHandler(db *sql.DB) *ProveedoresHandler {
	return &ProveedoresHandler{db: db}
}

type proveedor struct {
	ID       int    `json:"id"`
	Nombre   string `json:"nombre"`
	Email    string `json:"email"`
	Telefono string `json:"telefono"`
}

// GET /proveedores
func (h *ProveedoresHandler) List(w http.ResponseWriter, r *http.Request) {
	pag, ok := httpx.LeerPaginacion(r.URL.Query(), 50)
	if !ok {
		httpx.BadRequest(w, "paginacion invalida")
		return
	}
	var total int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM proveedores`).Scan(&total)
	rows, err := h.db.Query(`SELECT id, nombre, email, telefono FROM proveedores ORDER BY id ASC LIMIT ? OFFSET ?`, pag.Limite, pag.Desde)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []proveedor{}
	for rows.Next() {
		var p proveedor
		if err := rows.Scan(&p.ID, &p.Nombre, &p.Email, &p.Telefono); err != nil {
			httpx.InternalError(w, err)
			return
		}
		items = append(items, p)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: total, Items: items})
}

// GET /proveedores/{id}
func (h *ProveedoresHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	p, err := h.buscar(id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "proveedor")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *ProveedoresHandler) buscar(id int) (proveedor, error) {
	var p proveedor
	err := h.db.QueryRow(`SELECT id, nombre, email, telefono FROM proveedores WHERE id = ?`, id).Scan(&p.ID, &p.Nombre, &p.Email, &p.Telefono)
	return p, err
}

type proveedorBody struct {
	Nombre   string `json:"nombre"`
	Email    string `json:"email"`
	Telefono string `json:"telefono"`
}

func (b proveedorBody) validar() []string {
	var det []string
	if b.Nombre == "" {
		det = append(det, "nombre es requerido")
	}
	if b.Email == "" {
		det = append(det, "email es requerido")
	}
	return det
}

// POST /proveedores
func (h *ProveedoresHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var body proveedorBody
	if err := httpx.Decode(r, &body); err != nil {
		httpx.BadRequest(w, "cuerpo invalido")
		return
	}
	if det := body.validar(); len(det) > 0 {
		httpx.BadRequest(w, "datos invalidos", det...)
		return
	}
	res, err := h.db.Exec(`INSERT INTO proveedores(nombre, email, telefono) VALUES (?, ?, ?)`, body.Nombre, body.Email, body.Telefono)
	if esViolacionUnicidad(err) {
		httpx.Conflict(w, "el email ya existe")
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

// PUT /proveedores/{id}
func (h *ProveedoresHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "proveedor")
		return
	}
	var body proveedorBody
	if err := httpx.Decode(r, &body); err != nil {
		httpx.BadRequest(w, "cuerpo invalido")
		return
	}
	if det := body.validar(); len(det) > 0 {
		httpx.BadRequest(w, "datos invalidos", det...)
		return
	}
	_, err = h.db.Exec(`UPDATE proveedores SET nombre = ?, email = ?, telefono = ? WHERE id = ?`, body.Nombre, body.Email, body.Telefono, id)
	if esViolacionUnicidad(err) {
		httpx.Conflict(w, "el email ya existe")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	p, _ := h.buscar(id)
	httpx.JSON(w, http.StatusOK, p)
}

// DELETE /proveedores/{id}
func (h *ProveedoresHandler) Borrar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "proveedor")
		return
	}
	if _, err := h.db.Exec(`DELETE FROM proveedores WHERE id = ?`, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}
