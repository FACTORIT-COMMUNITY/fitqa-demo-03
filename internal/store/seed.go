package store

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// SeedPassword es la contraseña de todos los usuarios sembrados. Documentada en el
// README del SUT: no hay nada que "descubrir" para autenticarse, el objetivo es medir
// el resto de la superficie, no un ataque de fuerza bruta al login.
const SeedPassword = "Demo1234!"

// seed carga los datos deterministicos de arranque. Corre siempre dentro de la misma
// transaccion que recrea el esquema, asi que un fallo a mitad de camino no deja la base
// a medio sembrar.
func seed(tx *sql.Tx) error {
	steps := []func(*sql.Tx) error{
		seedRolesYPermisos,
		seedUsuarios,
		seedCategorias,
		seedProductos,
		seedBodegasYStock,
		seedProveedores,
		seedClientesYDirecciones,
		seedPreciosCliente,
		seedCupones,
		seedPedidos,
		seedResenas,
		seedNotificaciones,
	}
	for _, step := range steps {
		if err := step(tx); err != nil {
			return err
		}
	}
	return nil
}

func seedRolesYPermisos(tx *sql.Tx) error {
	roles := []string{"admin", "operador", "vendedor"}
	for _, r := range roles {
		if _, err := tx.Exec(`INSERT INTO roles(nombre) VALUES (?)`, r); err != nil {
			return err
		}
	}

	permisos := []string{
		"productos.gestionar", "categorias.gestionar", "inventario.gestionar",
		"pedidos.gestionar", "pedidos.cancelar", "pagos.confirmar", "finanzas.ver",
		"reportes.ver", "usuarios.gestionar", "cupones.gestionar",
	}
	for _, p := range permisos {
		if _, err := tx.Exec(`INSERT INTO permisos(nombre) VALUES (?)`, p); err != nil {
			return err
		}
	}

	asignaciones := map[string][]string{
		"admin": permisos, // el admin tiene todos
		"operador": {
			"productos.gestionar", "categorias.gestionar", "inventario.gestionar",
			"pedidos.gestionar", "cupones.gestionar",
		},
		"vendedor": {"pedidos.gestionar", "pedidos.cancelar"},
	}
	for rol, ps := range asignaciones {
		for _, p := range ps {
			if _, err := tx.Exec(`INSERT INTO rol_permisos(rol, permiso) VALUES (?, ?)`, rol, p); err != nil {
				return err
			}
		}
	}
	return nil
}

func seedUsuarios(tx *sql.Tx) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(SeedPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashear password de seed: %w", err)
	}

	usuarios := []struct {
		nombre, email, rol string
		activo             int
	}{
		{"Ana Rodríguez", "ana.rodriguez@backoffice.demo", "admin", 1},
		{"Bruno Salas", "bruno.salas@backoffice.demo", "operador", 1},
		{"Carla Núñez", "carla.nunez@backoffice.demo", "vendedor", 1},
		{"Diego Paredes", "diego.paredes@backoffice.demo", "vendedor", 1},
		{"Elena Vidal", "elena.vidal@backoffice.demo", "operador", 0}, // dado de baja
	}
	for _, u := range usuarios {
		if _, err := tx.Exec(
			`INSERT INTO usuarios(nombre, email, password_hash, rol, activo) VALUES (?, ?, ?, ?, ?)`,
			u.nombre, u.email, string(hash), u.rol, u.activo,
		); err != nil {
			return err
		}
	}
	return nil
}

func seedCategorias(tx *sql.Tx) error {
	categorias := []struct {
		nombre string
		orden  int
	}{
		{"Electrónica", 1}, {"Hogar", 2}, {"Deportes", 3}, {"Juguetes", 4}, {"Oficina", 5},
	}
	for _, c := range categorias {
		if _, err := tx.Exec(`INSERT INTO categorias(nombre, orden) VALUES (?, ?)`, c.nombre, c.orden); err != nil {
			return err
		}
	}
	return nil
}

