"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { api, ApiError } from "@/lib/api";

type Producto = {
  id: number;
  sku: string;
  nombre: string;
  categoria_id: number;
  precio_centavos: number;
  publicado: boolean;
  stock_total: number;
  creado_en: string;
};

export default function ProductoDetallePage() {
  const params = useParams<{ id: string }>();
  const [producto, setProducto] = useState<Producto | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api<Producto>(`/productos/${params.id}`)
      .then(setProducto)
      .catch((e) => setError(e instanceof ApiError ? e.message : "error"));
  }, [params.id]);

  if (error) return <p className="error">{error}</p>;
  if (!producto) return <p>Cargando...</p>;

  return (
    <div>
      <h1>{producto.nombre}</h1>
      <div className="card">
        <p>SKU: {producto.sku}</p>
        <p>Precio: ${(producto.precio_centavos / 100).toFixed(2)}</p>
        <p>Stock total: {producto.stock_total}</p>
        <p>Publicado: {producto.publicado ? "si" : "no"}</p>
        <p>Categoria: {producto.categoria_id}</p>
        <p>Creado: {producto.creado_en}</p>
      </div>
    </div>
  );
}
