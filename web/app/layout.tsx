import type { Metadata } from "next";
import Link from "next/link";
import LogoutButton from "./logout-button";
import "./globals.css";

export const metadata: Metadata = {
  title: "Back-office",
  description: "Panel de administracion del back-office",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="es">
      <body>
        <nav>
          <Link href="/">Dashboard</Link>
          <Link href="/productos">Productos</Link>
          <Link href="/pedidos">Pedidos</Link>
          <Link href="/clientes">Clientes</Link>
          <Link href="/reportes">Reportes</Link>
          <LogoutButton />
        </nav>
        <main>{children}</main>
      </body>
    </html>
  );
}
