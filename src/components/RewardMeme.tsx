import { useEffect, useRef, useState } from "react";
import { streamImage } from "@/lib/streamImage";

interface Props {
  prompt: string;
  onReady?: (dataUrl: string) => void;
}

export function RewardMeme({ prompt, onReady }: Props) {
  const [src, setSrc] = useState<string | null>(null);
  const [isFinal, setIsFinal] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const startedFor = useRef<string | null>(null);

  useEffect(() => {
    if (startedFor.current === prompt) return;
    startedFor.current = prompt;
    setSrc(null);
    setIsFinal(false);
    setError(null);
    streamImage("/api/reward-image", prompt, (dataUrl, final) => {
      setSrc(dataUrl);
      if (final) {
        setIsFinal(true);
        onReady?.(dataUrl);
      }
    }).catch((e) => setError(e instanceof Error ? e.message : "Meme failed"));
  }, [prompt, onReady]);

  return (
    <div className="relative overflow-hidden rounded-xl border bg-card">
      {error ? (
        <div className="flex aspect-square items-center justify-center p-6 text-center text-sm text-destructive">
          {error}
        </div>
      ) : src ? (
        <img
          src={src}
          alt="Reward meme"
          className={`aspect-square w-full object-cover transition-[filter] duration-500 ${
            isFinal ? "blur-0" : "blur-2xl"
          }`}
        />
      ) : (
        <div className="flex aspect-square items-center justify-center bg-muted text-sm text-muted-foreground">
          Cooking up your reward meme…
        </div>
      )}
      {!isFinal && !error && src && (
        <div className="absolute bottom-2 left-2 rounded-full bg-background/80 px-2 py-1 text-xs backdrop-blur">
          rendering…
        </div>
      )}
    </div>
  );
}
