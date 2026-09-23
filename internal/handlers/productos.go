package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
)

type ProductosHandler struct {
	db *sql.DB
}

func NewProductosHandler(db *sql.DB) *ProductosHandler {
	return &ProductosHandler{db: db}
}

type producto struct {
	ID             int    `json:"id"`
	SKU            string `json:"sku"`
	Nombre         string `json:"nombre"`
	CategoriaID    int    `json:"categoria_id"`
	PrecioCentavos int    `json:"precio_centavos"`
	Publicado      bool   `json:"publicado"`
	StockTotal     int    `json:"stock_total"`
	CreadoEn       string `json:"creado_en"`
}

var ordenesProductoValidas = map[string]string{
	"id": "p.id", "nombre": "p.nombre", "precio": "p.precio_centavos",
}

// GET /productos?limite&desde&categoria_id&publicado&orden
func (h *ProductosHandler) List(w http.ResponseWriter, r *http.Request) {
	pag, ok := httpx.LeerPaginacion(r.URL.Query(), 20)
	if !ok {
		httpx.BadRequest(w, "paginacion invalida", "limite debe estar entre 1 y 100, desde >= 0")
		return
	}
	orden := r.URL.Query().Get("orden")
	if orden == "" {
		orden = "id"
	}
	columnaOrden, ok := ordenesProductoValidas[orden]
	if !ok {
		httpx.BadRequest(w, "orden invalido", "valores aceptados: id, nombre, precio")
		return
	}

	where := "1=1"
	args := []any{}
	if v := r.URL.Query().Get("categoria_id"); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil {
			httpx.BadRequest(w, "categoria_id invalido")
			return
		}
		where += " AND p.categoria_id = ?"
		args = append(args, id)
	}
	if v := r.URL.Query().Get("publicado"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			httpx.BadRequest(w, "publicado invalido", "use true o false")
			return
		}
		where += " AND p.publicado = ?"
		args = append(args, boolAInt(b))
	}

	var total int
	if err := h.db.QueryRow("SELECT COUNT(*) FROM productos p WHERE "+where, args...).Scan(&total); err != nil {
		httpx.InternalError(w, err)
		return
	}

	_ = columnaOrden
	consulta := `
		SELECT p.id, p.sku, p.nombre, p.categoria_id, p.precio_centavos, p.publicado, p.creado_en,
		       COALESCE((SELECT SUM(cantidad) FROM stock_bodega sb WHERE sb.producto_id = p.id), 0)
		FROM productos p WHERE ` + where + ` ORDER BY p.id ASC LIMIT ? OFFSET ?`
	args = append(args, pag.Limite, pag.Desde)

	rows, err := h.db.Query(consulta, args...)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()

	items := []producto{}
	for rows.Next() {
		var p producto
		var pub int
		if err := rows.Scan(&p.ID, &p.SKU, &p.Nombre, &p.CategoriaID, &p.PrecioCentavos, &pub, &p.CreadoEn, &p.StockTotal); err != nil {
			httpx.InternalError(w, err)
			return
		}
		p.Publicado = pub != 0
		items = append(items, p)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: total, Items: items})
}

// GET /productos/{id}
func (h *ProductosHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	p, err := h.buscar(id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "producto")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *ProductosHandler) buscar(id int) (producto, error) {
	var p producto
	var pub int
	err := h.db.QueryRow(`
		SELECT p.id, p.sku, p.nombre, p.categoria_id, p.precio_centavos, p.publicado, p.creado_en,
		       COALESCE((SELECT SUM(cantidad) FROM stock_bodega sb WHERE sb.producto_id = p.id), 0)
		FROM productos p WHERE p.id = ?`, id,
	).Scan(&p.ID, &p.SKU, &p.Nombre, &p.CategoriaID, &p.PrecioCentavos, &pub, &p.CreadoEn, &p.StockTotal)
	p.Publicado = pub != 0
	return p, err
}

type productoBody struct {
	SKU            string `json:"sku"`
	Nombre         string `json:"nombre"`
	CategoriaID    int    `json:"categoria_id"`
	PrecioCentavos int    `json:"precio_centavos"`
}

func (b productoBody) validar() []string {
	var det []string
	if b.SKU == "" {
		det = append(det, "sku es requerido")
	}
	if b.Nombre == "" {
		det = append(det, "nombre es requerido")
	}
	if b.PrecioCentavos <= 0 {
		det = append(det, "precio_centavos debe ser mayor a 0")
	}
	return det
}

