package handlers

import (
	"database/sql"
	"net/http"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
)

type RolesHandler struct {
	db *sql.DB
}

func NewRolesHandler(db *sql.DB) *RolesHandler {
	return &RolesHandler{db: db}
}

// GET /roles
func (h *RolesHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT rol.nombre, GROUP_CONCAT(rp.permiso)
		FROM roles rol LEFT JOIN rol_permisos rp ON rp.rol = rol.nombre
		GROUP BY rol.nombre ORDER BY rol.nombre ASC`)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	type rolConPermisos struct {
		Nombre   string   `json:"nombre"`
		Permisos []string `json:"permisos"`
	}
	items := []rolConPermisos{}
	for rows.Next() {
		var nombre string
		var permisos sql.NullString
		if err := rows.Scan(&nombre, &permisos); err != nil {
			httpx.InternalError(w, err)
			return
		}
		rp := rolConPermisos{Nombre: nombre, Permisos: []string{}}
		if permisos.Valid {
			rp.Permisos = splitCSV(permisos.String)
		}
		items = append(items, rp)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: len(items), Items: items})
}

// GET /permisos
func (h *RolesHandler) ListPermisos(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`SELECT nombre FROM permisos ORDER BY nombre ASC`)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var nombre string
		if err := rows.Scan(&nombre); err != nil {
			httpx.InternalError(w, err)
			return
		}
		items = append(items, nombre)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: len(items), Items: items})
}

type asignarPermisoBody struct {
	Rol     string `json:"rol"`
	Permiso string `json:"permiso"`
}

// POST /roles/asignar-permiso
func (h *RolesHandler) AsignarPermiso(w http.ResponseWriter, r *http.Request) {
	var body asignarPermisoBody
	if err := httpx.Decode(r, &body); err != nil || body.Rol == "" || body.Permiso == "" {
		httpx.BadRequest(w, "datos invalidos", "se requiere rol y permiso")
		return
	}
	var rolExiste, permisoExiste int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM roles WHERE nombre = ?`, body.Rol).Scan(&rolExiste)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM permisos WHERE nombre = ?`, body.Permiso).Scan(&permisoExiste)
	if rolExiste == 0 || permisoExiste == 0 {
		httpx.BadRequest(w, "rol o permiso no existen")
		return
	}
	if _, err := h.db.Exec(`INSERT OR IGNORE INTO rol_permisos(rol, permiso) VALUES (?, ?)`, body.Rol, body.Permiso); err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"estado": "permiso asignado"})
}

func splitCSV(s string) []string {
	if s == "" {
		return []string{}
	}
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
