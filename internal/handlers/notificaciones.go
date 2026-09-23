package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/auth"
	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
)

type NotificacionesHandler struct {
	db *sql.DB
}

func NewNotificacionesHandler(db *sql.DB) *NotificacionesHandler {
	return &NotificacionesHandler{db: db}
}

type notificacion struct {
	ID        int    `json:"id"`
	UsuarioID int    `json:"usuario_id"`
	Mensaje   string `json:"mensaje"`
	Leida     bool   `json:"leida"`
	CreadoEn  string `json:"creado_en"`
}

// GET /notificaciones — las del usuario autenticado.
func (h *NotificacionesHandler) List(w http.ResponseWriter, r *http.Request) {
	pag, ok := httpx.LeerPaginacion(r.URL.Query(), 20)
	if !ok {
		httpx.BadRequest(w, "paginacion invalida")
		return
	}
	usuarioID := auth.UsuarioID(r)
	var total int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM notificaciones WHERE usuario_id = ?`, usuarioID).Scan(&total)
	rows, err := h.db.Query(
		`SELECT id, usuario_id, mensaje, leida, creado_en FROM notificaciones WHERE usuario_id = ? ORDER BY id DESC LIMIT ? OFFSET ?`,
		usuarioID, pag.Limite, pag.Desde,
	)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []notificacion{}
	for rows.Next() {
		var n notificacion
		var leida int
		if err := rows.Scan(&n.ID, &n.UsuarioID, &n.Mensaje, &leida, &n.CreadoEn); err != nil {
			httpx.InternalError(w, err)
			return
		}
		n.Leida = leida != 0
		items = append(items, n)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: total, Items: items})
}

// POST /notificaciones/{id}/marcar-leida
func (h *NotificacionesHandler) MarcarLeida(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	res, err := h.db.Exec(`UPDATE notificaciones SET leida = 1 WHERE id = ? AND usuario_id = ?`, id, auth.UsuarioID(r))
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		httpx.NotFound(w, "notificacion")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"estado": "marcada como leida"})
}