func seedProductos(tx *sql.Tx) error {
	productos := []struct {
		sku, nombre         string
		categoriaID, precio int
		publicado           int
	}{
		{"ELEC-001", "Auriculares inalámbricos", 1, 8999, 1},
		{"ELEC-002", "Cargador USB-C 30W", 1, 3499, 1},
		{"ELEC-003", "Mouse inalámbrico", 1, 1999, 1},
		{"HOG-001", "Set de sábanas queen", 2, 5499, 1},
		{"HOG-002", "Cafetera de goteo", 2, 12999, 1},
		{"DEP-001", "Balón de fútbol N°5", 3, 2999, 1},
		{"DEP-002", "Mancuernas 5kg (par)", 3, 4599, 0}, // sin publicar, para el endpoint de publicar
		{"JUG-001", "Cubo de rompecabezas", 4, 1999, 1},
		{"OFI-001", "Silla ergonómica", 5, 25999, 1},
		{"OFI-002", "Resma de papel A4", 5, 899, 1},
	}
	for _, p := range productos {
		if _, err := tx.Exec(
			`INSERT INTO productos(sku, nombre, categoria_id, precio_centavos, publicado) VALUES (?, ?, ?, ?, ?)`,
			p.sku, p.nombre, p.categoriaID, p.precio, p.publicado,
		); err != nil {
			return err
		}
	}
	return nil
}

func seedBodegasYStock(tx *sql.Tx) error {
	bodegas := []struct{ nombre, direccion string }{
		{"Bodega Central", "Cra 45 #12-30, Bogotá"},
		{"Bodega Norte", "Cl 170 #8-20, Bogotá"},
	}
	for _, b := range bodegas {
		if _, err := tx.Exec(`INSERT INTO bodegas(nombre, direccion) VALUES (?, ?)`, b.nombre, b.direccion); err != nil {
			return err
		}
	}

	// Existencias por producto y bodega. Determinístico: producto id * 7 en la central,
	// producto id * 3 en la del norte (menos surtida).
	for pid := 1; pid <= 10; pid++ {
		if _, err := tx.Exec(
			`INSERT INTO stock_bodega(producto_id, bodega_id, cantidad) VALUES (?, 1, ?)`,
			pid, pid*7,
		); err != nil {
			return err
		}
		if _, err := tx.Exec(
			`INSERT INTO stock_bodega(producto_id, bodega_id, cantidad) VALUES (?, 2, ?)`,
			pid, pid*3,
		); err != nil {
			return err
		}
	}
	return nil
}

func seedProveedores(tx *sql.Tx) error {
	proveedores := []struct{ nombre, email, telefono string }{
		{"Distribuidora Andina S.A.S.", "ventas@andina.demo", "+57 601 555 0101"},
		{"Importadora del Pacífico", "contacto@pacifico.demo", "+57 601 555 0102"},
		{"Manufacturas El Roble", "pedidos@elroble.demo", "+57 601 555 0103"},
	}
	for _, p := range proveedores {
		if _, err := tx.Exec(`INSERT INTO proveedores(nombre, email, telefono) VALUES (?, ?, ?)`, p.nombre, p.email, p.telefono); err != nil {
			return err
		}
	}
	return nil
}

func seedClientesYDirecciones(tx *sql.Tx) error {
	clientes := []struct {
		nombre, email string
		activo        int
	}{
		{"Laura Gómez", "laura.gomez@cliente.demo", 1},
		{"Marco Cifuentes", "marco.cifuentes@cliente.demo", 1},
		{"Nadia Ospina", "nadia.ospina@cliente.demo", 1},
		{"Óscar Villalba", "oscar.villalba@cliente.demo", 1},
		{"Paula Restrepo", "paula.restrepo@cliente.demo", 0}, // dada de baja
	}
	for i, c := range clientes {
		res, err := tx.Exec(`INSERT INTO clientes(nombre, email, activo) VALUES (?, ?, ?)`, c.nombre, c.email, c.activo)
		if err != nil {
			return err
		}
		clienteID, _ := res.LastInsertId()
		_ = i
		if _, err := tx.Exec(
			`INSERT INTO direcciones(cliente_id, calle, ciudad, es_principal) VALUES (?, ?, ?, 1)`,
			clienteID, fmt.Sprintf("Calle %d # %d-%d", 10+clienteID, clienteID, clienteID*2), "Bogotá",
		); err != nil {
			return err
		}
	}
	return nil
}

func seedPreciosCliente(tx *sql.Tx) error {
	// Laura (cliente 1) tiene precio preferencial en el producto 1.
	_, err := tx.Exec(
		`INSERT INTO precios_cliente(cliente_id, producto_id, precio_centavos) VALUES (1, 1, 7999)`,
	)
	return err
}

