import { useEffect, useMemo, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Loader2, Sparkles, Play } from "lucide-react";
import { ScoreDial } from "./ScoreDial";
import { PromptDiffView } from "./PromptDiffView";
import { OutputCard } from "./OutputCard";
import { RewardMeme } from "./RewardMeme";
import { ShareCard } from "./ShareCard";

interface CompressResult {
  compressed: string;
  originalTokens: number;
  compressedTokens: number;
  savingsPct: number;
  removedSpans: Array<[number, number]>;
}

interface CompareResult {
  bloated: { text: string; relevance: number; conciseness: number; total: number };
  trimmed: { text: string; relevance: number; conciseness: number; total: number };
}

const FILLER_PATTERNS: { label: string; re: RegExp }[] = [
  { label: "as an AI model", re: /\bas an ai (model|assistant|language model)\b/i },
  { label: "please note", re: /\bplease note (that)?\b/i },
  { label: "it is important", re: /\bit is important to (note|remember|mention)\b/i },
  { label: "in order to", re: /\bin order to\b/i },
  { label: "kindly", re: /\bkindly\b/i },
  { label: "I would like to", re: /\bi would like (to|you to)\b/i },
  { label: "very / really", re: /\b(very|really|actually|basically|literally)\b/i },
  { label: "make sure that", re: /\bmake sure (that|to)\b/i },
];

const SAMPLE = `As an AI model, I would kindly like you to please explain, in a very simple way that a 5-year-old could actually understand, the basic concept of quantum physics. It is important to note that you should make sure to avoid technical jargon and, in order to keep it fun, feel free to use analogies involving toys or animals if that helps.`;

