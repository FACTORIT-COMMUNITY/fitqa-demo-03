package handlers

import (
	"errors"
	"net/http"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/auth"
	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
)

type AuthHandler struct {
	svc *auth.Service
}

func NewAuthHandler(svc *auth.Service) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body loginBody
	if err := httpx.Decode(r, &body); err != nil || body.Email == "" || body.Password == "" {
		httpx.BadRequest(w, "cuerpo invalido", "se requiere email y password")
		return
	}
	u, err := h.svc.Login(body.Email, body.Password)
	if err != nil {
		if errors.Is(err, auth.ErrUsuarioInactivo) {
			httpx.Forbidden(w, "usuario inactivo")
			return
		}
		httpx.Unauthorized(w, "credenciales invalidas")
		return
	}
	access, refresh, err := h.svc.EmitirPar(u)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"access_token":  access,
		"refresh_token": refresh,
		"usuario": map[string]any{
			"id": u.ID, "nombre": u.Nombre, "email": u.Email, "rol": u.Rol,
		},
	})
}

type refreshBody struct {
	RefreshToken string `json:"refresh_token"`
}

// POST /auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body refreshBody
	if err := httpx.Decode(r, &body); err != nil || body.RefreshToken == "" {
		httpx.BadRequest(w, "cuerpo invalido", "se requiere refresh_token")
		return
	}
	access, err := h.svc.Refrescar(body.RefreshToken)
	if err != nil {
		httpx.Unauthorized(w, "refresh token invalido")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"access_token": access})
}

// POST /auth/logout — requiere auth. Revoca el jti del access token actual.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.svc.Revocar(auth.JTI(r))
	httpx.JSON(w, http.StatusOK, map[string]string{"estado": "sesion cerrada"})
}

// GET /auth/me — requiere auth.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"usuario_id": auth.UsuarioID(r),
		"email":      auth.Email(r),
		"rol":        auth.Rol(r),
	})
}
