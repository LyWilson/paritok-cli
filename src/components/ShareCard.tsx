import { useRef, useState } from "react";
import { toPng } from "html-to-image";
import { Button } from "@/components/ui/button";
import { Download, Twitter } from "lucide-react";

interface Props {
  savedTokens: number;
  savingsPct: number;
  memeDataUrl: string | null;
}

export function ShareCard({ savedTokens, savingsPct, memeDataUrl }: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const [busy, setBusy] = useState(false);

  const download = async () => {
    if (!ref.current) return;
    setBusy(true);
    try {
      const url = await toPng(ref.current, { pixelRatio: 2, cacheBust: true });
      const a = document.createElement("a");
      a.href = url;
      a.download = "prompt-diet-share.png";
      a.click();
    } finally {
      setBusy(false);
    }
  };

  const tweetText = encodeURIComponent(
    `I trimmed ${savedTokens} tokens (${savingsPct}%) with @paritok's Prompt Diet Coach 🥗 — same answer, way less bloat.`,
  );

  return (
    <div className="space-y-3">
      <div
        ref={ref}
        className="relative overflow-hidden rounded-2xl bg-gradient-to-br from-emerald-500 via-lime-500 to-yellow-400 p-6 text-white shadow-xl"
        style={{ width: "100%", maxWidth: 480 }}
      >
        <div className="flex items-start justify-between">
          <div>
            <div className="text-xs font-semibold uppercase tracking-widest opacity-80">
              Prompt Diet Coach
            </div>
            <div className="mt-1 text-5xl font-black tabular-nums drop-shadow">-{savingsPct}%</div>
            <div className="text-lg font-semibold">
              {savedTokens.toLocaleString()} tokens trimmed
            </div>
          </div>
          <div className="rounded-full bg-white/20 px-3 py-1 text-xs font-bold backdrop-blur">
            powered by Paritok
          </div>
        </div>
        {memeDataUrl && (
          <img
            src={memeDataUrl}
            alt="reward"
            className="mt-4 aspect-square w-full rounded-xl border-4 border-white/40 object-cover"
            crossOrigin="anonymous"
          />
        )}
        <div className="mt-4 text-sm font-medium opacity-90">
          Same answer. Way less bloat. Try it →
        </div>
      </div>
      <div className="flex gap-2">
        <Button onClick={download} disabled={busy} size="sm">
          <Download className="mr-2 h-4 w-4" />
          {busy ? "Rendering…" : "Download PNG"}
        </Button>
        <Button asChild size="sm" variant="outline">
          <a
            href={`https://twitter.com/intent/tweet?text=${tweetText}`}
            target="_blank"
            rel="noreferrer"
          >
            <Twitter className="mr-2 h-4 w-4" /> Tweet it
          </a>
        </Button>
      </div>
    </div>
  );
}
