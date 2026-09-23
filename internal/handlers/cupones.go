package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
)

type CuponesHandler struct {
	db *sql.DB
}

func NewCuponesHandler(db *sql.DB) *CuponesHandler {
	return &CuponesHandler{db: db}
}

type cupon struct {
	ID     int    `json:"id"`
	Codigo string `json:"codigo"`
	Tipo   string `json:"tipo"`
	Valor  int    `json:"valor"`
	Activo bool   `json:"activo"`
}

var tiposCuponValidos = map[string]bool{"porcentaje": true, "fijo": true}

// GET /cupones?activo=&limite=&desde=
func (h *CuponesHandler) List(w http.ResponseWriter, r *http.Request) {
	pag, ok := httpx.LeerPaginacion(r.URL.Query(), 50)
	if !ok {
		httpx.BadRequest(w, "paginacion invalida")
		return
	}
	where := "1=1"
	args := []any{}
	if v := r.URL.Query().Get("activo"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			httpx.BadRequest(w, "activo invalido")
			return
		}
		where += " AND activo = ?"
		args = append(args, boolAInt(b))
	}
	var total int
	_ = h.db.QueryRow("SELECT COUNT(*) FROM cupones WHERE "+where, args...).Scan(&total)
	args = append(args, pag.Limite, pag.Desde)
	rows, err := h.db.Query("SELECT id, codigo, tipo, valor, activo FROM cupones WHERE "+where+" ORDER BY id ASC LIMIT ? OFFSET ?", args...)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []cupon{}
	for rows.Next() {
		var c cupon
		var activo int
		if err := rows.Scan(&c.ID, &c.Codigo, &c.Tipo, &c.Valor, &activo); err != nil {
			httpx.InternalError(w, err)
			return
		}
		c.Activo = activo != 0
		items = append(items, c)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: total, Items: items})
}

// GET /cupones/{id}
func (h *CuponesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	c, err := h.buscar(id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "cupon")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (h *CuponesHandler) buscar(id int) (cupon, error) {
	var c cupon
	var activo int
	err := h.db.QueryRow(`SELECT id, codigo, tipo, valor, activo FROM cupones WHERE id = ?`, id).Scan(&c.ID, &c.Codigo, &c.Tipo, &c.Valor, &activo)
	c.Activo = activo != 0
	return c, err
}

type cuponBody struct {
	Codigo string `json:"codigo"`
	Tipo   string `json:"tipo"`
	Valor  int    `json:"valor"`
}

func (b cuponBody) validar() []string {
	var det []string
	if b.Codigo == "" {
		det = append(det, "codigo es requerido")
	}
	if !tiposCuponValidos[b.Tipo] {
		det = append(det, "tipo debe ser 'porcentaje' o 'fijo'")
	}
	if b.Tipo == "porcentaje" && (b.Valor <= 0 || b.Valor > 100) {
		det = append(det, "valor de un cupon porcentaje debe estar entre 1 y 100")
	}
	if b.Tipo == "fijo" && b.Valor <= 0 {
		det = append(det, "valor de un cupon fijo debe ser mayor a 0")
	}
	return det
}

// POST /cupones
func (h *CuponesHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var body cuponBody
	if err := httpx.Decode(r, &body); err != nil {
		httpx.BadRequest(w, "cuerpo invalido")
		return
	}
	if det := body.validar(); len(det) > 0 {
		httpx.BadRequest(w, "datos invalidos", det...)
		return
	}
	res, err := h.db.Exec(`INSERT INTO cupones(codigo, tipo, valor, activo) VALUES (?, ?, ?, 1)`, body.Codigo, body.Tipo, body.Valor)
	if esViolacionUnicidad(err) {
		httpx.Conflict(w, "el codigo ya existe")
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

// PUT /cupones/{id}
func (h *CuponesHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "cupon")
		return
	}
	var body cuponBody
	if err := httpx.Decode(r, &body); err != nil {
		httpx.BadRequest(w, "cuerpo invalido")
		return
	}
	if det := body.validar(); len(det) > 0 {
		httpx.BadRequest(w, "datos invalidos", det...)
		return
	}
	_, err = h.db.Exec(`UPDATE cupones SET codigo = ?, tipo = ?, valor = ? WHERE id = ?`, body.Codigo, body.Tipo, body.Valor, id)
	if esViolacionUnicidad(err) {
		httpx.Conflict(w, "el codigo ya existe")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	c, _ := h.buscar(id)
	httpx.JSON(w, http.StatusOK, c)
}

// DELETE /cupones/{id}
func (h *CuponesHandler) Borrar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	c, err := h.buscar(id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "cupon")
		return
	}
	var enUso int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM pedidos WHERE cupon_codigo = ?`, c.Codigo).Scan(&enUso)
	if enUso > 0 {
		httpx.Conflict(w, "el cupon esta referenciado por pedidos y no se puede eliminar")
		return
	}
	if _, err := h.db.Exec(`DELETE FROM cupones WHERE id = ?`, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

type activarCuponBody struct {
	Activo bool `json:"activo"`
}

// POST /cupones/{id}/activar
func (h *CuponesHandler) Activar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "cupon")
		return
	}
	var body activarCuponBody
	if err := httpx.Decode(r, &body); err != nil {
		httpx.BadRequest(w, "cuerpo invalido", "se requiere activo (bool)")
		return
	}
	if _, err := h.db.Exec(`UPDATE cupones SET activo = ? WHERE id = ?`, boolAInt(body.Activo), id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	c, _ := h.buscar(id)
	httpx.JSON(w, http.StatusOK, c)
}

// GET /cupones/validar?codigo=...
func (h *CuponesHandler) Validar(w http.ResponseWriter, r *http.Request) {
	codigo := r.URL.Query().Get("codigo")
	if codigo == "" {
		httpx.BadRequest(w, "se requiere el parametro codigo")
		return
	}
	var c cupon
	var activo int
	err := h.db.QueryRow(`SELECT id, codigo, tipo, valor, activo FROM cupones WHERE codigo = ?`, codigo).Scan(&c.ID, &c.Codigo, &c.Tipo, &c.Valor, &activo)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.JSON(w, http.StatusOK, map[string]any{"valido": false, "motivo": "el codigo no existe"})
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	if activo == 0 {
		httpx.JSON(w, http.StatusOK, map[string]any{"valido": false, "motivo": "el cupon esta inactivo"})
		return
	}
	c.Activo = true
	httpx.JSON(w, http.StatusOK, map[string]any{"valido": true, "cupon": c})
}
