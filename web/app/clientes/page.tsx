"use client";

import { useEffect, useState, useCallback } from "react";
import { api, ApiError } from "@/lib/api";

type Cliente = { id: number; nombre: string; email: string; activo: boolean; creado_en: string };

export default function ClientesPage() {
  const [items, setItems] = useState<Cliente[]>([]);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [accion, setAccion] = useState<string | null>(null);
  const [ocupado, setOcupado] = useState(false);
  const [mostrarForm, setMostrarForm] = useState(false);

  const cargar = useCallback(async () => {
    try {
      const r = await api<{ total: number; items: Cliente[] }>("/clientes?limite=50");
      setItems(r.items);
      setTotal(r.total);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "error");
    }
  }, []);

  useEffect(() => { cargar(); }, [cargar]);

  async function darDeBaja(c: Cliente) {
    setOcupado(true);
    setAccion(null);
    try {
      await api(`/clientes/${c.id}/dar-de-baja`, { method: "POST" });
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
        <h1>Clientes ({total})</h1>
        <button onClick={() => setMostrarForm((v) => !v)}>{mostrarForm ? "Cancelar" : "+ Nuevo cliente"}</button>
      </div>
      {error && <p className="error">{error}</p>}
      {accion && <p className="error">{accion}</p>}
      {mostrarForm && <NuevoClienteForm onCreado={() => { setMostrarForm(false); cargar(); }} />}
      <table>
        <thead><tr><th>Nombre</th><th>Email</th><th>Estado</th><th></th></tr></thead>
        <tbody>
          {items.map((c) => (
            <tr key={c.id}>
              <td>{c.nombre}</td>
              <td>{c.email}</td>
              <td><span className="badge">{c.activo ? "activo" : "de baja"}</span></td>
              <td>{c.activo && <button disabled={ocupado} onClick={() => darDeBaja(c)}>Dar de baja</button>}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function NuevoClienteForm({ onCreado }: { onCreado: () => void }) {
  const [nombre, setNombre] = useState("");
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [enviando, setEnviando] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setEnviando(true);
    try {
      await api("/clientes", { method: "POST", body: JSON.stringify({ nombre, email }) });
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
        <input placeholder="Nombre" value={nombre} onChange={(e) => setNombre(e.target.value)} required />
        <input placeholder="Email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
        <button type="submit" disabled={enviando}>{enviando ? "Creando..." : "Crear"}</button>
      </div>
      {error && <p className="error">{error}</p>}
    </form>
  );
}
