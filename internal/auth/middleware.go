package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// Service agrupa el estado de autenticacion: la conexion para resolver usuarios/permisos
// y la lista de tokens de acceso revocados por logout.
type Service struct {
	db  *sql.DB
	mu  sync.Mutex
	rev map[string]bool // jti revocado -> true
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db, rev: make(map[string]bool)}
}

var ErrCredencialesInvalidas = errors.New("credenciales invalidas")
var ErrUsuarioInactivo = errors.New("usuario inactivo")

type Usuario struct {
	ID     int
	Nombre string
	Email  string
	Rol    string
	Activo bool
}

// Login valida email+password contra la base y devuelve el usuario si son correctos.
func (s *Service) Login(email, password string) (Usuario, error) {
	var u Usuario
	var hash string
	var activo int
	err := s.db.QueryRow(
		`SELECT id, nombre, email, password_hash, rol, activo FROM usuarios WHERE email = ?`, email,
	).Scan(&u.ID, &u.Nombre, &u.Email, &hash, &u.Rol, &activo)
	if errors.Is(err, sql.ErrNoRows) {
		return Usuario{}, ErrCredencialesInvalidas
	}
	if err != nil {
		return Usuario{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return Usuario{}, ErrCredencialesInvalidas
	}
	if activo == 0 {
		return Usuario{}, ErrUsuarioInactivo
	}
	u.Activo = activo != 0
	return u, nil
}

// EmitirPar firma el access+refresh token para un usuario ya autenticado.
func (s *Service) EmitirPar(u Usuario) (access, refresh string, err error) {
	access, _, err = SignAccessToken(u.ID, u.Email, u.Rol)
	if err != nil {
		return "", "", err
	}
	refresh, _, err = SignRefreshToken(u.ID, u.Email, u.Rol)
	return access, refresh, err
}

// Refrescar canjea un refresh token vigente por un access token nuevo, releyendo el rol
// y el estado actual del usuario (si lo desactivaron o le cambiaron el rol despues de
// emitir el refresh, el access nuevo ya sale con los datos correctos).
func (s *Service) Refrescar(refreshToken string) (access string, err error) {
	claims, err := parse(refreshToken)
	if err != nil {
		return "", err
	}
	if claims.Tipo != "refresh" {
		return "", errors.New("no es un refresh token")
	}
	if s.estaRevocado(claims.ID) {
		return "", errors.New("token revocado")
	}
	var rol string
	var activo int
	err = s.db.QueryRow(`SELECT rol, activo FROM usuarios WHERE id = ?`, claims.UsuarioID).Scan(&rol, &activo)
	if err != nil {
		return "", err
	}
	if activo == 0 {
		return "", ErrUsuarioInactivo
	}
	access, _, err = SignAccessToken(claims.UsuarioID, claims.Email, rol)
	return access, err
}

// Revocar invalida un jti (usado por logout). Vive en memoria: el reset de datos no
// necesita tocarlo, un jti nuevo nunca va a colisionar con uno viejo.
func (s *Service) Revocar(jti string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rev[jti] = true
}

func (s *Service) estaRevocado(jti string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rev[jti]
}

type ctxKey int

const (
	ctxUsuarioID ctxKey = iota
	ctxEmail
	ctxRol
	ctxJTI
)

func UsuarioID(r *http.Request) int { v, _ := r.Context().Value(ctxUsuarioID).(int); return v }
func Email(r *http.Request) string  { v, _ := r.Context().Value(ctxEmail).(string); return v }
func Rol(r *http.Request) string    { v, _ := r.Context().Value(ctxRol).(string); return v }
func JTI(r *http.Request) string    { v, _ := r.Context().Value(ctxJTI).(string); return v }

// RequireAuth exige un Bearer token de acceso valido, no revocado, de un usuario que
// sigue activo. El chequeo de "sigue activo" es contra la base en cada request: si un
// admin desactiva a alguien a mitad de sesion, el token deja de servir de inmediato.
func (s *Service) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		tokenStr, ok := strings.CutPrefix(auth, "Bearer ")
		if !ok || tokenStr == "" {
			responderNoAutenticado(w)
			return
		}
		claims, err := parse(tokenStr)
		if err != nil || claims.Tipo != "access" {
			responderNoAutenticado(w)
			return
		}
		if s.estaRevocado(claims.ID) {
			responderNoAutenticado(w)
			return
		}
		var activo int
		if err := s.db.QueryRow(`SELECT activo FROM usuarios WHERE id = ?`, claims.UsuarioID).Scan(&activo); err != nil || activo == 0 {
			responderNoAutenticado(w)
			return
		}

		ctx := context.WithValue(r.Context(), ctxUsuarioID, claims.UsuarioID)
		ctx = context.WithValue(ctx, ctxEmail, claims.Email)
		ctx = context.WithValue(ctx, ctxRol, claims.Rol)
		ctx = context.WithValue(ctx, ctxJTI, claims.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func responderNoAutenticado(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"no autenticado"}`))
}

// RequirePermiso exige que el rol del usuario autenticado tenga asignado ese permiso.
// Es el chequeo "correcto": los defectos de autorizacion sembrados en este SUT no rompen
// esta funcion, la omiten en el wiring de la ruta o le falta un chequeo adicional de
// pertenencia (ver README de la ficha en el repo de conocimiento).
func (s *Service) RequirePermiso(permiso string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rol := Rol(r)
			var existe int
			err := s.db.QueryRow(
				`SELECT COUNT(*) FROM rol_permisos WHERE rol = ? AND permiso = ?`, rol, permiso,
			).Scan(&existe)
			if err != nil || existe == 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"no tiene el permiso requerido: ` + permiso + `"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
