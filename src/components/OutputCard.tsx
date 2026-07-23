import ReactMarkdown from "react-markdown";

interface Props {
  title: string;
  variant: "bloated" | "trimmed";
  text: string;
  relevance: number;
  conciseness: number;
  total: number;
  tokenCount: number;
}

export function OutputCard({ title, variant, text, relevance, conciseness, total, tokenCount }: Props) {
  const ring = variant === "trimmed" ? "ring-2 ring-emerald-500/40" : "ring-1 ring-border";
  const badge =
    variant === "trimmed"
      ? "bg-emerald-500/10 text-emerald-600"
      : "bg-muted text-muted-foreground";
  return (
    <div className={`flex flex-col rounded-xl border bg-card p-4 ${ring}`}>
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">{title}</h3>
        <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${badge}`}>
          {tokenCount} tok
        </span>
      </div>
      <div className="prose prose-sm dark:prose-invert mt-3 max-h-72 overflow-auto text-sm leading-relaxed">
        <ReactMarkdown>{text}</ReactMarkdown>
      </div>
      <div className="mt-3 flex flex-wrap gap-2 border-t pt-3">
        <Chip label="Relevance" value={relevance} />
        <Chip label="Conciseness" value={conciseness} />
        <Chip label="Total" value={total} strong />
      </div>
    </div>
  );
}

function Chip({ label, value, strong }: { label: string; value: number; strong?: boolean }) {
  const tone =
    value >= 80
      ? "bg-emerald-500/15 text-emerald-600"
      : value >= 60
        ? "bg-lime-500/15 text-lime-700"
        : value >= 40
          ? "bg-amber-500/15 text-amber-700"
          : "bg-destructive/15 text-destructive";
  return (
    <span
      className={`rounded-full px-2.5 py-1 text-xs ${tone} ${strong ? "font-bold" : "font-medium"}`}
    >
      {label} {value}
    </span>
  );
}
