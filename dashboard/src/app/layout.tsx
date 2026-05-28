import type { Metadata } from "next";
import "./globals.css";
import { Providers } from "@/components/providers";
import { TechBackground } from "@/components/tech-background";

export const metadata: Metadata = {
  title: "Sub2API Dashboard",
  description: "API usage dashboard for Sub2API",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="h-full antialiased">
      <body className="min-h-full flex flex-col font-sans text-slate-800">
        <TechBackground />
        <Providers>
          <div className="relative flex min-h-full flex-1 flex-col">
            {children}
          </div>
        </Providers>
      </body>
    </html>
  );
}
