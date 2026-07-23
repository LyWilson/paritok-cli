import { createFileRoute } from "@tanstack/react-router";

// Rough token estimate: ~4 chars per token. Used as a fallback + baseline
// for both original & compressed counts so they're always comparable.
function estimateTokens(text: string) {
  if (!text) return 0;
  return Math.max(1, Math.ceil(text.length / 4));
}

// Compute character-offset spans of `original` that were removed to produce `compressed`.
// Uses a simple LCS-token diff at whitespace boundaries so we can highlight what Paritok cut.
function computeRemovedSpans(original: string, compressed: string): Array<[number, number]> {
  if (!original || !compressed) return original ? [[0, original.length]] : [];

  // Split original by whitespace, keeping positions.
  const tokens: { text: string; start: number; end: number }[] = [];
  const re = /\S+/g;
  let m: RegExpExecArray | null;
  while ((m = re.exec(original)) !== null) {
    tokens.push({ text: m[0], start: m.index, end: m.index + m[0].length });
  }

  const compTokens = compressed.match(/\S+/g) ?? [];
  const normalize = (s: string) => s.toLowerCase().replace(/[^\w]/g, "");

  // LCS between tokens (normalized) — mark tokens NOT in the LCS as removed.
  const n = tokens.length,
    k = compTokens.length;
  const dp: number[][] = Array.from({ length: n + 1 }, () => new Array(k + 1).fill(0));
  for (let i = 0; i < n; i++) {
    for (let j = 0; j < k; j++) {
      dp[i + 1][j + 1] =
        normalize(tokens[i].text) === normalize(compTokens[j])
          ? dp[i][j] + 1
          : Math.max(dp[i + 1][j], dp[i][j + 1]);
    }
  }
  const kept = new Set<number>();
  let i = n,
    j = k;
  while (i > 0 && j > 0) {
    if (normalize(tokens[i - 1].text) === normalize(compTokens[j - 1])) {
      kept.add(i - 1);
      i--;
      j--;
    } else if (dp[i - 1][j] >= dp[i][j - 1]) {
      i--;
    } else {
      j--;
    }
  }

  // Coalesce contiguous removed tokens into character-range spans.
  const spans: Array<[number, number]> = [];
  let cur: [number, number] | null = null;
  for (let idx = 0; idx < tokens.length; idx++) {
    if (!kept.has(idx)) {
      const t = tokens[idx];
      if (cur && t.start <= cur[1] + 3) {
        cur[1] = t.end;
      } else {
        if (cur) spans.push(cur);
        cur = [t.start, t.end];
      }
    }
  }
  if (cur) spans.push(cur);
  return spans;
}

export const Route = createFileRoute("/api/compress")({
  server: {
    handlers: {
      POST: async ({ request }) => {
        const key = process.env.PARITOK_API_KEY;
        if (!key) {
          return Response.json({ error: "Missing PARITOK_API_KEY" }, { status: 500 });
        }
        let body: { content?: string; query?: string };
        try {
          body = await request.json();
        } catch {
          return Response.json({ error: "Invalid JSON" }, { status: 400 });
        }
        const content = (body.content ?? "").trim();
        const query = (body.query ?? "").trim();
        if (!content || !query) {
          return Response.json({
            compressed: "",
            originalTokens: estimateTokens(content),
            compressedTokens: 0,
            savingsPct: 0,
            removedSpans: [],
          });
        }

        const resp = await fetch("https://www.paritok.com/api/compress", {
          method: "POST",
          headers: {
            Authorization: `Bearer ${key}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            content,
            query,
            kind: "file_read",
          }),
        });

        if (!resp.ok) {
          const errText = await resp.text().catch(() => "");
          return Response.json(
            { error: `Paritok error [${resp.status}]: ${errText}` },
            { status: resp.status },
          );
        }

        const data = (await resp.json()) as { compressed?: string; gpu_available?: boolean };
        const compressed = data.compressed ?? "";
        const originalTokens = estimateTokens(content);
        const compressedTokens = estimateTokens(compressed);
        const savingsPct =
          originalTokens > 0
            ? Math.max(0, Math.round(((originalTokens - compressedTokens) / originalTokens) * 100))
            : 0;
        const removedSpans = computeRemovedSpans(content, compressed);

        return Response.json({
          compressed,
          originalTokens,
          compressedTokens,
          savingsPct,
          removedSpans,
          gpuAvailable: data.gpu_available ?? false,
        });
      },
    },
  },
});
