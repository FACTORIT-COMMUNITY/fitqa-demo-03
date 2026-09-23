// Comando api: back-office de un e-commerce chico. Todo el estado vive en sqlite en
// memoria; POST /admin/reset lo vuelve a la semilla determinística.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/auth"
	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/handlers"
	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/httpx"
	"github.com/SantiagoMartinezCO/fitqa-demo-03/internal/store"
)

func main() {
	db, err := store.Open()
	if err != nil {
		log.Fatalf("abrir la base: %v", err)
	}

	authSvc := auth.NewService(db.DB)

	authH := handlers.NewAuthHandler(authSvc)
	productosH := handlers.NewProductosHandler(db.DB)
	categoriasH := handlers.NewCategoriasHandler(db.DB)
	proveedoresH := handlers.NewProveedoresHandler(db.DB)
	clientesH := handlers.NewClientesHandler(db.DB)
	pedidosH := handlers.NewPedidosHandler(db.DB)
	pagosH := handlers.NewPagosHandler(db.DB)
	enviosH := handlers.NewEnviosHandler(db.DB)
	cuponesH := handlers.NewCuponesHandler(db.DB)
	inventarioH := handlers.NewInventarioHandler(db.DB)
	bodegasH := handlers.NewBodegasHandler(db.DB)
	preciosH := handlers.NewPreciosHandler(db.DB)
	resenasH := handlers.NewResenasHandler(db.DB)
	usuariosH := handlers.NewUsuariosHandler(db.DB)
	rolesH := handlers.NewRolesHandler(db.DB)
	notificacionesH := handlers.NewNotificacionesHandler(db.DB)
	auditoriaH := handlers.NewAuditoriaHandler(db.DB)
	reportesH := handlers.NewReportesHandler(db.DB)
	sistemaH := handlers.NewSistemaHandler(db)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) { httpx.NotFound(w, "ruta") })
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		httpx.Error(w, http.StatusMethodNotAllowed, "metodo no permitido para esta ruta")
	})

	// --- Sistema (sin auth: el agente necesita poder resetear y consultar
	// credenciales antes de tener un token) ---
	r.Get("/health", sistemaH.Health)
	r.Post("/admin/reset", sistemaH.Reset)
	r.Get("/admin/seed-info", sistemaH.SeedInfo)

	// --- Auth ---
	r.Post("/auth/login", authH.Login)
	r.Post("/auth/refresh", authH.Refresh)

	r.Group(func(r chi.Router) {
		r.Use(authSvc.RequireAuth)

		r.Post("/auth/logout", authH.Logout)
		r.Get("/auth/me", authH.Me)

		// Productos: catalogo visible para cualquier rol autenticado, escritura
		// reservada a quien gestiona el catalogo.
		r.Route("/productos", func(r chi.Router) {
			r.Get("/", productosH.List)
			r.Get("/{id}", productosH.Get)
			r.Group(func(r chi.Router) {
				r.Use(authSvc.RequirePermiso("productos.gestionar"))
				r.Post("/", productosH.Crear)
				r.Put("/{id}", productosH.Actualizar)
				r.Delete("/{id}", productosH.Borrar)
				r.Post("/{id}/publicar", productosH.Publicar)
				r.Post("/{id}/ajustar-stock", productosH.AjustarStock)
			})
		})

		r.Route("/categorias", func(r chi.Router) {
			r.Get("/", categoriasH.List)
			r.Get("/{id}", categoriasH.Get)
			r.Group(func(r chi.Router) {
				r.Use(authSvc.RequirePermiso("categorias.gestionar"))
				r.Post("/", categoriasH.Crear)
				r.Put("/{id}", categoriasH.Actualizar)
				r.Delete("/{id}", categoriasH.Borrar)
				r.Post("/reordenar", categoriasH.Reordenar)
			})
		})

		r.Route("/proveedores", func(r chi.Router) {
			r.Get("/", proveedoresH.List)
			r.Get("/{id}", proveedoresH.Get)
			r.Group(func(r chi.Router) {
				r.Use(authSvc.RequirePermiso("inventario.gestionar"))
				r.Post("/", proveedoresH.Crear)
				r.Put("/{id}", proveedoresH.Actualizar)
				r.Delete("/{id}", proveedoresH.Borrar)
			})
		})

		r.Route("/clientes", func(r chi.Router) {
			r.Get("/", clientesH.List)
			r.Get("/{id}", clientesH.Get)
			r.Get("/{id}/direcciones", clientesH.ListDirecciones)
			r.Group(func(r chi.Router) {
				r.Use(authSvc.RequirePermiso("pedidos.gestionar"))
				r.Post("/", clientesH.Crear)
				r.Put("/{id}", clientesH.Actualizar)
				r.Post("/{id}/dar-de-baja", clientesH.DarDeBaja)
				r.Post("/{id}/direcciones", clientesH.CrearDireccion)
				r.Put("/{id}/direcciones/{direccion_id}", clientesH.ActualizarDireccion)
				r.Delete("/{id}/direcciones/{direccion_id}", clientesH.BorrarDireccion)
			})
		})

		r.Route("/pedidos", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(authSvc.RequirePermiso("pedidos.gestionar"))
				r.Get("/", pedidosH.List)
				r.Get("/{id}", pedidosH.Get)
				r.Post("/", pedidosH.Crear)
				r.Post("/{id}/actualizar-estado", pedidosH.ActualizarEstado)
				r.Get("/{id}/items", pedidosH.ListItems)
				r.Post("/{id}/items", pedidosH.AgregarItem)
				r.Delete("/{id}/items/{item_id}", pedidosH.QuitarItem)
			})
			r.Group(func(r chi.Router) {
				r.Use(authSvc.RequirePermiso("pedidos.cancelar"))
				r.Post("/{id}/cancelar", pedidosH.Cancelar)
			})
			r.Group(func(r chi.Router) {
				r.Use(authSvc.RequirePermiso("pagos.confirmar"))
				r.Post("/{id}/reembolsar", pedidosH.Reembolsar)
			})
		})

		r.Route("/pagos", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(authSvc.RequirePermiso("pedidos.gestionar"))
				r.Get("/", pagosH.List)
				r.Get("/{id}", pagosH.Get)
				r.Post("/", pagosH.Crear)
				r.Post("/{id}/confirmar", pagosH.Confirmar)
			})
			r.Group(func(r chi.Router) {
				r.Use(authSvc.RequirePermiso("pagos.confirmar"))
				r.Post("/{id}/anular", pagosH.Anular)
			})
		})

		r.Route("/envios", func(r chi.Router) {
			r.Use(authSvc.RequirePermiso("pedidos.gestionar"))
			r.Get("/", enviosH.List)
			r.Get("/{id}", enviosH.Get)
			r.Post("/", enviosH.Crear)
			r.Post("/{id}/actualizar-estado", enviosH.ActualizarEstado)
			r.Post("/{id}/marcar-entregado", enviosH.MarcarEntregado)
		})

		r.Route("/cupones", func(r chi.Router) {
			r.Get("/", cuponesH.List)
			r.Get("/validar", cuponesH.Validar)
			r.Get("/{id}", cuponesH.Get)
			r.Group(func(r chi.Router) {
				r.Use(authSvc.RequirePermiso("cupones.gestionar"))
				r.Post("/", cuponesH.Crear)
				r.Put("/{id}", cuponesH.Actualizar)
				r.Delete("/{id}", cuponesH.Borrar)
				r.Post("/{id}/activar", cuponesH.Activar)
			})
		})

		r.Route("/inventario", func(r chi.Router) {
			r.Get("/movimientos", inventarioH.List)
			r.Get("/productos/{producto_id}/historial", inventarioH.HistorialPorProducto)
			r.Group(func(r chi.Router) {
				r.Use(authSvc.RequirePermiso("inventario.gestionar"))
				r.Post("/movimientos", inventarioH.Crear)
			})
		})

		r.Route("/bodegas", func(r chi.Router) {
			r.Get("/", bodegasH.List)
			r.Get("/{id}", bodegasH.Get)
			r.Group(func(r chi.Router) {
				r.Use(authSvc.RequirePermiso("inventario.gestionar"))
				r.Post("/", bodegasH.Crear)
				r.Put("/{id}", bodegasH.Actualizar)
				r.Delete("/{id}", bodegasH.Borrar)
				r.Post("/transferir-stock", bodegasH.TransferirStock)
			})
		})

		r.Route("/precios-cliente", func(r chi.Router) {
			r.Get("/", preciosH.List)
			r.Get("/{id}", preciosH.Get)
			r.Group(func(r chi.Router) {
				r.Use(authSvc.RequirePermiso("productos.gestionar"))
				r.Post("/", preciosH.Crear)
				r.Put("/{id}", preciosH.Actualizar)
				r.Delete("/{id}", preciosH.Borrar)
			})
		})

		r.Route("/resenas", func(r chi.Router) {
			r.Get("/", resenasH.List)
			r.Post("/", resenasH.Crear)
			r.Group(func(r chi.Router) {
				r.Use(authSvc.RequirePermiso("productos.gestionar"))
				r.Post("/{id}/moderar", resenasH.Moderar)
				r.Delete("/{id}", resenasH.Borrar)
			})
		})

		r.Route("/usuarios", func(r chi.Router) {
			r.Use(authSvc.RequirePermiso("usuarios.gestionar"))
			r.Get("/", usuariosH.List)
			r.Get("/{id}", usuariosH.Get)
			r.Post("/", usuariosH.Crear)
			r.Put("/{id}", usuariosH.Actualizar)
			r.Post("/{id}/cambiar-rol", usuariosH.CambiarRol)
			r.Post("/{id}/resetear-password", usuariosH.ResetearPassword)
		})

		r.Group(func(r chi.Router) {
			r.Use(authSvc.RequirePermiso("usuarios.gestionar"))
			r.Get("/roles", rolesH.ListRoles)
			r.Get("/permisos", rolesH.ListPermisos)
			r.Post("/roles/asignar-permiso", rolesH.AsignarPermiso)
			r.Get("/auditoria", auditoriaH.List)
			r.Get("/auditoria/recurso/{recurso}/{recurso_id}", auditoriaH.GetPorRecurso)
		})

		r.Route("/notificaciones", func(r chi.Router) {
			r.Get("/", notificacionesH.List)
			r.Post("/{id}/marcar-leida", notificacionesH.MarcarLeida)
		})

		r.Group(func(r chi.Router) {
			r.Use(authSvc.RequirePermiso("reportes.ver"))
			r.Get("/reportes/ventas-por-periodo", reportesH.VentasPorPeriodo)
			r.Get("/reportes/top-productos", reportesH.TopProductos)
			r.Get("/reportes/stock-bajo", reportesH.StockBajo)
		})
	})

	puerto := os.Getenv("PORT")
	if puerto == "" {
		puerto = "8080"
	}
	log.Printf("escuchando en :%s", puerto)
	if err := http.ListenAndServe(":"+puerto, r); err != nil {
		log.Fatal(err)
	}
}

// cors habilita al panel Next (otro origen/puerto) a llamar la API. No hay
// credenciales por cookie: el token viaja en el header Authorization.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
