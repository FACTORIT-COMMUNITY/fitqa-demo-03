package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
)

type PagosHandler struct {
	db *sql.DB
}

func NewPagosHandler(db *sql.DB) *PagosHandler {
	return &PagosHandler{db: db}
}

type pago struct {
	ID             int    `json:"id"`
	PedidoID       int    `json:"pedido_id"`
	MontoCentavos  int    `json:"monto_centavos"`
	Estado         string `json:"estado"`
	Metodo         string `json:"metodo"`
	CreadoEn       string `json:"creado_en"`
}

// GET /pagos?pedido_id&estado&limite&desde
func (h *PagosHandler) List(w http.ResponseWriter, r *http.Request) {
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
	_ = h.db.QueryRow("SELECT COUNT(*) FROM pagos WHERE "+where, args...).Scan(&total)
	args = append(args, pag.Limite, pag.Desde)
	rows, err := h.db.Query("SELECT id, pedido_id, monto_centavos, estado, metodo, creado_en FROM pagos WHERE "+where+" ORDER BY id ASC LIMIT ? OFFSET ?", args...)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []pago{}
	for rows.Next() {
		var p pago
		if err := rows.Scan(&p.ID, &p.PedidoID, &p.MontoCentavos, &p.Estado, &p.Metodo, &p.CreadoEn); err != nil {
			httpx.InternalError(w, err)
			return
		}
		items = append(items, p)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: total, Items: items})
}

// GET /pagos/{id}
func (h *PagosHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	p, err := h.buscar(id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "pago")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *PagosHandler) buscar(id int) (pago, error) {
	var p pago
	err := h.db.QueryRow(`SELECT id, pedido_id, monto_centavos, estado, metodo, creado_en FROM pagos WHERE id = ?`, id).
		Scan(&p.ID, &p.PedidoID, &p.MontoCentavos, &p.Estado, &p.Metodo, &p.CreadoEn)
	return p, err
}

type crearPagoBody struct {
	PedidoID      int    `json:"pedido_id"`
	MontoCentavos int    `json:"monto_centavos"`
	Metodo        string `json:"metodo"`
}

// POST /pagos — registra un pago pendiente para un pedido confirmado que aun no tiene uno.
func (h *PagosHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var body crearPagoBody
	if err := httpx.Decode(r, &body); err != nil || body.MontoCentavos <= 0 || body.Metodo == "" {
		httpx.BadRequest(w, "datos invalidos", "se requiere pedido_id, monto_centavos > 0 y metodo")
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
		httpx.Conflict(w, "solo un pedido confirmado puede recibir un pago")
		return
	}
	var yaTiene int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM pagos WHERE pedido_id = ? AND estado != 'anulado'`, body.PedidoID).Scan(&yaTiene)
	if yaTiene > 0 {
		httpx.Conflict(w, "el pedido ya tiene un pago activo")
		return
	}
	res, err := h.db.Exec(
		`INSERT INTO pagos(pedido_id, monto_centavos, estado, metodo) VALUES (?, ?, 'pendiente', ?)`,
		body.PedidoID, body.MontoCentavos, body.Metodo,
	)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	id, _ := res.LastInsertId()
	p, _ := h.buscar(int(id))
	httpx.JSON(w, http.StatusCreated, p)
}

// POST /pagos/{id}/confirmar
func (h *PagosHandler) Confirmar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	p, err := h.buscar(id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "pago")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	if p.Estado != "pendiente" {
		httpx.Conflict(w, "solo un pago pendiente se puede confirmar")
		return
	}
	if _, err := h.db.Exec(`UPDATE pagos SET estado = 'confirmado' WHERE id = ?`, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	p, _ = h.buscar(id)
	httpx.JSON(w, http.StatusOK, p)
}

// POST /pagos/{id}/anular
func (h *PagosHandler) Anular(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	p, err := h.buscar(id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "pago")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	if p.Estado == "anulado" {
		httpx.Conflict(w, "el pago ya esta anulado")
		return
	}
	if _, err := h.db.Exec(`UPDATE pagos SET estado = 'anulado' WHERE id = ?`, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	p, _ = h.buscar(id)
	httpx.JSON(w, http.StatusOK, p)
}
