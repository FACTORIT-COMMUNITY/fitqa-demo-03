package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
)

type ResenasHandler struct {
	db *sql.DB
}

func NewResenasHandler(db *sql.DB) *ResenasHandler {
	return &ResenasHandler{db: db}
}

type resena struct {
	ID           int    `json:"id"`
	ProductoID   int    `json:"producto_id"`
	ClienteID    int    `json:"cliente_id"`
	Calificacion int    `json:"calificacion"`
	Comentario   string `json:"comentario"`
	Estado       string `json:"estado"`
	CreadoEn     string `json:"creado_en"`
}

// GET /resenas?producto_id&estado&limite&desde
func (h *ResenasHandler) List(w http.ResponseWriter, r *http.Request) {
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
	if v := r.URL.Query().Get("estado"); v != "" {
		where += " AND estado = ?"
		args = append(args, v)
	}
	var total int
	_ = h.db.QueryRow("SELECT COUNT(*) FROM resenas WHERE "+where, args...).Scan(&total)
	args = append(args, pag.Limite, pag.Desde)
	rows, err := h.db.Query("SELECT id, producto_id, cliente_id, calificacion, COALESCE(comentario,''), estado, creado_en FROM resenas WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?", args...)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []resena{}
	for rows.Next() {
		var res resena
		if err := rows.Scan(&res.ID, &res.ProductoID, &res.ClienteID, &res.Calificacion, &res.Comentario, &res.Estado, &res.CreadoEn); err != nil {
			httpx.InternalError(w, err)
			return
		}
		items = append(items, res)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: total, Items: items})
}

func (h *ResenasHandler) buscar(id int) (resena, error) {
	var res resena
	err := h.db.QueryRow(`SELECT id, producto_id, cliente_id, calificacion, COALESCE(comentario,''), estado, creado_en FROM resenas WHERE id = ?`, id).
		Scan(&res.ID, &res.ProductoID, &res.ClienteID, &res.Calificacion, &res.Comentario, &res.Estado, &res.CreadoEn)
	return res, err
}

type resenaBody struct {
	ProductoID   int    `json:"producto_id"`
	ClienteID    int    `json:"cliente_id"`
	Calificacion int    `json:"calificacion"`
	Comentario   string `json:"comentario"`
}

// POST /resenas
func (h *ResenasHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var body resenaBody
	if err := httpx.Decode(r, &body); err != nil || body.Calificacion < 1 || body.Calificacion > 5 {
		httpx.BadRequest(w, "datos invalidos", "calificacion debe estar entre 1 y 5")
		return
	}
	res, err := h.db.Exec(
		`INSERT INTO resenas(producto_id, cliente_id, calificacion, comentario, estado) VALUES (?, ?, ?, ?, 'pendiente')`,
		body.ProductoID, body.ClienteID, body.Calificacion, body.Comentario,
	)
	if esViolacionFK(err) {
		httpx.BadRequest(w, "producto_id o cliente_id no existen")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	id, _ := res.LastInsertId()
	rr, _ := h.buscar(int(id))
	httpx.JSON(w, http.StatusCreated, rr)
}

type moderarResenaBody struct {
	Estado string `json:"estado"` // aprobada | rechazada
}

// POST /resenas/{id}/moderar
func (h *ResenasHandler) Moderar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	rr, err := h.buscar(id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "resena")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	var body moderarResenaBody
	if err := httpx.Decode(r, &body); err != nil || (body.Estado != "aprobada" && body.Estado != "rechazada") {
		httpx.BadRequest(w, "estado invalido", "valores aceptados: aprobada, rechazada")
		return
	}
	if rr.Estado != "pendiente" {
		httpx.Conflict(w, "solo una resena pendiente se puede moderar")
		return
	}
	if _, err := h.db.Exec(`UPDATE resenas SET estado = ? WHERE id = ?`, body.Estado, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	rr, _ = h.buscar(id)
	httpx.JSON(w, http.StatusOK, rr)
}

// DELETE /resenas/{id}
func (h *ResenasHandler) Borrar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "resena")
		return
	}
	if _, err := h.db.Exec(`DELETE FROM resenas WHERE id = ?`, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}
