package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/auth"
	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
)

type PedidosHandler struct {
	db *sql.DB
}

func NewPedidosHandler(db *sql.DB) *PedidosHandler {
	return &PedidosHandler{db: db}
}

type itemPedido struct {
	ID                     int `json:"id"`
	ProductoID             int `json:"producto_id"`
	Cantidad               int `json:"cantidad"`
	PrecioUnitarioCentavos int `json:"precio_unitario_centavos"`
}

type pedido struct {
	ID                 int          `json:"id"`
	ClienteID          int          `json:"cliente_id"`
	VendedorID         int          `json:"vendedor_id"`
	Estado             string       `json:"estado"`
	SubtotalCentavos   int          `json:"subtotal_centavos"`
	DescuentoCentavos  int          `json:"descuento_centavos"`
	ImpuestosCentavos  int          `json:"impuestos_centavos"`
	TotalCentavos      int          `json:"total_centavos"`
	CuponCodigo        *string      `json:"cupon_codigo"`
	CreadoEn           string       `json:"creado_en"`
	Items              []itemPedido `json:"items"`
}

// GET /pedidos?limite&desde&estado&cliente_id&vendedor_id
func (h *PedidosHandler) List(w http.ResponseWriter, r *http.Request) {
	pag, ok := httpx.LeerPaginacion(r.URL.Query(), 20)
	if !ok {
		httpx.BadRequest(w, "paginacion invalida")
		return
	}
	where := "1=1"
	args := []any{}
	if v := r.URL.Query().Get("estado"); v != "" {
		where += " AND estado = ?"
		args = append(args, v)
	}
	if v := r.URL.Query().Get("cliente_id"); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil {
			httpx.BadRequest(w, "cliente_id invalido")
			return
		}
		where += " AND cliente_id = ?"
		args = append(args, id)
	}
	if v := r.URL.Query().Get("vendedor_id"); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil {
			httpx.BadRequest(w, "vendedor_id invalido")
			return
		}
		where += " AND vendedor_id = ?"
		args = append(args, id)
	}

	var total int
	_ = h.db.QueryRow("SELECT COUNT(*) FROM pedidos WHERE "+where, args...).Scan(&total)

	args = append(args, pag.Limite, pag.Desde+1)
	rows, err := h.db.Query(
		"SELECT id, cliente_id, vendedor_id, estado, subtotal_centavos, descuento_centavos, impuestos_centavos, total_centavos, cupon_codigo, creado_en FROM pedidos WHERE "+where+" ORDER BY id ASC LIMIT ? OFFSET ?",
		args...,
	)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []pedido{}
	for rows.Next() {
		p, err := escanearPedido(rows)
		if err != nil {
			httpx.InternalError(w, err)
			return
		}
		items = append(items, p)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: total, Items: items})
}

type fila interface {
	Scan(dest ...any) error
}

func escanearPedido(f fila) (pedido, error) {
	var p pedido
	var cupon sql.NullString
	err := f.Scan(&p.ID, &p.ClienteID, &p.VendedorID, &p.Estado, &p.SubtotalCentavos, &p.DescuentoCentavos, &p.ImpuestosCentavos, &p.TotalCentavos, &cupon, &p.CreadoEn)
	if cupon.Valid {
		p.CuponCodigo = &cupon.String
	}
	return p, err
}

// GET /pedidos/{id}
func (h *PedidosHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	p, err := h.buscar(id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "pedido")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *PedidosHandler) buscar(id int) (pedido, error) {
	row := h.db.QueryRow(
		`SELECT id, cliente_id, vendedor_id, estado, subtotal_centavos, descuento_centavos, impuestos_centavos, total_centavos, cupon_codigo, creado_en FROM pedidos WHERE id = ?`, id,
	)
	p, err := escanearPedido(row)
	if err != nil {
		return p, err
	}
	items, err := h.itemsDe(id)
	if err != nil {
		return p, err
	}
	p.Items = items
	return p, nil
}

