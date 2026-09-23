"use client";

import { useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api";

type TopProducto = { producto_id: number; nombre: string; unidades_vendidas: number };
type StockBajo = { producto_id: number; nombre: string; stock_total: number };

export default function ReportesPage() {
  const [top, setTop] = useState<TopProducto[] | null>(null);
  const [stockBajo, setStockBajo] = useState<StockBajo[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([
      api<{ items: TopProducto[] }>("/reportes/top-productos?limite=5"),
      api<{ items: StockBajo[] }>("/reportes/stock-bajo"),
    ])
      .then(([t, s]) => {
        setTop(t.items);
        setStockBajo(s.items);
      })
      .catch((e) => setError(e instanceof ApiError ? e.message : "error"));
  }, []);

  if (error) {
    return (
      <div>
        <h1>Reportes</h1>
        <p className="error">{error}</p>
        <p>Los reportes son visibles solo para el rol <span className="badge">admin</span>.</p>
      </div>
    );
  }

  return (
    <div>
      <h1>Reportes</h1>
      <div className="card">
        <h2>Top productos</h2>
        <table>
          <thead><tr><th>Producto</th><th>Unidades vendidas</th></tr></thead>
          <tbody>
            {top?.map((p) => (
              <tr key={p.producto_id}><td>{p.nombre}</td><td>{p.unidades_vendidas}</td></tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="card">
        <h2>Stock bajo</h2>
        <table>
          <thead><tr><th>Producto</th><th>Stock total</th></tr></thead>
          <tbody>
            {stockBajo?.map((p) => (
              <tr key={p.producto_id}><td>{p.nombre}</td><td>{p.stock_total}</td></tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
