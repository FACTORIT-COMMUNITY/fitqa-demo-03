"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, ApiError } from "@/lib/api";

type Pedido = {
  id: number;
  cliente_id: number;
  vendedor_id: number;
  estado: string;
  total_centavos: number;
  creado_en: string;
};

export default function PedidosPage() {
  const [items, setItems] = useState<Pedido[]>([]);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api<{ total: number; items: Pedido[] }>("/pedidos?limite=50")
      .then((r) => {
        setItems(r.items);
        setTotal(r.total);
      })
      .catch((e) => setError(e instanceof ApiError ? e.message : "error"));
  }, []);

  return (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <h1>Pedidos ({total})</h1>
        <Link href="/pedidos/nuevo"><button>+ Nuevo pedido</button></Link>
      </div>
      {error && <p className="error">{error}</p>}
      <table>
        <thead>
          <tr><th>#</th><th>Cliente</th><th>Vendedor</th><th>Estado</th><th>Total</th><th>Creado</th></tr>
        </thead>
        <tbody>
          {items.map((p) => (
            <tr key={p.id}>
              <td><Link href={`/pedidos/${p.id}`}>#{p.id}</Link></td>
              <td>{p.cliente_id}</td>
              <td>{p.vendedor_id}</td>
              <td><span className="badge">{p.estado}</span></td>
              <td>${(p.total_centavos / 100).toFixed(2)}</td>
              <td>{p.creado_en}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