export function DietCoach() {
  const [input, setInput] = useState(SAMPLE);
  const [result, setResult] = useState<CompressResult | null>(null);
  const [compressing, setCompressing] = useState(false);
  const [compressError, setCompressError] = useState<string | null>(null);

  const [comparing, setComparing] = useState(false);
  const [comparison, setComparison] = useState<CompareResult | null>(null);
  const [compareError, setCompareError] = useState<string | null>(null);

  const [milestones, setMilestones] = useState<Set<number>>(new Set());
  const [activeMeme, setActiveMeme] = useState<string | null>(null);
  const [memeDataUrl, setMemeDataUrl] = useState<string | null>(null);

  // Debounced compression on input change.
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(() => {
    if (timer.current) clearTimeout(timer.current);
    if (!input.trim()) {
      setResult(null);
      return;
    }
    timer.current = setTimeout(async () => {
      setCompressing(true);
      setCompressError(null);
      try {
        const r = await fetch("/api/compress", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ content: input }),
        });
        const data = await r.json();
        if (!r.ok) throw new Error(data.error ?? "Compression failed");
        setResult(data);
      } catch (e) {
        setCompressError(e instanceof Error ? e.message : "Compression failed");
      } finally {
        setCompressing(false);
      }
    }, 600);
    return () => {
      if (timer.current) clearTimeout(timer.current);
    };
  }, [input]);

  // Milestone rewards.
  useEffect(() => {
    if (!result) return;
    const pct = result.savingsPct;
    for (const m of [50, 70, 90]) {
      if (pct >= m && !milestones.has(m)) {
        setMilestones((prev) => new Set(prev).add(m));
        const prompts: Record<number, string> = {
          50: "Isometric pixel art meme: a chibi robot lifting a barbell labeled 'TOKENS', sweat droplets shaped like tiny green leaves, bright neon colors, meme style, celebrating cutting bloat.",
          70: "Cartoon robot in a yoga pose on a green mat, slimmer waist, glowing halo, meme caption energy, bright colors, celebrating a leaner prompt.",
          90: "A tiny robot in a superhero cape flexing on top of a huge pile of shredded paper labeled 'FILLER', confetti, meme style, gold and green.",
        };
        setActiveMeme(prompts[m]);
        break;
      }
    }
  }, [result, milestones]);

  const fillerHits = useMemo(() => {
    return FILLER_PATTERNS.filter((p) => p.re.test(input));
  }, [input]);

  const runCompare = async () => {
    if (!result || !result.compressed) return;
    setComparing(true);
    setCompareError(null);
    setComparison(null);
    try {
      const r = await fetch("/api/compare", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ original: input, compressed: result.compressed }),
      });
      const data = await r.json();
      if (!r.ok) throw new Error(data.error ?? "Comparison failed");
      setComparison(data);
    } catch (e) {
      setCompareError(e instanceof Error ? e.message : "Comparison failed");
    } finally {
      setComparing(false);
    }
  };

  const saved = result ? result.originalTokens - result.compressedTokens : 0;

  return (
    <div className="mx-auto grid max-w-6xl gap-6 p-4 md:p-8 lg:grid-cols-[1.2fr_1fr]">
      {/* LEFT: editor + score */}
      <div className="space-y-4">
        <Tabs defaultValue="edit">
          <div className="flex items-center justify-between">
            <TabsList>
              <TabsTrigger value="edit">Edit prompt</TabsTrigger>
              <TabsTrigger value="diff">Diff view</TabsTrigger>
            </TabsList>
            {compressing && (
              <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <Loader2 className="h-3 w-3 animate-spin" /> Paritok trimming…
              </span>
            )}
          </div>
          <TabsContent value="edit" className="mt-3">
            <Textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              rows={10}
              placeholder="Paste or type an AI prompt…"
              className="resize-none font-mono text-sm"
            />
          </TabsContent>
          <TabsContent value="diff" className="mt-3">
            <div className="min-h-[240px] rounded-lg border bg-card p-4">
              <PromptDiffView original={input} removedSpans={result?.removedSpans ?? []} />
            </div>
          </TabsContent>
        </Tabs>

        {compressError && (
          <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
            {compressError}
          </div>
        )}

        <ScoreDial savingsPct={result?.savingsPct ?? 0} />

        <div className="grid grid-cols-3 gap-3 text-center">
          <Stat label="Original" value={result?.originalTokens ?? 0} unit="tok" />
          <Stat
            label="Trimmed"
            value={result?.compressedTokens ?? 0}
            unit="tok"
            tone="text-emerald-600"
          />
          <Stat label="Saved" value={saved} unit="tok" tone="text-emerald-600" />
        </div>

        {fillerHits.length > 0 && (
          <div className="rounded-lg border bg-muted/40 p-3">
            <div className="mb-2 flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
              <Sparkles className="h-3 w-3" /> Coach spots filler phrases
            </div>
            <div className="flex flex-wrap gap-1.5">
              {fillerHits.map((f) => (
                <span
                  key={f.label}
                  className="rounded-full bg-amber-500/15 px-2.5 py-1 text-xs font-medium text-amber-700"
                >
                  {f.label}
                </span>
              ))}
            </div>
          </div>
        )}

        <div>
          <Button
            onClick={runCompare}
            disabled={!result?.compressed || comparing}
            className="w-full"
            size="lg"
          >
            {comparing ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" /> Running both prompts through GPT-5.5…
              </>
            ) : (
              <>
                <Play className="mr-2 h-4 w-4" /> Run side-by-side
              </>
            )}
          </Button>
          {compareError && (
            <div className="mt-2 rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
              {compareError}
            </div>
          )}
        </div>

        {comparison && result && (
          <div className="grid gap-3 md:grid-cols-2">
            <OutputCard
              title="🍔 Bloated prompt"
              variant="bloated"
              text={comparison.bloated.text}
              relevance={comparison.bloated.relevance}
              conciseness={comparison.bloated.conciseness}
              total={comparison.bloated.total}
              tokenCount={result.originalTokens}
            />
            <OutputCard
              title="🥗 Trimmed prompt"
              variant="trimmed"
              text={comparison.trimmed.text}
              relevance={comparison.trimmed.relevance}
              conciseness={comparison.trimmed.conciseness}
              total={comparison.trimmed.total}
              tokenCount={result.compressedTokens}
            />
          </div>
        )}
      </div>

      {/* RIGHT: rewards */}
      <div className="space-y-4">
        <div className="rounded-2xl border bg-gradient-to-br from-emerald-500/10 via-transparent to-yellow-400/10 p-5">
          <h2 className="text-sm font-semibold uppercase tracking-widest text-muted-foreground">
            Reward zone
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Hit 50% savings to unlock your first Paritok-branded meme. 70% and 90% unlock more.
          </p>
          <div className="mt-3 flex gap-2 text-xs font-semibold">
            {[50, 70, 90].map((m) => (
              <span
                key={m}
                className={`rounded-full border px-2.5 py-1 ${
                  milestones.has(m)
                    ? "border-emerald-500 bg-emerald-500/15 text-emerald-700"
                    : "border-border bg-muted text-muted-foreground"
                }`}
              >
                {milestones.has(m) ? "✓" : "○"} {m}%
              </span>
            ))}
          </div>
        </div>

        {activeMeme ? (
          <RewardMeme prompt={activeMeme} onReady={setMemeDataUrl} />
        ) : (
          <div className="flex aspect-square items-center justify-center rounded-xl border border-dashed bg-card p-6 text-center text-sm text-muted-foreground">
            Keep trimming — your reward unlocks at 50% savings.
          </div>
        )}

        {result && result.savingsPct >= 50 && (
          <ShareCard
            savedTokens={saved}
            savingsPct={result.savingsPct}
            memeDataUrl={memeDataUrl}
          />
        )}
      </div>
    </div>
  );
}

function Stat({
  label,
  value,
  unit,
  tone,
}: {
  label: string;
  value: number;
  unit: string;
  tone?: string;
}) {
  return (
    <div className="rounded-lg border bg-card p-3">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className={`text-2xl font-bold tabular-nums ${tone ?? ""}`}>
        {value.toLocaleString()}
        <span className="ml-1 text-xs font-normal text-muted-foreground">{unit}</span>
      </div>
    </div>
  );
}
