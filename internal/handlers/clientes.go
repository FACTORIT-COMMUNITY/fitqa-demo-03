package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
)

type ClientesHandler struct {
	db *sql.DB
}

func NewClientesHandler(db *sql.DB) *ClientesHandler {
	return &ClientesHandler{db: db}
}

type cliente struct {
	ID       int    `json:"id"`
	Nombre   string `json:"nombre"`
	Email    string `json:"email"`
	Activo   bool   `json:"activo"`
	CreadoEn string `json:"creado_en"`
}

// GET /clientes?activo=&limite=&desde=
func (h *ClientesHandler) List(w http.ResponseWriter, r *http.Request) {
	pag, ok := httpx.LeerPaginacion(r.URL.Query(), 20)
	if !ok {
		httpx.BadRequest(w, "paginacion invalida")
		return
	}
	where := "1=1"
	args := []any{}
	if v := r.URL.Query().Get("activo"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			httpx.BadRequest(w, "activo invalido", "use true o false")
			return
		}
		where += " AND activo = ?"
		args = append(args, boolAInt(b))
	}
	var total int
	_ = h.db.QueryRow("SELECT COUNT(*) FROM clientes WHERE "+where, args...).Scan(&total)

	args = append(args, pag.Limite, pag.Desde)
	rows, err := h.db.Query("SELECT id, nombre, email, activo, creado_en FROM clientes WHERE "+where+" ORDER BY id ASC LIMIT ? OFFSET ?", args...)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []cliente{}
	for rows.Next() {
		var c cliente
		var activo int
		if err := rows.Scan(&c.ID, &c.Nombre, &c.Email, &activo, &c.CreadoEn); err != nil {
			httpx.InternalError(w, err)
			return
		}
		c.Activo = activo != 0
		items = append(items, c)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: total, Items: items})
}

// GET /clientes/{id}
func (h *ClientesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	c, err := h.buscar(id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "cliente")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (h *ClientesHandler) buscar(id int) (cliente, error) {
	var c cliente
	var activo int
	err := h.db.QueryRow(`SELECT id, nombre, email, activo, creado_en FROM clientes WHERE id = ?`, id).Scan(&c.ID, &c.Nombre, &c.Email, &activo, &c.CreadoEn)
	c.Activo = activo != 0
	return c, err
}

type clienteBody struct {
	Nombre string `json:"nombre"`
	Email  string `json:"email"`
}

func (b clienteBody) validar() []string {
	var det []string
	if b.Nombre == "" {
		det = append(det, "nombre es requerido")
	}
	if b.Email == "" {
		det = append(det, "email es requerido")
	}
	return det
}

// POST /clientes
func (h *ClientesHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var body clienteBody
	if err := httpx.Decode(r, &body); err != nil {
		httpx.BadRequest(w, "cuerpo invalido")
		return
	}
	if det := body.validar(); len(det) > 0 {
		httpx.BadRequest(w, "datos invalidos", det...)
		return
	}
	res, err := h.db.Exec(`INSERT INTO clientes(nombre, email) VALUES (?, ?)`, body.Nombre, body.Email)
	if esViolacionUnicidad(err) {
		httpx.Conflict(w, "el email ya existe")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	id, _ := res.LastInsertId()
	c, _ := h.buscar(int(id))
	httpx.JSON(w, http.StatusCreated, c)
}

// PUT /clientes/{id}
func (h *ClientesHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "cliente")
		return
	}
	var body clienteBody
	if err := httpx.Decode(r, &body); err != nil {
		httpx.BadRequest(w, "cuerpo invalido")
		return
	}
	if det := body.validar(); len(det) > 0 {
		httpx.BadRequest(w, "datos invalidos", det...)
		return
	}
	_, err = h.db.Exec(`UPDATE clientes SET nombre = ?, email = ? WHERE id = ?`, body.Nombre, body.Email, id)
	if esViolacionUnicidad(err) {
		httpx.Conflict(w, "el email ya existe")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	c, _ := h.buscar(id)
	httpx.JSON(w, http.StatusOK, c)
}

// POST /clientes/{id}/dar-de-baja
func (h *ClientesHandler) DarDeBaja(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "cliente")
		return
	}
	if _, err := h.db.Exec(`UPDATE clientes SET activo = 0 WHERE id = ?`, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	c, _ := h.buscar(id)
	httpx.JSON(w, http.StatusOK, c)
}

