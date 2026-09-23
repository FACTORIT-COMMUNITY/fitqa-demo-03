"use client";

import { useEffect, useState } from "react";
import { api, ApiError, getToken } from "@/lib/api";
import { useRouter } from "next/navigation";

type Me = { usuario_id: number; email: string; rol: string };

export default function DashboardPage() {
  const router = useRouter();
  const [me, setMe] = useState<Me | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!getToken()) {
      router.push("/login");
      return;
    }
    api<Me>("/auth/me")
      .then(setMe)
      .catch((e) => setError(e instanceof ApiError ? e.message : "error"));
  }, [router]);

  return (
    <div>
      <h1>Dashboard</h1>
      {error && <p className="error">{error}</p>}
      {me && (
        <div className="card">
          <p>Sesion activa: <strong>{me.email}</strong> <span className="badge">{me.rol}</span></p>
        </div>
      )}
      <div className="card">
        <p>Este es el panel liviano del back-office. La mayoria de la superficie del
           sistema vive en la API; este panel cubre los flujos de consulta mas comunes.</p>
      </div>
    </div>
  );
}
