"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, ApiError } from "@/lib/api";

type Producto = {
  id: number;
  sku: string;
  nombre: string;
  categoria_id: number;
  precio_centavos: number;
  publicado: boolean;
  stock_total: number;
};

export default function ProductosPage() {
  const [items, setItems] = useState<Producto[]>([]);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api<{ total: number; items: Producto[] }>("/productos?limite=50")
      .then((r) => {
        setItems(r.items);
        setTotal(r.total);
      })
      .catch((e) => setError(e instanceof ApiError ? e.message : "error"));
  }, []);

  return (
    <div>
      <h1>Productos ({total})</h1>
      {error && <p className="error">{error}</p>}
      <table>
        <thead>
          <tr><th>SKU</th><th>Nombre</th><th>Precio</th><th>Stock</th><th>Publicado</th></tr>
        </thead>
        <tbody>
          {items.map((p) => (
            <tr key={p.id}>
              <td>{p.sku}</td>
              <td><Link href={`/productos/${p.id}`}>{p.nombre}</Link></td>
              <td>${(p.precio_centavos / 100).toFixed(2)}</td>
              <td>{p.stock_total}</td>
              <td>{p.publicado ? "si" : "no"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