// --- Direcciones (sub-recurso de cliente) ---

type direccion struct {
	ID           int    `json:"id"`
	ClienteID    int    `json:"cliente_id"`
	Calle        string `json:"calle"`
	Ciudad       string `json:"ciudad"`
	EsPrincipal  bool   `json:"es_principal"`
}

// GET /clientes/{id}/direcciones
func (h *ClientesHandler) ListDirecciones(w http.ResponseWriter, r *http.Request) {
	clienteID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(clienteID); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "cliente")
		return
	}
	rows, err := h.db.Query(`SELECT id, cliente_id, calle, ciudad, es_principal FROM direcciones WHERE cliente_id = ? ORDER BY id ASC`, clienteID)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []direccion{}
	for rows.Next() {
		var d direccion
		var principal int
		if err := rows.Scan(&d.ID, &d.ClienteID, &d.Calle, &d.Ciudad, &principal); err != nil {
			httpx.InternalError(w, err)
			return
		}
		d.EsPrincipal = principal != 0
		items = append(items, d)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: len(items), Items: items})
}

type direccionBody struct {
	Calle       string `json:"calle"`
	Ciudad      string `json:"ciudad"`
	EsPrincipal bool   `json:"es_principal"`
}

// POST /clientes/{id}/direcciones
func (h *ClientesHandler) CrearDireccion(w http.ResponseWriter, r *http.Request) {
	clienteID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(clienteID); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "cliente")
		return
	}
	var body direccionBody
	if err := httpx.Decode(r, &body); err != nil || body.Calle == "" || body.Ciudad == "" {
		httpx.BadRequest(w, "datos invalidos", "calle y ciudad son requeridos")
		return
	}
	tx, err := h.db.Begin()
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer tx.Rollback()
	if body.EsPrincipal {
		if _, err := tx.Exec(`UPDATE direcciones SET es_principal = 0 WHERE cliente_id = ?`, clienteID); err != nil {
			httpx.InternalError(w, err)
			return
		}
	}
	res, err := tx.Exec(
		`INSERT INTO direcciones(cliente_id, calle, ciudad, es_principal) VALUES (?, ?, ?, ?)`,
		clienteID, body.Calle, body.Ciudad, boolAInt(body.EsPrincipal),
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
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id, "cliente_id": clienteID, "calle": body.Calle, "ciudad": body.Ciudad, "es_principal": body.EsPrincipal})
}

// PUT /clientes/{id}/direcciones/{direccion_id}
func (h *ClientesHandler) ActualizarDireccion(w http.ResponseWriter, r *http.Request) {
	clienteID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	direccionID, err := strconv.Atoi(chi.URLParam(r, "direccion_id"))
	if err != nil {
		httpx.BadRequest(w, "direccion_id invalido")
		return
	}
	var existe int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM direcciones WHERE id = ? AND cliente_id = ?`, direccionID, clienteID).Scan(&existe)
	if existe == 0 {
		httpx.NotFound(w, "direccion")
		return
	}
	var body direccionBody
	if err := httpx.Decode(r, &body); err != nil || body.Calle == "" || body.Ciudad == "" {
		httpx.BadRequest(w, "datos invalidos", "calle y ciudad son requeridos")
		return
	}
	if _, err := h.db.Exec(
		`UPDATE direcciones SET calle = ?, ciudad = ?, es_principal = ? WHERE id = ?`,
		body.Calle, body.Ciudad, boolAInt(body.EsPrincipal), direccionID,
	); err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"id": direccionID, "cliente_id": clienteID, "calle": body.Calle, "ciudad": body.Ciudad, "es_principal": body.EsPrincipal})
}

// DELETE /clientes/{id}/direcciones/{direccion_id}
func (h *ClientesHandler) BorrarDireccion(w http.ResponseWriter, r *http.Request) {
	clienteID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	direccionID, err := strconv.Atoi(chi.URLParam(r, "direccion_id"))
	if err != nil {
		httpx.BadRequest(w, "direccion_id invalido")
		return
	}
	res, err := h.db.Exec(`DELETE FROM direcciones WHERE id = ? AND cliente_id = ?`, direccionID, clienteID)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		httpx.NotFound(w, "direccion")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}
