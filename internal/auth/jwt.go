// Package auth firma y valida los JWT propios (sin proveedor externo) y expone los
// middlewares de autenticacion y autorizacion por permiso.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// clave de firma del proceso. En un sistema real vendria de una variable de entorno;
// aca se genera una vez al arrancar porque el reset del SUT no tiene por que invalidar
// sesiones firmadas antes del reset (el reset es de DATOS, no de sesiones).
var claveFirma = generarClave()

func generarClave() []byte {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return b
}

const (
	duracionAccess  = 15 * time.Minute
	duracionRefresh = 7 * 24 * time.Hour
)

// Claims son los datos propios que viajan en el JWT, ademas de los registrados (exp, jti).
type Claims struct {
	UsuarioID int    `json:"usuario_id"`
	Email     string `json:"email"`
	Rol       string `json:"rol"`
	Tipo      string `json:"tipo"` // "access" | "refresh"
	jwt.RegisteredClaims
}

func nuevoJTI() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func firmar(usuarioID int, email, rol, tipo string, duracion time.Duration) (string, string, error) {
	jti := nuevoJTI()
	claims := Claims{
		UsuarioID: usuarioID,
		Email:     email,
		Rol:       rol,
		Tipo:      tipo,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duracion)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString(claveFirma)
	return s, jti, err
}

// SignAccessToken emite el token corto que se manda en cada request como Bearer.
func SignAccessToken(usuarioID int, email, rol string) (string, string, error) {
	return firmar(usuarioID, email, rol, "access", duracionAccess)
}

// SignRefreshToken emite el token largo que solo sirve para pedir un access nuevo.
func SignRefreshToken(usuarioID int, email, rol string) (string, string, error) {
	return firmar(usuarioID, email, rol, "refresh", duracionRefresh)
}

func parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("metodo de firma inesperado: %v", t.Header["alg"])
		}
		return claveFirma, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("token invalido")
	}
	return claims, nil
}