func (h *PedidosHandler) itemsDe(pedidoID int) ([]itemPedido, error) {
	rows, err := h.db.Query(`SELECT id, producto_id, cantidad, precio_unitario_centavos FROM pedido_items WHERE pedido_id = ? ORDER BY id ASC`, pedidoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []itemPedido{}
	for rows.Next() {
		var it itemPedido
		if err := rows.Scan(&it.ID, &it.ProductoID, &it.Cantidad, &it.PrecioUnitarioCentavos); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

type itemBody struct {
	ProductoID int `json:"producto_id"`
	Cantidad   int `json:"cantidad"`
}

type crearPedidoBody struct {
	ClienteID   int        `json:"cliente_id"`
	CuponCodigo string     `json:"cupon_codigo"`
	Items       []itemBody `json:"items"`
}

// POST /pedidos
func (h *PedidosHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var body crearPedidoBody
	if err := httpx.Decode(r, &body); err != nil {
		httpx.BadRequest(w, "cuerpo invalido")
		return
	}
	if body.ClienteID == 0 || len(body.Items) == 0 {
		httpx.BadRequest(w, "datos invalidos", "se requiere cliente_id y al menos un item")
		return
	}
	var activo int
	err := h.db.QueryRow(`SELECT activo FROM clientes WHERE id = ?`, body.ClienteID).Scan(&activo)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.BadRequest(w, "cliente_id no existe")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	if activo == 0 {
		httpx.Conflict(w, "el cliente esta dado de baja y no puede generar pedidos")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`INSERT INTO pedidos(cliente_id, vendedor_id, estado) VALUES (?, ?, 'borrador')`,
		body.ClienteID, auth.UsuarioID(r),
	)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	pedidoID64, _ := res.LastInsertId()
	pedidoID := int(pedidoID64)

	for _, it := range body.Items {
		if it.Cantidad <= 0 {
			httpx.BadRequest(w, "cantidad debe ser mayor a 0")
			return
		}
		precio, err := precioParaCliente(tx, body.ClienteID, it.ProductoID)
		if errors.Is(err, sql.ErrNoRows) {
			httpx.BadRequest(w, "producto_id no existe", strconv.Itoa(it.ProductoID))
			return
		}
		if err != nil {
			httpx.InternalError(w, err)
			return
		}
		if _, err := tx.Exec(
			`INSERT INTO pedido_items(pedido_id, producto_id, cantidad, precio_unitario_centavos) VALUES (?, ?, ?, ?)`,
			pedidoID, it.ProductoID, it.Cantidad, precio,
		); err != nil {
			httpx.InternalError(w, err)
			return
		}
	}

	if body.CuponCodigo != "" {
		if err := aplicarCupon(tx, pedidoID, body.CuponCodigo); err != nil {
			httpx.Conflict(w, err.Error())
			return
		}
	}
	if err := recalcularTotales(tx, pedidoID); err != nil {
		httpx.InternalError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		httpx.InternalError(w, err)
		return
	}
	p, _ := h.buscar(pedidoID)
	httpx.JSON(w, http.StatusCreated, p)
}

// precioParaCliente resuelve el precio unitario: la lista de precios del cliente manda
// sobre el precio de catalogo si existe una entrada.
func precioParaCliente(tx *sql.Tx, clienteID, productoID int) (int, error) {
	var precio int
	err := tx.QueryRow(`SELECT precio_centavos FROM precios_cliente WHERE cliente_id = ? AND producto_id = ?`, clienteID, productoID).Scan(&precio)
	if err == nil {
		return precio, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	err = tx.QueryRow(`SELECT precio_centavos FROM productos WHERE id = ?`, productoID).Scan(&precio)
	return precio, err
}

const tasaImpuesto = 19 // IVA, en porcentaje entero

// recalcularTotales recomputa subtotal/descuento/impuestos/total de un pedido a partir
// de sus items y, si tiene, su cupon. El impuesto se calcula sobre (subtotal - descuento).
func recalcularTotales(tx *sql.Tx, pedidoID int) error {
	var subtotal int
	if err := tx.QueryRow(
		`SELECT COALESCE(SUM(cantidad * precio_unitario_centavos), 0) FROM pedido_items WHERE pedido_id = ?`, pedidoID,
	).Scan(&subtotal); err != nil {
		return err
	}

	var cuponCodigo sql.NullString
	if err := tx.QueryRow(`SELECT cupon_codigo FROM pedidos WHERE id = ?`, pedidoID).Scan(&cuponCodigo); err != nil {
		return err
	}

	descuento := 0
	if cuponCodigo.Valid {
		var tipo string
		var valor int
		if err := tx.QueryRow(`SELECT tipo, valor FROM cupones WHERE codigo = ?`, cuponCodigo.String).Scan(&tipo, &valor); err != nil {
			return err
		}
		if tipo == "porcentaje" {
			descuento = subtotal * valor / 100
		} else {
			descuento = valor
		}
		if descuento > subtotal {
			descuento = subtotal
		}
	}

	impuestos := subtotal * tasaImpuesto / 100
	total := subtotal - descuento + impuestos

	_, err := tx.Exec(
		`UPDATE pedidos SET subtotal_centavos = ?, descuento_centavos = ?, impuestos_centavos = ?, total_centavos = ? WHERE id = ?`,
		subtotal, descuento, impuestos, total, pedidoID,
	)
	return err
}

func aplicarCupon(tx *sql.Tx, pedidoID int, codigo string) error {
	var activo int
	err := tx.QueryRow(`SELECT activo FROM cupones WHERE codigo = ?`, codigo).Scan(&activo)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("el cupon no existe")
	}
	if err != nil {
		return err
	}
	if activo == 0 {
		return errors.New("el cupon esta inactivo")
	}
	_, err = tx.Exec(`UPDATE pedidos SET cupon_codigo = ? WHERE id = ?`, codigo, pedidoID)
	return err
}

// --- Items del pedido ---

// GET /pedidos/{id}/items
func (h *PedidosHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	pedidoID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscarSinItems(pedidoID); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "pedido")
		return
	}
	items, err := h.itemsDe(pedidoID)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: len(items), Items: items})
}

