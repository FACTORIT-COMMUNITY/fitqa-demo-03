"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { setTokens } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("ana.rodriguez@backoffice.demo");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [cargando, setCargando] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setCargando(true);
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });
      const body = await res.json();
      if (!res.ok) throw new Error(body.error ?? "credenciales invalidas");
      setTokens(body.access_token, body.refresh_token);
      router.push("/");
    } catch (err) {
      setError(err instanceof Error ? err.message : "error desconocido");
    } finally {
      setCargando(false);
    }
  }

  return (
    <div className="card" style={{ maxWidth: 360, margin: "60px auto" }}>
      <h1>Back-office</h1>
      <form onSubmit={onSubmit}>
        <p>
          <label>Email<br />
            <input value={email} onChange={(e) => setEmail(e.target.value)} style={{ width: "100%" }} />
          </label>
        </p>
        <p>
          <label>Password<br />
            <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} style={{ width: "100%" }} />
          </label>
        </p>
        {error && <p className="error">{error}</p>}
        <button type="submit" disabled={cargando}>{cargando ? "Ingresando..." : "Ingresar"}</button>
      </form>
    </div>
  );
}
