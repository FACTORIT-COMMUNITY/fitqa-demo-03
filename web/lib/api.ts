// Cliente HTTP minimo del panel: guarda el access token en localStorage (no hay cookies
// de sesion, el back no las usa) y redirige a /login si una llamada vuelve 401.
"use client";

const BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem("access_token");
}

export function setTokens(access: string, refresh: string) {
  window.localStorage.setItem("access_token", access);
  window.localStorage.setItem("refresh_token", refresh);
}

export function clearTokens() {
  window.localStorage.removeItem("access_token");
  window.localStorage.removeItem("refresh_token");
}

export class ApiError extends Error {
  status: number;
  detalles?: string[];
  constructor(status: number, mensaje: string, detalles?: string[]) {
    super(mensaje);
    this.status = status;
    this.detalles = detalles;
  }
}

export async function api<T>(path: string, opciones: RequestInit = {}): Promise<T> {
  const token = getToken();
  const headers = new Headers(opciones.headers);
  headers.set("Content-Type", "application/json");
  if (token) headers.set("Authorization", `Bearer ${token}`);

  const res = await fetch(`${BASE_URL}${path}`, { ...opciones, headers });

  if (res.status === 401 && typeof window !== "undefined") {
    clearTokens();
    window.location.href = "/login";
    throw new ApiError(401, "sesion expirada");
  }

  if (res.status === 204) return undefined as T;

  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new ApiError(res.status, body.error ?? "error desconocido", body.detalles);
  }
  return body as T;
}