func (h *PedidosHandler) buscarSinItems(id int) (pedido, error) {
	row := h.db.QueryRow(
		`SELECT id, cliente_id, vendedor_id, estado, subtotal_centavos, descuento_centavos, impuestos_centavos, total_centavos, cupon_codigo, creado_en FROM pedidos WHERE id = ?`, id,
	)
	return escanearPedido(row)
}

// POST /pedidos/{id}/items — solo mientras el pedido esta en borrador.
func (h *PedidosHandler) AgregarItem(w http.ResponseWriter, r *http.Request) {
	pedidoID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	p, err := h.buscarSinItems(pedidoID)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "pedido")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	if p.Estado != "borrador" {
		httpx.Conflict(w, "solo se pueden agregar items a un pedido en borrador")
		return
	}
	var body itemBody
	if err := httpx.Decode(r, &body); err != nil || body.Cantidad <= 0 {
		httpx.BadRequest(w, "datos invalidos", "se requiere producto_id y cantidad > 0")
		return
	}
	tx, err := h.db.Begin()
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer tx.Rollback()
	precio, err := precioParaCliente(tx, p.ClienteID, body.ProductoID)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.BadRequest(w, "producto_id no existe")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	if _, err := tx.Exec(
		`INSERT INTO pedido_items(pedido_id, producto_id, cantidad, precio_unitario_centavos) VALUES (?, ?, ?, ?)`,
		pedidoID, body.ProductoID, body.Cantidad, precio,
	); err != nil {
		httpx.InternalError(w, err)
		return
	}
	if err := recalcularTotales(tx, pedidoID); err != nil {
		httpx.InternalError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		httpx.InternalError(w, err)
		return
	}
	pActualizado, _ := h.buscar(pedidoID)
	httpx.JSON(w, http.StatusCreated, pActualizado)
}

