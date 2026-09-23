package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
)

type CategoriasHandler struct {
	db *sql.DB
}

func NewCategoriasHandler(db *sql.DB) *CategoriasHandler {
	return &CategoriasHandler{db: db}
}

type categoria struct {
	ID       int    `json:"id"`
	Nombre   string `json:"nombre"`
	Orden    int    `json:"orden"`
	CreadoEn string `json:"creado_en"`
}

// GET /categorias
func (h *CategoriasHandler) List(w http.ResponseWriter, r *http.Request) {
	pag, ok := httpx.LeerPaginacion(r.URL.Query(), 50)
	if !ok {
		httpx.BadRequest(w, "paginacion invalida")
		return
	}
	var total int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM categorias`).Scan(&total)

	rows, err := h.db.Query(`SELECT id, nombre, orden, creado_en FROM categorias ORDER BY orden ASC LIMIT ? OFFSET ?`, pag.Limite, pag.Desde)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []categoria{}
	for rows.Next() {
		var c categoria
		if err := rows.Scan(&c.ID, &c.Nombre, &c.Orden, &c.CreadoEn); err != nil {
			httpx.InternalError(w, err)
			return
		}
		items = append(items, c)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: total, Items: items})
}

// GET /categorias/{id}
func (h *CategoriasHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	c, err := h.buscar(id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "categoria")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (h *CategoriasHandler) buscar(id int) (categoria, error) {
	var c categoria
	err := h.db.QueryRow(`SELECT id, nombre, orden, creado_en FROM categorias WHERE id = ?`, id).Scan(&c.ID, &c.Nombre, &c.Orden, &c.CreadoEn)
	return c, err
}

type categoriaBody struct {
	Nombre string `json:"nombre"`
	Orden  int    `json:"orden"`
}

// POST /categorias
func (h *CategoriasHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var body categoriaBody
	if err := httpx.Decode(r, &body); err != nil || body.Nombre == "" {
		httpx.BadRequest(w, "datos invalidos", "nombre es requerido")
		return
	}
	res, err := h.db.Exec(`INSERT INTO categorias(nombre, orden) VALUES (?, ?)`, body.Nombre, body.Orden)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	id, _ := res.LastInsertId()
	c, _ := h.buscar(int(id))
	httpx.JSON(w, http.StatusCreated, c)
}

// PUT /categorias/{id}
func (h *CategoriasHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "categoria")
		return
	}
	var body categoriaBody
	if err := httpx.Decode(r, &body); err != nil || body.Nombre == "" {
		httpx.BadRequest(w, "datos invalidos", "nombre es requerido")
		return
	}
	if _, err := h.db.Exec(`UPDATE categorias SET nombre = ?, orden = ? WHERE id = ?`, body.Nombre, body.Orden, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	c, _ := h.buscar(id)
	httpx.JSON(w, http.StatusOK, c)
}

// DELETE /categorias/{id} — rechaza el borrado si tiene productos asociados.
func (h *CategoriasHandler) Borrar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "categoria")
		return
	}
	if _, err := h.db.Exec(`DELETE FROM categorias WHERE id = ?`, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

type reordenarBody struct {
	Orden []int `json:"orden"` // lista de ids de categoria en el nuevo orden
}

// POST /categorias/reordenar
func (h *CategoriasHandler) Reordenar(w http.ResponseWriter, r *http.Request) {
	var body reordenarBody
	if err := httpx.Decode(r, &body); err != nil || len(body.Orden) == 0 {
		httpx.BadRequest(w, "cuerpo invalido", "se requiere orden: lista de ids")
		return
	}
	tx, err := h.db.Begin()
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer tx.Rollback()
	for i, id := range body.Orden {
		if _, err := tx.Exec(`UPDATE categorias SET orden = ? WHERE id = ?`, i+1, id); err != nil {
			httpx.InternalError(w, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"estado": "reordenado"})
}
