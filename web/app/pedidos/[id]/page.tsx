"use client";

import { useEffect, useState, useCallback } from "react";
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
type Pago = { id: number; monto_centavos: number; estado: string; metodo: string };
type Envio = { id: number; estado: string; transportista: string; tracking: string };

function moneda(centavos: number) {
  return `$${(centavos / 100).toFixed(2)}`;
}

export default function PedidoDetallePage() {
  const params = useParams<{ id: string }>();
  const [pedido, setPedido] = useState<Pedido | null>(null);
  const [pagos, setPagos] = useState<Pago[]>([]);
  const [envios, setEnvios] = useState<Envio[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [accion, setAccion] = useState<string | null>(null); // mensaje de error de la ULTIMA accion
  const [ocupado, setOcupado] = useState(false);

  const cargar = useCallback(async () => {
    try {
      const [p, ps, es] = await Promise.all([
        api<Pedido>(`/pedidos/${params.id}`),
        api<{ items: Pago[] }>(`/pagos?pedido_id=${params.id}`),
        api<{ items: Envio[] }>(`/envios?pedido_id=${params.id}`),
      ]);
      setPedido(p);
      setPagos(ps.items);
      setEnvios(es.items);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "error");
    }
  }, [params.id]);

  useEffect(() => { cargar(); }, [cargar]);

  async function ejecutar(fn: () => Promise<unknown>) {
    setOcupado(true);
    setAccion(null);
    try {
      await fn();
      await cargar();
    } catch (e) {
      setAccion(e instanceof ApiError ? [e.message, ...(e.detalles ?? [])].join(" — ") : "error desconocido");
    } finally {
      setOcupado(false);
    }
  }

  if (error) return <p className="error">{error}</p>;
  if (!pedido) return <p>Cargando...</p>;

  const pagoActivo = pagos.find((p) => p.estado !== "anulado");
  const envio = envios[0];

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

        {accion && <p className="error">{accion}</p>}

        <div style={{ display: "flex", gap: 8, marginTop: 12 }}>
          {pedido.estado === "borrador" && (
            <button disabled={ocupado} onClick={() => ejecutar(() =>
              api(`/pedidos/${pedido.id}/actualizar-estado`, { method: "POST", body: JSON.stringify({ estado: "confirmado" }) })
            )}>Confirmar pedido</button>
          )}
          {(pedido.estado === "borrador" || pedido.estado === "confirmado") && (
            <button disabled={ocupado} onClick={() => ejecutar(() =>
              api(`/pedidos/${pedido.id}/cancelar`, { method: "POST" })
            )}>Cancelar pedido</button>
          )}
          {pedido.estado === "confirmado" && pagoActivo?.estado === "confirmado" && (
            <button disabled={ocupado} onClick={() => ejecutar(() =>
              api(`/pedidos/${pedido.id}/reembolsar`, { method: "POST" })
            )}>Reembolsar pedido</button>
          )}
        </div>
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

      {pedido.estado === "confirmado" && (
        <div className="card">
          <h2>Pago</h2>
          {!pagoActivo && (
            <button disabled={ocupado} onClick={() => ejecutar(() =>
              api("/pagos", { method: "POST", body: JSON.stringify({ pedido_id: pedido.id, monto_centavos: pedido.total_centavos, metodo: "tarjeta" }) })
            )}>Registrar pago por {moneda(pedido.total_centavos)}</button>
          )}
          {pagoActivo && (
            <p>
              {moneda(pagoActivo.monto_centavos)} — {pagoActivo.metodo} — <span className="badge">{pagoActivo.estado}</span>
              {pagoActivo.estado === "pendiente" && (
                <button style={{ marginLeft: 12 }} disabled={ocupado} onClick={() => ejecutar(() =>
                  api(`/pagos/${pagoActivo.id}/confirmar`, { method: "POST" })
                )}>Confirmar pago</button>
              )}
            </p>
          )}
        </div>
      )}

      {pedido.estado === "confirmado" && (
        <div className="card">
          <h2>Envío</h2>
          {!envio && (
            <EnvioNuevoForm ocupado={ocupado} onCrear={(transportista) => ejecutar(() =>
              api("/envios", { method: "POST", body: JSON.stringify({ pedido_id: pedido.id, transportista }) })
            )} />
          )}
          {envio && envio.estado === "pendiente" && (
            <EnvioTrackingForm ocupado={ocupado} onEnviar={(tracking) => ejecutar(() =>
              api(`/envios/${envio.id}/actualizar-estado`, { method: "POST", body: JSON.stringify({ estado: "en_transito", tracking }) })
            )} />
          )}
          {envio && envio.estado === "en_transito" && (
            <div>
              <p>Tracking: {envio.tracking} — <span className="badge">{envio.estado}</span></p>
              <button disabled={ocupado} onClick={() => ejecutar(() =>
                api(`/envios/${envio.id}/marcar-entregado`, { method: "POST" })
              )}>Marcar entregado</button>
            </div>
          )}
          {envio && envio.estado === "entregado" && <p><span className="badge">entregado</span></p>}
        </div>
      )}
    </div>
  );
}

function EnvioNuevoForm({ ocupado, onCrear }: { ocupado: boolean; onCrear: (transportista: string) => void }) {
  const [transportista, setTransportista] = useState("Servientrega");
  return (
    <div style={{ display: "flex", gap: 8 }}>
      <input value={transportista} onChange={(e) => setTransportista(e.target.value)} placeholder="transportista" />
      <button disabled={ocupado || !transportista} onClick={() => onCrear(transportista)}>Crear envío</button>
    </div>
  );
}

function EnvioTrackingForm({ ocupado, onEnviar }: { ocupado: boolean; onEnviar: (tracking: string) => void }) {
  const [tracking, setTracking] = useState("");
  return (
    <div style={{ display: "flex", gap: 8 }}>
      <input value={tracking} onChange={(e) => setTracking(e.target.value)} placeholder="numero de tracking" />
      <button disabled={ocupado || !tracking} onClick={() => onEnviar(tracking)}>Marcar en tránsito</button>
    </div>
  );
}
