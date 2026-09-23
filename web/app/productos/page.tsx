"use client";

import { useEffect, useState, useCallback } from "react";
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
type Categoria = { id: number; nombre: string };

export default function ProductosPage() {
  const [items, setItems] = useState<Producto[]>([]);
  const [categorias, setCategorias] = useState<Categoria[]>([]);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [accion, setAccion] = useState<string | null>(null);
  const [ocupado, setOcupado] = useState(false);
  const [mostrarForm, setMostrarForm] = useState(false);

  const cargar = useCallback(async () => {
    try {
      const [r, c] = await Promise.all([
        api<{ total: number; items: Producto[] }>("/productos?limite=50"),
        api<{ items: Categoria[] }>("/categorias?limite=50"),
      ]);
      setItems(r.items);
      setTotal(r.total);
      setCategorias(c.items);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "error");
    }
  }, []);

  useEffect(() => { cargar(); }, [cargar]);

  async function publicar(p: Producto) {
    setOcupado(true);
    setAccion(null);
    try {
      await api(`/productos/${p.id}/publicar`, { method: "POST", body: JSON.stringify({ publicado: !p.publicado }) });
      await cargar();
    } catch (e) {
      setAccion(e instanceof ApiError ? e.message : "error desconocido");
    } finally {
      setOcupado(false);
    }
  }

  return (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <h1>Productos ({total})</h1>
        <button onClick={() => setMostrarForm((v) => !v)}>{mostrarForm ? "Cancelar" : "+ Nuevo producto"}</button>
      </div>
      {error && <p className="error">{error}</p>}
      {accion && <p className="error">{accion}</p>}
      {mostrarForm && (
        <NuevoProductoForm
          categorias={categorias}
          onCreado={() => { setMostrarForm(false); cargar(); }}
        />
      )}
      <table>
        <thead>
          <tr><th>SKU</th><th>Nombre</th><th>Precio</th><th>Stock</th><th>Publicado</th><th></th></tr>
        </thead>
        <tbody>
          {items.map((p) => (
            <tr key={p.id}>
              <td>{p.sku}</td>
              <td><Link href={`/productos/${p.id}`}>{p.nombre}</Link></td>
              <td>${(p.precio_centavos / 100).toFixed(2)}</td>
              <td>{p.stock_total}</td>
              <td>{p.publicado ? "si" : "no"}</td>
              <td><button disabled={ocupado} onClick={() => publicar(p)}>{p.publicado ? "Despublicar" : "Publicar"}</button></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function NuevoProductoForm({ categorias, onCreado }: { categorias: Categoria[]; onCreado: () => void }) {
  const [sku, setSku] = useState("");
  const [nombre, setNombre] = useState("");
  const [categoriaId, setCategoriaId] = useState<number | "">("");
  const [precio, setPrecio] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [enviando, setEnviando] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setEnviando(true);
    try {
      await api("/productos", {
        method: "POST",
        body: JSON.stringify({ sku, nombre, categoria_id: Number(categoriaId), precio_centavos: Math.round(Number(precio) * 100) }),
      });
      onCreado();
    } catch (err) {
      setError(err instanceof ApiError ? [err.message, ...(err.detalles ?? [])].join(" — ") : "error desconocido");
    } finally {
      setEnviando(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="card">
      <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
        <input placeholder="SKU" value={sku} onChange={(e) => setSku(e.target.value)} required />
        <input placeholder="Nombre" value={nombre} onChange={(e) => setNombre(e.target.value)} required />
        <select value={categoriaId} onChange={(e) => setCategoriaId(e.target.value ? Number(e.target.value) : "")} required>
          <option value="">— categoría —</option>
          {categorias.map((c) => <option key={c.id} value={c.id}>{c.nombre}</option>)}
        </select>
        <input placeholder="Precio (ej. 19.99)" type="number" step="0.01" value={precio} onChange={(e) => setPrecio(e.target.value)} required />
        <button type="submit" disabled={enviando}>{enviando ? "Creando..." : "Crear"}</button>
      </div>
      {error && <p className="error">{error}</p>}
    </form>
  );
}
