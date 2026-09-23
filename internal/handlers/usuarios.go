package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/store"
)

type UsuariosHandler struct {
	db *sql.DB
}

func NewUsuariosHandler(db *sql.DB) *UsuariosHandler {
	return &UsuariosHandler{db: db}
}

type usuario struct {
	ID       int    `json:"id"`
	Nombre   string `json:"nombre"`
	Email    string `json:"email"`
	Rol      string `json:"rol"`
	Activo   bool   `json:"activo"`
	CreadoEn string `json:"creado_en"`
}

// GET /usuarios?rol&activo&limite&desde
func (h *UsuariosHandler) List(w http.ResponseWriter, r *http.Request) {
	pag, ok := httpx.LeerPaginacion(r.URL.Query(), 20)
	if !ok {
		httpx.BadRequest(w, "paginacion invalida")
		return
	}
	where := "1=1"
	args := []any{}
	if v := r.URL.Query().Get("rol"); v != "" {
		where += " AND rol = ?"
		args = append(args, v)
	}
	if v := r.URL.Query().Get("activo"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			httpx.BadRequest(w, "activo invalido")
			return
		}
		where += " AND activo = ?"
		args = append(args, boolAInt(b))
	}
	var total int
	_ = h.db.QueryRow("SELECT COUNT(*) FROM usuarios WHERE "+where, args...).Scan(&total)
	args = append(args, pag.Limite, pag.Desde)
	rows, err := h.db.Query("SELECT id, nombre, email, rol, activo, creado_en FROM usuarios WHERE "+where+" ORDER BY id ASC LIMIT ? OFFSET ?", args...)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	defer rows.Close()
	items := []usuario{}
	for rows.Next() {
		var u usuario
		var activo int
		if err := rows.Scan(&u.ID, &u.Nombre, &u.Email, &u.Rol, &activo, &u.CreadoEn); err != nil {
			httpx.InternalError(w, err)
			return
		}
		u.Activo = activo != 0
		items = append(items, u)
	}
	httpx.JSON(w, http.StatusOK, httpx.ListaRespuesta{Total: total, Items: items})
}

// GET /usuarios/{id}
//
// Nota de diseño (documentada en el README del SUT): un usuario desactivado devuelve
// 403 y no 404. El usuario existe; lo que no hay es acceso a su ficha. Un 404 permitiria
// mapear que emails existen en el sistema probando ids uno por uno.
func (h *UsuariosHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	u, err := h.buscar(id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "usuario")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	if !u.Activo {
		httpx.Forbidden(w, "el usuario esta desactivado")
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

func (h *UsuariosHandler) buscar(id int) (usuario, error) {
	var u usuario
	var activo int
	err := h.db.QueryRow(`SELECT id, nombre, email, rol, activo, creado_en FROM usuarios WHERE id = ?`, id).Scan(&u.ID, &u.Nombre, &u.Email, &u.Rol, &activo, &u.CreadoEn)
	u.Activo = activo != 0
	return u, err
}

var rolesValidos = map[string]bool{"admin": true, "operador": true, "vendedor": true}

type crearUsuarioBody struct {
	Nombre string `json:"nombre"`
	Email  string `json:"email"`
	Rol    string `json:"rol"`
}

// POST /usuarios — el password inicial es siempre el de siembra (store.SeedPassword);
// se cambia con /usuarios/{id}/resetear-password.
func (h *UsuariosHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var body crearUsuarioBody
	if err := httpx.Decode(r, &body); err != nil || body.Nombre == "" || body.Email == "" || !rolesValidos[body.Rol] {
		httpx.BadRequest(w, "datos invalidos", "nombre, email y rol (admin|operador|vendedor) son requeridos")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(store.SeedPassword), bcrypt.DefaultCost)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	res, err := h.db.Exec(`INSERT INTO usuarios(nombre, email, password_hash, rol) VALUES (?, ?, ?, ?)`, body.Nombre, body.Email, string(hash), body.Rol)
	if esViolacionUnicidad(err) {
		httpx.Conflict(w, "el email ya existe")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	id, _ := res.LastInsertId()
	u, _ := h.buscar(int(id))
	httpx.JSON(w, http.StatusCreated, u)
}

type actualizarUsuarioBody struct {
	Nombre string `json:"nombre"`
	Email  string `json:"email"`
}

// PUT /usuarios/{id}
func (h *UsuariosHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "usuario")
		return
	}
	var body actualizarUsuarioBody
	if err := httpx.Decode(r, &body); err != nil || body.Nombre == "" || body.Email == "" {
		httpx.BadRequest(w, "datos invalidos", "nombre y email son requeridos")
		return
	}
	_, err = h.db.Exec(`UPDATE usuarios SET nombre = ?, email = ? WHERE id = ?`, body.Nombre, body.Email, id)
	if esViolacionUnicidad(err) {
		httpx.Conflict(w, "el email ya existe")
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	u, _ := h.buscar(id)
	httpx.JSON(w, http.StatusOK, u)
}

type cambiarRolBody struct {
	Rol string `json:"rol"`
}

// POST /usuarios/{id}/cambiar-rol
func (h *UsuariosHandler) CambiarRol(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "usuario")
		return
	}
	var body cambiarRolBody
	if err := httpx.Decode(r, &body); err != nil || !rolesValidos[body.Rol] {
		httpx.BadRequest(w, "rol invalido", "valores aceptados: admin, operador, vendedor")
		return
	}
	if _, err := h.db.Exec(`UPDATE usuarios SET rol = ? WHERE id = ?`, body.Rol, id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	u, _ := h.buscar(id)
	httpx.JSON(w, http.StatusOK, u)
}

// POST /usuarios/{id}/resetear-password — vuelve la contraseña a la de siembra.
func (h *UsuariosHandler) ResetearPassword(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadRequest(w, "id invalido")
		return
	}
	if _, err := h.buscar(id); errors.Is(err, sql.ErrNoRows) {
		httpx.NotFound(w, "usuario")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(store.SeedPassword), bcrypt.DefaultCost)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	if _, err := h.db.Exec(`UPDATE usuarios SET password_hash = ? WHERE id = ?`, string(hash), id); err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"estado": "password restablecida a la de siembra"})
}