// POST /productos
func (h *ProductosHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var body productoBody
	if err := httpx.Decode(r, &body); err != nil {
		httpx.BadRequest(w, "cuerpo invalido")
		return
	}
	if det := body.validar(); len(det) > 0 {
		httpx.BadRequest(w, "datos invalidos", det...)
		return
	}
	if !h.categoriaExiste(body.CategoriaID) {
		httpx.BadRequest(w, "categoria_id no existe")
		return
	}
	res, err := h.db.Exec(
		`INSERT INTO productos(sku, nombre, categoria_id, precio_centavos, publicado) VALUES (?, ?, ?, ?, 0)`,
		body.SKU, body.Nombre, body.CategoriaID, body.PrecioCentavos,
	)
	if esViolacionUnicidad(err) {
		httpx.Conflict(w, "el sku ya existe")
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

// PUT /productos/{id}
func (h *ProductosHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "producto")
		return
	}
	var body productoBody
	if err := httpx.Decode(r, &body); err != nil {
		httpx.BadRequest(w, "cuerpo invalido")
		return
	}
	if det := body.validar(); len(det) > 0 {
		httpx.BadRequest(w, "datos invalidos", det...)
		return
	}
	if !h.categoriaExiste(body.CategoriaID) {
		httpx.BadRequest(w, "categoria_id no existe")
		return
	}
	_, err = h.db.Exec(
		`UPDATE productos SET sku = ?, nombre = ?, categoria_id = ?, precio_centavos = ? WHERE id = ?`,
		body.SKU, body.Nombre, body.CategoriaID, body.PrecioCentavos, id,
	)
	if esViolacionUnicidad(err) {
		httpx.Conflict(w, "el sku ya existe")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	p, _ := h.buscar(id)
	httpx.JSON(w, http.StatusOK, p)
}

// DELETE /productos/{id}
func (h *ProductosHandler) Borrar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "producto")
		return
	}
	var enPedidos int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM pedido_items WHERE producto_id = ?`, id).Scan(&enPedidos)
	if enPedidos > 0 {
		httpx.Conflict(w, "el producto tiene pedidos asociados y no se puede eliminar")
		return
	}
	if _, err := h.db.Exec(`DELETE FROM stock_bodega WHERE producto_id = ?`, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	if _, err := h.db.Exec(`DELETE FROM productos WHERE id = ?`, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

type publicarBody struct {
	Publicado bool `json:"publicado"`
}

// POST /productos/{id}/publicar
func (h *ProductosHandler) Publicar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "producto")
		return
	}
	var body publicarBody
	if err := httpx.Decode(r, &body); err != nil {
		httpx.BadRequest(w, "cuerpo invalido", "se requiere publicado (bool)")
		return
	}
	if _, err := h.db.Exec(`UPDATE productos SET publicado = ? WHERE id = ?`, boolAInt(body.Publicado), id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	p, _ := h.buscar(id)
	httpx.JSON(w, http.StatusOK, p)
}

type ajustarStockBody struct {
	BodegaID   int    `json:"bodega_id"`
	Delta      int    `json:"delta"`
	Referencia string `json:"referencia"`
}

// POST /productos/{id}/ajustar-stock — correccion manual de inventario en una bodega.
func (h *ProductosHandler) AjustarStock(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "producto")
		return
	}
	var body ajustarStockBody
	if err := httpx.Decode(r, &body); err != nil || body.Delta == 0 {
		httpx.BadRequest(w, "cuerpo invalido", "se requiere bodega_id y delta distinto de 0")
		return
	}
	var actual int
	err = h.db.QueryRow(`SELECT cantidad FROM stock_bodega WHERE producto_id = ? AND bodega_id = ?`, id, body.BodegaID).Scan(&actual)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.BadRequest(w, "el producto no tiene stock registrado en esa bodega")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	nuevo := actual + body.Delta
	if nuevo < 0 {
		httpx.Conflict(w, "el ajuste dejaria el stock en negativo")
		return
	}
	tx, err := h.db.Begin()
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE stock_bodega SET cantidad = ? WHERE producto_id = ? AND bodega_id = ?`, nuevo, id, body.BodegaID); err != nil {
		httpx.InternalError(w, err)
		return
	}
	if _, err := tx.Exec(
		`INSERT INTO movimientos_stock(producto_id, bodega_id, tipo, cantidad, referencia) VALUES (?, ?, 'ajuste', ?, ?)`,
		id, body.BodegaID, body.Delta, body.Referencia,
	); err != nil {
		httpx.InternalError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		httpx.InternalError(w, err)
		return
	}
	p, _ := h.buscar(id)
	httpx.JSON(w, http.StatusOK, p)
}

func (h *ProductosHandler) categoriaExiste(id int) bool {
	var n int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM categorias WHERE id = ?`, id).Scan(&n)
	return n > 0
}
