"use client";

import { useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api";

type Cliente = { id: number; nombre: string; email: string; activo: boolean; creado_en: string };

export default function ClientesPage() {
  const [items, setItems] = useState<Cliente[]>([]);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api<{ total: number; items: Cliente[] }>("/clientes?limite=50")
      .then((r) => {
        setItems(r.items);
        setTotal(r.total);
      })
      .catch((e) => setError(e instanceof ApiError ? e.message : "error"));
  }, []);

  return (
    <div>
      <h1>Clientes ({total})</h1>
      {error && <p className="error">{error}</p>}
      <table>
        <thead><tr><th>Nombre</th><th>Email</th><th>Estado</th></tr></thead>
        <tbody>
          {items.map((c) => (
            <tr key={c.id}>
              <td>{c.nombre}</td>
              <td>{c.email}</td>
              <td><span className="badge">{c.activo ? "activo" : "de baja"}</span></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
