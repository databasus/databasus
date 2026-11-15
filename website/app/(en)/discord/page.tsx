import type { Metadata } from "next";
import Image from "next/image";

const DISCORD_INVITE_URL = "https://discord.gg/cP8h8EwEzK";

export const metadata: Metadata = {
  title: "Databasus Discord",
  description: "Join the Databasus chat on Discord.",
  robots: { index: false, follow: false },
};

export default function DiscordPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-[#0F1115] px-4 text-center">
      <meta httpEquiv="refresh" content={`0; url=${DISCORD_INVITE_URL}`} />

      <a href="/" className="mb-8 flex items-center">
        <Image
          src="/logo.svg"
          alt="Databasus logo"
          width={50}
          height={50}
          className="shrink-0"
          priority
        />

        <div className="ml-4 select-none text-2xl font-bold text-white">
          Databasus
        </div>
      </a>

      <h1 className="mb-3 text-2xl font-bold text-white md:text-3xl">
        Join the Databasus Discord
      </h1>

      <p className="mb-6 max-w-md text-base text-gray-400">
        If the redirect did not work, open the invite with the button below.
      </p>

      <a
        href={DISCORD_INVITE_URL}
        className="rounded-lg bg-white px-5 py-2.5 text-sm font-semibold text-black transition-opacity hover:opacity-70 md:px-6 md:py-3 md:text-base"
      >
        Open Discord
      </a>
    </div>
  );
}
