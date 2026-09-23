"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { api, ApiError } from "@/lib/api";

type Item = { id: number; producto_id: number; cantidad: number; precio_unitario_centavos: number };
type Pedido = {
  id: number;
  cliente_id: number;
  vendedor_id: number;
  estado: string;
  subtotal_centavos: number;
  descuento_centavos: number;
  impuestos_centavos: number;
  total_centavos: number;
  cupon_codigo: string | null;
  creado_en: string;
  items: Item[];
};

function moneda(centavos: number) {
  return `$${(centavos / 100).toFixed(2)}`;
}

export default function PedidoDetallePage() {
  const params = useParams<{ id: string }>();
  const [pedido, setPedido] = useState<Pedido | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api<Pedido>(`/pedidos/${params.id}`)
      .then(setPedido)
      .catch((e) => setError(e instanceof ApiError ? e.message : "error"));
  }, [params.id]);

  if (error) return <p className="error">{error}</p>;
  if (!pedido) return <p>Cargando...</p>;

  return (
    <div>
      <h1>Pedido #{pedido.id} <span className="badge">{pedido.estado}</span></h1>
      <div className="card">
        <p>Cliente: {pedido.cliente_id} — Vendedor: {pedido.vendedor_id}</p>
        {pedido.cupon_codigo && <p>Cupon: {pedido.cupon_codigo}</p>}
        <p>Subtotal: {moneda(pedido.subtotal_centavos)}</p>
        <p>Descuento: {moneda(pedido.descuento_centavos)}</p>
        <p>Impuestos: {moneda(pedido.impuestos_centavos)}</p>
        <p><strong>Total: {moneda(pedido.total_centavos)}</strong></p>
      </div>
      <h2>Items</h2>
      <table>
        <thead><tr><th>Producto</th><th>Cantidad</th><th>Precio unitario</th><th>Subtotal</th></tr></thead>
        <tbody>
          {pedido.items.map((it) => (
            <tr key={it.id}>
              <td>{it.producto_id}</td>
              <td>{it.cantidad}</td>
              <td>{moneda(it.precio_unitario_centavos)}</td>
              <td>{moneda(it.cantidad * it.precio_unitario_centavos)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
