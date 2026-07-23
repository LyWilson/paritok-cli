interface Props {
  savingsPct: number;
}

function grade(pct: number) {
  if (pct >= 80) return { letter: "A+", tone: "text-emerald-500" };
  if (pct >= 60) return { letter: "A", tone: "text-emerald-500" };
  if (pct >= 40) return { letter: "B", tone: "text-lime-500" };
  if (pct >= 20) return { letter: "C", tone: "text-amber-500" };
  return { letter: "F", tone: "text-destructive" };
}

export function ScoreDial({ savingsPct }: Props) {
  const g = grade(savingsPct);
  return (
    <div className="flex items-center gap-4 rounded-xl border bg-card p-4 shadow-sm">
      <div
        className={`flex h-16 w-16 items-center justify-center rounded-full border-4 ${g.tone.replace("text-", "border-")} text-2xl font-bold ${g.tone}`}
      >
        {g.letter}
      </div>
      <div className="flex-1">
        <div className="flex items-baseline justify-between">
          <span className="text-sm font-medium text-muted-foreground">Token diet score</span>
          <span className={`text-2xl font-bold tabular-nums ${g.tone}`}>{savingsPct}%</span>
        </div>
        <div className="mt-2 h-2 w-full overflow-hidden rounded-full bg-muted">
          <div
            className="h-full bg-gradient-to-r from-lime-400 to-emerald-500 transition-all duration-500"
            style={{ width: `${Math.min(100, savingsPct)}%` }}
          />
        </div>
      </div>
    </div>
  );
}
