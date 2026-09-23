"use client";

import { useRouter } from "next/navigation";
import { api, clearTokens, getToken } from "@/lib/api";

export default function LogoutButton() {
  const router = useRouter();

  async function cerrarSesion() {
    if (getToken()) {
      await api("/auth/logout", { method: "POST" }).catch(() => {});
    }
    clearTokens();
    router.push("/login");
  }

  return (
    <button
      onClick={cerrarSesion}
      style={{ marginLeft: "auto", background: "transparent", border: "1px solid #cfd4ff", color: "#cfd4ff" }}
    >
      Cerrar sesión
    </button>
  );
}
