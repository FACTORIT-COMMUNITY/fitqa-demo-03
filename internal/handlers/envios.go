package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
)

type EnviosHandler struct {
	db *sql.DB
}

func NewEnviosHandler(db *sql.DB) *EnviosHandler {
	return &EnviosHandler{db: db}
}

type envio struct {
	ID            int    `json:"id"`
	PedidoID      int    `json:"pedido_id"`
	Estado        string `json:"estado"`
	Transportista string `json:"transportista"`
	Tracking      string `json:"tracking"`
	CreadoEn      string `json:"creado_en"`
}

// GET /envios?pedido_id&estado&limite&desde
func (h *EnviosHandler) List(w http.ResponseWriter, r *http.Request) {
	pag, ok := httpx.LeerPaginacion(r.URL.Query(), 20)
	if !ok {
		httpx.BadRequest(w, "paginacion invalida")
		return
	}
	where := "1=1"
	args := []any{}
	if v := r.URL.Query().Get("pedido_id"); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil {
			httpx.BadRequest(w, "pedido_id invalido")
			return
		}
		where += " AND pedido_id = ?"
		args = append(args, id)
	}
	if v := r.URL.Query().Get("estado"); v != "" {
		where += " AND estado = ?"
		args = append(args, v)
	}
	var total int
	_ = h.db.QueryRow("SELECT COUNT(*) FROM envios WHERE "+where, args...).Scan(&total)
	args = append(args, pag.Limite, pag.Desde)
	rows, err := h.db.Query("SELECT id, pedido_id, estado, COALESCE(transportista,''), COALESCE(tracking,''), creado_en FROM envios WHERE "+where+" ORDER BY id ASC LIMIT ? OFFSET ?", args...)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []envio{}
	for rows.Next() {
		var e envio
		if err := rows.Scan(&e.ID, &e.PedidoID, &e.Estado, &e.Transportista, &e.Tracking, &e.CreadoEn); err != nil {
			httpx.InternalError(w, err)
			return
		}
		items = append(items, e)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: total, Items: items})
}

// GET /envios/{id}
func (h *EnviosHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	e, err := h.buscar(id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "envio")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, e)
}

func (h *EnviosHandler) buscar(id int) (envio, error) {
	var e envio
	err := h.db.QueryRow(`SELECT id, pedido_id, estado, COALESCE(transportista,''), COALESCE(tracking,''), creado_en FROM envios WHERE id = ?`, id).
		Scan(&e.ID, &e.PedidoID, &e.Estado, &e.Transportista, &e.Tracking, &e.CreadoEn)
	return e, err
}

type crearEnvioBody struct {
	PedidoID      int    `json:"pedido_id"`
	Transportista string `json:"transportista"`
}

// POST /envios — un pedido confirmado sin envio previo puede generar uno.
func (h *EnviosHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var body crearEnvioBody
	if err := httpx.Decode(r, &body); err != nil || body.Transportista == "" {
		httpx.BadRequest(w, "datos invalidos", "se requiere pedido_id y transportista")
		return
	}
	var estadoPedido string
	err := h.db.QueryRow(`SELECT estado FROM pedidos WHERE id = ?`, body.PedidoID).Scan(&estadoPedido)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.BadRequest(w, "pedido_id no existe")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	if estadoPedido != "confirmado" {
		httpx.Conflict(w, "solo un pedido confirmado puede generar un envio")
		return
	}
	var yaTiene int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM envios WHERE pedido_id = ?`, body.PedidoID).Scan(&yaTiene)
	if yaTiene > 0 {
		httpx.Conflict(w, "el pedido ya tiene un envio")
		return
	}
	res, err := h.db.Exec(`INSERT INTO envios(pedido_id, estado, transportista) VALUES (?, 'pendiente', ?)`, body.PedidoID, body.Transportista)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	id, _ := res.LastInsertId()
	e, _ := h.buscar(int(id))
	httpx.JSON(w, http.StatusCreated, e)
}

var transicionesEnvioValidas = map[string]bool{"pendiente": true, "en_transito": true, "entregado": true}

type actualizarEnvioBody struct {
	Estado   string `json:"estado"`
	Tracking string `json:"tracking"`
}

// POST /envios/{id}/actualizar-estado
func (h *EnviosHandler) ActualizarEstado(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	e, err := h.buscar(id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "envio")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	var body actualizarEnvioBody
	if err := httpx.Decode(r, &body); err != nil || !transicionesEnvioValidas[body.Estado] {
		httpx.BadRequest(w, "estado invalido", "valores aceptados: pendiente, en_transito, entregado")
		return
	}
	if e.Estado == "entregado" {
		httpx.Conflict(w, "un envio entregado no cambia de estado")
		return
	}
	if body.Estado == "en_transito" && body.Tracking == "" {
		httpx.BadRequest(w, "datos invalidos", "se requiere tracking para pasar a en_transito")
		return
	}
	if _, err := h.db.Exec(`UPDATE envios SET estado = ?, tracking = COALESCE(NULLIF(?, ''), tracking) WHERE id = ?`, body.Estado, body.Tracking, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	e, _ = h.buscar(id)
	httpx.JSON(w, http.StatusOK, e)
}

// POST /envios/{id}/marcar-entregado
func (h *EnviosHandler) MarcarEntregado(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	e, err := h.buscar(id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "envio")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	if e.Estado != "en_transito" {
		httpx.Conflict(w, "solo un envio en transito se puede marcar como entregado")
		return
	}
	if _, err := h.db.Exec(`UPDATE envios SET estado = 'entregado' WHERE id = ?`, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	e, _ = h.buscar(id)
	httpx.JSON(w, http.StatusOK, e)
}
