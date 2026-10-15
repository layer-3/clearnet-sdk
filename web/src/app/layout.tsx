import type { Metadata, Viewport } from "next";
import { Archivo, JetBrains_Mono } from "next/font/google";
import { AppShell } from "@/components/shell/app-shell";
import { Providers } from "@/components/providers";
import { themeScript, walletHintScript } from "@/lib/boot-scripts";
import "./globals.css";

const archivo = Archivo({ subsets: ["latin"], axes: ["wdth"], variable: "--font-archivo", display: "swap" });
const mono = JetBrains_Mono({ subsets: ["latin"], variable: "--font-jetbrains", display: "swap", weight: ["400", "500"] });

function metadataOrigin(raw: string | undefined): URL | undefined {
  if (!raw) return undefined;
  try {
    const url = new URL(raw);
    if (!["http:", "https:"].includes(url.protocol) || url.username || url.password || url.pathname !== "/" || url.search || url.hash) {
      throw new Error("not a web origin");
    }
    return url;
  } catch {
    throw new Error("NEXT_PUBLIC_SITE_URL must be an absolute HTTP(S) origin without credentials, path or query");
  }
}

const siteOrigin = metadataOrigin(process.env.NEXT_PUBLIC_SITE_URL);

export const metadata: Metadata = {
  title: { default: "Yellow", template: "%s · Yellow" },
  description: "Your account on Yellow Network. Hold, send and track assets that clear in one signature.",
  applicationName: "Yellow",
  ...(siteOrigin && { metadataBase: siteOrigin }),
  openGraph: { title: "Yellow", description: "Your account on Yellow Network.", siteName: "Yellow", type: "website" },
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#F4F4F2" },
    { media: "(prefers-color-scheme: dark)", color: "#000000" },
  ],
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${archivo.variable} ${mono.variable}`} suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript + walletHintScript }} />
      </head>
      <body>
        <Providers>
          <AppShell>{children}</AppShell>
        </Providers>
      </body>
    </html>
  );
}
