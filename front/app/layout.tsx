import { AuthProvider } from "@/lib/auth-context";
import "./globals.css";
import { Metadata } from "next";

export const metadata: Metadata = {
  title: "Chat App",
  description: "Real-time chat application",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body
        suppressHydrationWarning
        className="min-h-screen bg-gray-50 dark:bg-gray-900"
      >
        <AuthProvider>{children}</AuthProvider>
      </body>
    </html>
  );
}
