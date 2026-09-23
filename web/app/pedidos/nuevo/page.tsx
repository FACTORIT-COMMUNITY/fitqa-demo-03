"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";

type Cliente = { id: number; nombre: string; activo: boolean };
type Producto = { id: number; nombre: string; precio_centavos: number; publicado: boolean };
type ItemForm = { producto_id: number; cantidad: number };

export default function NuevoPedidoPage() {
  const router = useRouter();
  const [clientes, setClientes] = useState<Cliente[]>([]);
  const [productos, setProductos] = useState<Producto[]>([]);
  const [clienteId, setClienteId] = useState<number | "">("");
  const [cuponCodigo, setCuponCodigo] = useState("");
  const [items, setItems] = useState<ItemForm[]>([{ producto_id: 0, cantidad: 1 }]);
  const [error, setError] = useState<string | null>(null);
  const [enviando, setEnviando] = useState(false);

  useEffect(() => {
    api<{ items: Cliente[] }>("/clientes?limite=100").then((r) => setClientes(r.items.filter((c) => c.activo)));
    api<{ items: Producto[] }>("/productos?limite=100").then((r) => setProductos(r.items.filter((p) => p.publicado)));
  }, []);

  function actualizarItem(i: number, campo: keyof ItemForm, valor: number) {
    setItems((prev) => prev.map((it, idx) => (idx === i ? { ...it, [campo]: valor } : it)));
  }
  function agregarFila() {
    setItems((prev) => [...prev, { producto_id: 0, cantidad: 1 }]);
  }
  function quitarFila(i: number) {
    setItems((prev) => prev.filter((_, idx) => idx !== i));
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    const validos = items.filter((it) => it.producto_id > 0 && it.cantidad > 0);
    if (!clienteId || validos.length === 0) {
      setError("elegí un cliente y al menos un producto con cantidad");
      return;
    }
    setEnviando(true);
    try {
      const body: Record<string, unknown> = { cliente_id: Number(clienteId), items: validos };
      if (cuponCodigo.trim()) body.cupon_codigo = cuponCodigo.trim();
      const pedido = await api<{ id: number }>("/pedidos", { method: "POST", body: JSON.stringify(body) });
      router.push(`/pedidos/${pedido.id}`);
    } catch (err) {
      setError(err instanceof ApiError ? [err.message, ...(err.detalles ?? [])].join(" — ") : "error desconocido");
    } finally {
      setEnviando(false);
    }
  }

  return (
    <div>
      <h1>Nuevo pedido</h1>
      <form onSubmit={onSubmit} className="card">
        <p>
          <label>Cliente<br />
            <select value={clienteId} onChange={(e) => setClienteId(e.target.value ? Number(e.target.value) : "")}>
              <option value="">— elegir —</option>
              {clientes.map((c) => <option key={c.id} value={c.id}>{c.nombre}</option>)}
            </select>
          </label>
        </p>
        <p>
          <label>Cupón (opcional)<br />
            <input value={cuponCodigo} onChange={(e) => setCuponCodigo(e.target.value)} placeholder="BIENVENIDA10" />
          </label>
        </p>
        <h3>Items</h3>
        {items.map((it, i) => (
          <div key={i} style={{ display: "flex", gap: 8, marginBottom: 8, alignItems: "center" }}>
            <select value={it.producto_id} onChange={(e) => actualizarItem(i, "producto_id", Number(e.target.value))} style={{ flex: 1 }}>
              <option value={0}>— producto —</option>
              {productos.map((p) => (
                <option key={p.id} value={p.id}>{p.nombre} (${(p.precio_centavos / 100).toFixed(2)})</option>
              ))}
            </select>
            <input
              type="number" min={1} value={it.cantidad}
              onChange={(e) => actualizarItem(i, "cantidad", Number(e.target.value))}
              style={{ width: 70 }}
            />
            {items.length > 1 && <button type="button" onClick={() => quitarFila(i)}>quitar</button>}
          </div>
        ))}
        <button type="button" onClick={agregarFila}>+ agregar producto</button>
        <div style={{ marginTop: 16 }}>
          {error && <p className="error">{error}</p>}
          <button type="submit" disabled={enviando}>{enviando ? "Creando..." : "Crear pedido"}</button>
        </div>
      </form>
    </div>
  );
}
