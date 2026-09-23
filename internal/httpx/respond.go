// Package httpx trae los helpers de request/response que se repiten en todos los
// handlers: responder JSON, responder error con el mismo shape, y leer paginación.
package httpx

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// ErrorBody es el shape uniforme de error de toda la API: {"error": "...", "detalles": [...]}.
type ErrorBody struct {
	Error    string   `json:"error"`
	Detalles []string `json:"detalles,omitempty"`
}

func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

func Error(w http.ResponseWriter, status int, mensaje string, detalles ...string) {
	JSON(w, status, ErrorBody{Error: mensaje, Detalles: detalles})
}

func BadRequest(w http.ResponseWriter, mensaje string, detalles ...string) {
	Error(w, http.StatusBadRequest, mensaje, detalles...)
}

func NotFound(w http.ResponseWriter, recurso string) {
	Error(w, http.StatusNotFound, recurso+" no encontrado")
}

func Conflict(w http.ResponseWriter, mensaje string) {
	Error(w, http.StatusConflict, mensaje)
}

func Forbidden(w http.ResponseWriter, mensaje string) {
	Error(w, http.StatusForbidden, mensaje)
}

func Unauthorized(w http.ResponseWriter, mensaje string) {
	Error(w, http.StatusUnauthorized, mensaje)
}

func InternalError(w http.ResponseWriter, err error) {
	Error(w, http.StatusInternalServerError, "error interno")
}

func Decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// Paginacion son los parametros de listado que casi todos los endpoints de lista aceptan.
type Paginacion struct {
	Limite int
	Desde  int
}

// LeerPaginacion valida limite (1-100, default 20) y desde (>=0, default 0). Devuelve
// false si algun parametro esta fuera de rango (el caller responde 400).
func LeerPaginacion(q map[string][]string, limiteDefault int) (Paginacion, bool) {
	p := Paginacion{Limite: limiteDefault, Desde: 0}
	if v, ok := valorUnico(q, "limite"); ok {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			return p, false
		}
		p.Limite = n
	}
	if v, ok := valorUnico(q, "desde"); ok {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return p, false
		}
		p.Desde = n
	}
	return p, true
}

func valorUnico(q map[string][]string, clave string) (string, bool) {
	vs, ok := q[clave]
	if !ok || len(vs) == 0 {
		return "", false
	}
	return vs[0], true
}

// ListaRespuesta es el shape uniforme de toda respuesta de listado paginado.
type ListaRespuesta struct {
	Total int `json:"total"`
	Items any `json:"items"`
}
