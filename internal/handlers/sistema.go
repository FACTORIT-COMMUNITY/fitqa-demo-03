package handlers

import (
	"net/http"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/store"
)

type SistemaHandler struct {
	db *store.DB
}

func NewSistemaHandler(db *store.DB) *SistemaHandler {
	return &SistemaHandler{db: db}
}

// GET /health
func (h *SistemaHandler) Health(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"estado": "ok", "version": "1.0.0"})
}

// POST /admin/reset — recrea el esquema entero y vuelve a sembrar los datos
// deterministicos. Las sesiones (JWT) emitidas antes del reset siguen valiendo: el
// reset es de datos, no de autenticacion.
func (h *SistemaHandler) Reset(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Reset(); err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"estado": "base reiniciada a la semilla"})
}

// GET /admin/seed-info — credenciales de los usuarios sembrados, para no tener que leer
// el codigo fuente para poder autenticarse contra un clone recien levantado.
func (h *SistemaHandler) SeedInfo(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"password_de_todos_los_usuarios_sembrados": store.SeedPassword,
		"usuarios": []map[string]string{
			{"email": "ana.rodriguez@backoffice.demo", "rol": "admin"},
			{"email": "bruno.salas@backoffice.demo", "rol": "operador"},
			{"email": "carla.nunez@backoffice.demo", "rol": "vendedor"},
			{"email": "diego.paredes@backoffice.demo", "rol": "vendedor"},
			{"email": "elena.vidal@backoffice.demo", "rol": "operador (desactivado)"},
		},
	})
}