// DELETE /pedidos/{id}/items/{item_id} — solo mientras el pedido esta en borrador.
func (h *PedidosHandler) QuitarItem(w http.ResponseWriter, r *http.Request) {
	pedidoID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	itemID, err := strconv.Atoi(chi.URLParam(r, "item_id"))
	if err != nil {
		httpx.BadRequest(w, "item_id invalido")
		return
	}
	p, err := h.buscarSinItems(pedidoID)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "pedido")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	if p.Estado != "borrador" {
		httpx.Conflict(w, "solo se pueden quitar items de un pedido en borrador")
		return
	}
	tx, err := h.db.Begin()
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec(`DELETE FROM pedido_items WHERE id = ? AND pedido_id = ?`, itemID, pedidoID)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		httpx.NotFound(w, "item")
		return
	}
	if err := recalcularTotales(tx, pedidoID); err != nil {
		httpx.InternalError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

// --- Transiciones de estado ---

type actualizarEstadoBody struct {
	Estado string `json:"estado"`
}

// POST /pedidos/{id}/actualizar-estado — hoy solo soporta borrador -> confirmado.
func (h *PedidosHandler) ActualizarEstado(w http.ResponseWriter, r *http.Request) {
	pedidoID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	p, err := h.buscarSinItems(pedidoID)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "pedido")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	var body actualizarEstadoBody
	if err := httpx.Decode(r, &body); err != nil {
		httpx.BadRequest(w, "cuerpo invalido")
		return
	}
	if body.Estado != "confirmado" {
		httpx.BadRequest(w, "transicion invalida", "desde /actualizar-estado solo se admite 'confirmado'; para cancelar o reembolsar use esos endpoints")
		return
	}
	if p.Estado != "borrador" {
		httpx.Conflict(w, "solo un pedido en borrador puede confirmarse")
		return
	}
	items, err := h.itemsDe(pedidoID)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	if len(items) == 0 {
		httpx.Conflict(w, "un pedido sin items no se puede confirmar")
		return
	}
	if _, err := h.db.Exec(`UPDATE pedidos SET estado = 'confirmado' WHERE id = ?`, pedidoID); err != nil {
		httpx.InternalError(w, err)
		return
	}
	pActualizado, _ := h.buscar(pedidoID)
	httpx.JSON(w, http.StatusOK, pActualizado)
}

// POST /pedidos/{id}/cancelar
func (h *PedidosHandler) Cancelar(w http.ResponseWriter, r *http.Request) {
	pedidoID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	p, err := h.buscarSinItems(pedidoID)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "pedido")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	if p.Estado != "borrador" && p.Estado != "confirmado" {
		httpx.Conflict(w, "el pedido no se puede cancelar en su estado actual")
		return
	}
	if _, err := h.db.Exec(`UPDATE pedidos SET estado = 'cancelado' WHERE id = ?`, pedidoID); err != nil {
		httpx.InternalError(w, err)
		return
	}
	pActualizado, _ := h.buscar(pedidoID)
	httpx.JSON(w, http.StatusOK, pActualizado)
}

// POST /pedidos/{id}/reembolsar — requiere que el pedido este confirmado y con un pago
// confirmado; anula el pago y marca el pedido como reembolsado.
func (h *PedidosHandler) Reembolsar(w http.ResponseWriter, r *http.Request) {
	pedidoID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	p, err := h.buscarSinItems(pedidoID)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "pedido")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	if p.Estado != "confirmado" {
		httpx.Conflict(w, "solo un pedido confirmado se puede reembolsar")
		return
	}
	var pagoID int
	err = h.db.QueryRow(`SELECT id FROM pagos WHERE pedido_id = ? AND estado = 'confirmado'`, pedidoID).Scan(&pagoID)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.Conflict(w, "el pedido no tiene un pago confirmado para reembolsar")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	tx, err := h.db.Begin()
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE pagos SET estado = 'anulado' WHERE id = ?`, pagoID); err != nil {
		httpx.InternalError(w, err)
		return
	}
	if _, err := tx.Exec(`UPDATE pedidos SET estado = 'reembolsado' WHERE id = ?`, pedidoID); err != nil {
		httpx.InternalError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		httpx.InternalError(w, err)
		return
	}
	pActualizado, _ := h.buscar(pedidoID)
	httpx.JSON(w, http.StatusOK, pActualizado)
}
