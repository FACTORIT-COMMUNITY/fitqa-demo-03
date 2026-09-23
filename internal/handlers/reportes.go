package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
)

type ReportesHandler struct {
	db *sql.DB
}

func NewReportesHandler(db *sql.DB) *ReportesHandler {
	return &ReportesHandler{db: db}
}

// GET /reportes/ventas-por-periodo?desde=YYYY-MM-DD&hasta=YYYY-MM-DD
func (h *ReportesHandler) VentasPorPeriodo(w http.ResponseWriter, r *http.Request) {
	desde := r.URL.Query().Get("desde")
	hasta := r.URL.Query().Get("hasta")
	if desde == "" || hasta == "" {
		httpx.BadRequest(w, "se requieren los parametros desde y hasta (YYYY-MM-DD)")
		return
	}
	row := h.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(total_centavos), 0)
		FROM pedidos
		WHERE estado IN ('confirmado', 'reembolsado')
		  AND date(creado_en) BETWEEN date(?) AND date(?)`, desde, hasta)
	var cantidad, totalCentavos int
	if err := row.Scan(&cantidad, &totalCentavos); err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"desde": desde, "hasta": hasta,
		"pedidos": cantidad, "total_centavos": totalCentavos,
	})
}

type productoVendido struct {
	ProductoID       int    `json:"producto_id"`
	Nombre           string `json:"nombre"`
	UnidadesVendidas int    `json:"unidades_vendidas"`
}

// GET /reportes/top-productos?limite=
func (h *ReportesHandler) TopProductos(w http.ResponseWriter, r *http.Request) {
	limite := 10
	if v := r.URL.Query().Get("limite"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			httpx.BadRequest(w, "limite invalido", "debe estar entre 1 y 100")
			return
		}
		limite = n
	}
	rows, err := h.db.Query(`
		SELECT p.id, p.nombre, SUM(pi.cantidad) AS unidades
		FROM pedido_items pi
		JOIN pedidos ped ON ped.id = pi.pedido_id
		JOIN productos p ON p.id = pi.producto_id
		WHERE ped.estado IN ('confirmado', 'reembolsado')
		GROUP BY p.id, p.nombre
		ORDER BY unidades DESC
		LIMIT ?`, limite)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []productoVendido{}
	for rows.Next() {
		var pv productoVendido
		if err := rows.Scan(&pv.ProductoID, &pv.Nombre, &pv.UnidadesVendidas); err != nil {
			httpx.InternalError(w, err)
			return
		}
		items = append(items, pv)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: len(items), Items: items})
}

type productoStockBajo struct {
	ProductoID int    `json:"producto_id"`
	Nombre     string `json:"nombre"`
	StockTotal int    `json:"stock_total"`
}

// GET /reportes/stock-bajo?umbral=
func (h *ReportesHandler) StockBajo(w http.ResponseWriter, r *http.Request) {
	umbral := 10
	if v := r.URL.Query().Get("umbral"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			httpx.BadRequest(w, "umbral invalido")
			return
		}
		umbral = n
	}
	rows, err := h.db.Query(`
		SELECT p.id, p.nombre, COALESCE(SUM(sb.cantidad), 0) AS stock
		FROM productos p LEFT JOIN stock_bodega sb ON sb.producto_id = p.id
		GROUP BY p.id, p.nombre
		HAVING stock < ?
		ORDER BY stock ASC`, umbral)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []productoStockBajo{}
	for rows.Next() {
		var ps productoStockBajo
		if err := rows.Scan(&ps.ProductoID, &ps.Nombre, &ps.StockTotal); err != nil {
			httpx.InternalError(w, err)
			return
		}
		items = append(items, ps)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: len(items), Items: items})
}
