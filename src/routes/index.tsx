import { createFileRoute } from "@tanstack/react-router";
import { DietCoach } from "@/components/DietCoach";

export const Route = createFileRoute("/")({
  head: () => ({
    meta: [
      { title: "Prompt Diet Coach — Trim AI Tokens with Paritok" },
      {
        name: "description",
        content:
          "Gamify prompt efficiency. Watch Paritok trim filler in real time, compare AI outputs side-by-side, and unlock rewards for slimmer prompts.",
      },
      { property: "og:title", content: "Prompt Diet Coach — Trim AI Tokens with Paritok" },
      {
        property: "og:description",
        content:
          "Live token diet coach powered by Paritok. Cut prompt bloat, keep answer quality, unlock meme rewards.",
      },
      { property: "og:type", content: "website" },
      { name: "twitter:card", content: "summary_large_image" },
    ],
  }),
  component: Home,
});

function Home() {
  return (
    <div className="min-h-screen bg-background">
      <header className="border-b bg-gradient-to-r from-emerald-500/10 via-lime-500/5 to-yellow-400/10">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-5 md:px-8">
          <div>
            <h1 className="text-2xl font-bold tracking-tight md:text-3xl">
              Prompt Diet Coach <span aria-hidden>🥗</span>
            </h1>
            <p className="text-sm text-muted-foreground">
              Trim token bloat in real time · powered by{" "}
              <a
                href="https://www.paritok.com"
                target="_blank"
                rel="noreferrer"
                className="font-semibold text-emerald-600 hover:underline"
              >
                Paritok
              </a>
            </p>
          </div>
        </div>
      </header>
      <main>
        <DietCoach />
      </main>
      <footer className="border-t py-6 text-center text-xs text-muted-foreground">
        Built for the Paritok hackathon · Prompts stay in your browser session.
      </footer>
    </div>
  );
}
