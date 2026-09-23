package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
)

type AuditoriaHandler struct {
	db *sql.DB
}

func NewAuditoriaHandler(db *sql.DB) *AuditoriaHandler {
	return &AuditoriaHandler{db: db}
}

type entradaAuditoria struct {
	ID         int    `json:"id"`
	UsuarioID  *int   `json:"usuario_id"`
	Accion     string `json:"accion"`
	Recurso    string `json:"recurso"`
	RecursoID  *int   `json:"recurso_id"`
	CreadoEn   string `json:"creado_en"`
}

func escanearAuditoria(rows *sql.Rows) (entradaAuditoria, error) {
	var e entradaAuditoria
	var usuarioID, recursoID sql.NullInt64
	err := rows.Scan(&e.ID, &usuarioID, &e.Accion, &e.Recurso, &recursoID, &e.CreadoEn)
	if usuarioID.Valid {
		v := int(usuarioID.Int64)
		e.UsuarioID = &v
	}
	if recursoID.Valid {
		v := int(recursoID.Int64)
		e.RecursoID = &v
	}
	return e, err
}

// GET /auditoria?limite&desde
func (h *AuditoriaHandler) List(w http.ResponseWriter, r *http.Request) {
	pag, ok := httpx.LeerPaginacion(r.URL.Query(), 20)
	if !ok {
		httpx.BadRequest(w, "paginacion invalida")
		return
	}
	var total int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM auditoria`).Scan(&total)
	rows, err := h.db.Query(`SELECT id, usuario_id, accion, recurso, recurso_id, creado_en FROM auditoria ORDER BY id DESC LIMIT ? OFFSET ?`, pag.Limite, pag.Desde)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []entradaAuditoria{}
	for rows.Next() {
		e, err := escanearAuditoria(rows)
		if err != nil {
			httpx.InternalError(w, err)
			return
		}
		items = append(items, e)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: total, Items: items})
}

// GET /auditoria/recurso/{recurso}/{recurso_id}
func (h *AuditoriaHandler) GetPorRecurso(w http.ResponseWriter, r *http.Request) {
	recurso := chi.URLParam(r, "recurso")
	recursoID, err := strconv.Atoi(chi.URLParam(r, "recurso_id"))
	if err != nil {
		httpx.BadRequest(w, "recurso_id invalido")
		return
	}
	rows, err := h.db.Query(
		`SELECT id, usuario_id, accion, recurso, recurso_id, creado_en FROM auditoria WHERE recurso = ? AND recurso_id = ? ORDER BY id DESC`,
		recurso, recursoID,
	)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []entradaAuditoria{}
	for rows.Next() {
		e, err := escanearAuditoria(rows)
		if err != nil {
			httpx.InternalError(w, err)
			return
		}
		items = append(items, e)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: len(items), Items: items})
}
