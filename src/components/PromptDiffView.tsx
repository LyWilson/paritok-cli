interface Props {
  original: string;
  removedSpans: Array<[number, number]>;
}

export function PromptDiffView({ original, removedSpans }: Props) {
  if (!original) {
    return (
      <p className="text-sm text-muted-foreground italic">
        Type a prompt above to see what Paritok trims.
      </p>
    );
  }
  if (removedSpans.length === 0) {
    return <p className="whitespace-pre-wrap text-sm leading-relaxed">{original}</p>;
  }

  const parts: React.ReactNode[] = [];
  let cursor = 0;
  const sorted = [...removedSpans].sort((a, b) => a[0] - b[0]);
  sorted.forEach(([start, end], i) => {
    if (start > cursor) {
      parts.push(<span key={`k-${i}`}>{original.slice(cursor, start)}</span>);
    }
    parts.push(
      <mark
        key={`r-${i}`}
        className="rounded bg-destructive/15 px-0.5 text-destructive line-through decoration-destructive/60"
      >
        {original.slice(start, end)}
      </mark>,
    );
    cursor = end;
  });
  if (cursor < original.length) {
    parts.push(<span key="tail">{original.slice(cursor)}</span>);
  }
  return <p className="whitespace-pre-wrap text-sm leading-relaxed">{parts}</p>;
}
