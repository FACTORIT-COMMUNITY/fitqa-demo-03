"use client";

import { useEffect, useState, useCallback } from "react";
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
type Bodega = { id: number; nombre: string };

export default function ProductoDetallePage() {
  const params = useParams<{ id: string }>();
  const [producto, setProducto] = useState<Producto | null>(null);
  const [bodegas, setBodegas] = useState<Bodega[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [accion, setAccion] = useState<string | null>(null);
  const [ocupado, setOcupado] = useState(false);
  const [bodegaId, setBodegaId] = useState<number | "">("");
  const [delta, setDelta] = useState("");

  const cargar = useCallback(async () => {
    try {
      const [p, b] = await Promise.all([
        api<Producto>(`/productos/${params.id}`),
        api<{ items: Bodega[] }>("/bodegas?limite=50"),
      ]);
      setProducto(p);
      setBodegas(b.items);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "error");
    }
  }, [params.id]);

  useEffect(() => { cargar(); }, [cargar]);

  async function publicar() {
    if (!producto) return;
    setOcupado(true);
    setAccion(null);
    try {
      await api(`/productos/${producto.id}/publicar`, { method: "POST", body: JSON.stringify({ publicado: !producto.publicado }) });
      await cargar();
    } catch (e) {
      setAccion(e instanceof ApiError ? e.message : "error desconocido");
    } finally {
      setOcupado(false);
    }
  }

  async function ajustarStock(e: React.FormEvent) {
    e.preventDefault();
    if (!producto || !bodegaId || !delta) return;
    setOcupado(true);
    setAccion(null);
    try {
      await api(`/productos/${producto.id}/ajustar-stock`, {
        method: "POST",
        body: JSON.stringify({ bodega_id: Number(bodegaId), delta: Number(delta), referencia: "ajuste manual desde el panel" }),
      });
      setDelta("");
      await cargar();
    } catch (err) {
      setAccion(err instanceof ApiError ? [err.message, ...(err.detalles ?? [])].join(" — ") : "error desconocido");
    } finally {
      setOcupado(false);
    }
  }

  if (error) return <p className="error">{error}</p>;
  if (!producto) return <p>Cargando...</p>;

  return (
    <div>
      <h1>{producto.nombre}</h1>
      <div className="card">
        <p>SKU: {producto.sku}</p>
        <p>Precio: ${(producto.precio_centavos / 100).toFixed(2)}</p>
        <p>Stock total: {producto.stock_total}</p>
        <p>Publicado: <span className="badge">{producto.publicado ? "si" : "no"}</span></p>
        <p>Categoria: {producto.categoria_id}</p>
        <p>Creado: {producto.creado_en}</p>

        {accion && <p className="error">{accion}</p>}
        <button disabled={ocupado} onClick={publicar}>{producto.publicado ? "Despublicar" : "Publicar"}</button>
      </div>

      <div className="card">
        <h2>Ajustar stock</h2>
        <form onSubmit={ajustarStock} style={{ display: "flex", gap: 8, alignItems: "center" }}>
          <select value={bodegaId} onChange={(e) => setBodegaId(e.target.value ? Number(e.target.value) : "")} required>
            <option value="">— bodega —</option>
            {bodegas.map((b) => <option key={b.id} value={b.id}>{b.nombre}</option>)}
          </select>
          <input
            type="number" placeholder="+10 o -5" value={delta}
            onChange={(e) => setDelta(e.target.value)} style={{ width: 100 }} required
          />
          <button type="submit" disabled={ocupado}>Ajustar</button>
        </form>
      </div>
    </div>
  );
}