func seedCupones(tx *sql.Tx) error {
	cupones := []struct {
		codigo, tipo string
		valor        int
		activo       int
	}{
		{"BIENVENIDA10", "porcentaje", 10, 1},
		{"DESCUENTO5K", "fijo", 5000, 1},
		{"VERANO2026", "porcentaje", 15, 0}, // inactivo, para el endpoint de activar/desactivar
	}
	for _, c := range cupones {
		if _, err := tx.Exec(
			`INSERT INTO cupones(codigo, tipo, valor, activo) VALUES (?, ?, ?, ?)`,
			c.codigo, c.tipo, c.valor, c.activo,
		); err != nil {
			return err
		}
	}
	return nil
}

func seedPedidos(tx *sql.Tx) error {
	type item struct {
		productoID, cantidad, precioUnitario int
	}
	pedidos := []struct {
		clienteID, vendedorID int
		estado                string
		cuponCodigo           string
		items                 []item
	}{
		{1, 3, "confirmado", "BIENVENIDA10", []item{{1, 1, 8999}, {2, 2, 3499}}},
		{2, 3, "confirmado", "", []item{{4, 1, 12999}}},
		{3, 4, "borrador", "", []item{{6, 2, 2999}}},
		{4, 4, "cancelado", "", []item{{8, 1, 25999}}},
		{1, 3, "confirmado", "DESCUENTO5K", []item{{3, 3, 1999}, {9, 5, 899}}},
	}

	for _, p := range pedidos {
		subtotal := 0
		for _, it := range p.items {
			subtotal += it.cantidad * it.precioUnitario
		}
		descuento := 0
		switch p.cuponCodigo {
		case "BIENVENIDA10":
			descuento = subtotal * 10 / 100
		case "DESCUENTO5K":
			descuento = 5000
		}
		impuestos := (subtotal - descuento) * 19 / 100
		total := subtotal - descuento + impuestos

		var cuponCodigo sql.NullString
		if p.cuponCodigo != "" {
			cuponCodigo = sql.NullString{String: p.cuponCodigo, Valid: true}
		}

		res, err := tx.Exec(
			`INSERT INTO pedidos(cliente_id, vendedor_id, estado, subtotal_centavos, descuento_centavos, impuestos_centavos, total_centavos, cupon_codigo)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			p.clienteID, p.vendedorID, p.estado, subtotal, descuento, impuestos, total, cuponCodigo,
		)
		if err != nil {
			return err
		}
		pedidoID, _ := res.LastInsertId()

		for _, it := range p.items {
			if _, err := tx.Exec(
				`INSERT INTO pedido_items(pedido_id, producto_id, cantidad, precio_unitario_centavos) VALUES (?, ?, ?, ?)`,
				pedidoID, it.productoID, it.cantidad, it.precioUnitario,
			); err != nil {
				return err
			}
		}

		if p.estado == "confirmado" {
			if _, err := tx.Exec(
				`INSERT INTO pagos(pedido_id, monto_centavos, estado, metodo) VALUES (?, ?, 'confirmado', 'tarjeta')`,
				pedidoID, total,
			); err != nil {
				return err
			}
			if _, err := tx.Exec(
				`INSERT INTO envios(pedido_id, estado, transportista, tracking) VALUES (?, 'pendiente', 'Servientrega', NULL)`,
				pedidoID,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func seedResenas(tx *sql.Tx) error {
	resenas := []struct {
		productoID, clienteID, calificacion int
		comentario, estado                  string
	}{
		{1, 1, 5, "Excelente calidad de sonido.", "aprobada"},
		{1, 2, 2, "Se desconecta seguido.", "pendiente"},
		{4, 3, 4, "Buena cafetera, un poco ruidosa.", "aprobada"},
	}
	for _, r := range resenas {
		if _, err := tx.Exec(
			`INSERT INTO resenas(producto_id, cliente_id, calificacion, comentario, estado) VALUES (?, ?, ?, ?, ?)`,
			r.productoID, r.clienteID, r.calificacion, r.comentario, r.estado,
		); err != nil {
			return err
		}
	}
	return nil
}

func seedNotificaciones(tx *sql.Tx) error {
	notificaciones := []struct {
		usuarioID int
		mensaje   string
	}{
		{1, "Hay 1 reseña pendiente de moderar."},
		{2, "El cupón VERANO2026 está inactivo."},
	}
	for _, n := range notificaciones {
		if _, err := tx.Exec(
			`INSERT INTO notificaciones(usuario_id, mensaje) VALUES (?, ?)`,
			n.usuarioID, n.mensaje,
		); err != nil {
			return err
		}
	}
	return nil
}
