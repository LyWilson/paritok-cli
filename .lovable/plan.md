
# Prompt Diet Coach

Single-page web app: users type a prompt, Paritok trims it live, a hybrid coach shows exactly what was removed (diff), side-by-side LLM outputs are scored objectively, and shareable Paritok-branded memes unlock at savings milestones.

## Stack
- TanStack Start (existing template), Tailwind + shadcn tokens
- Paritok `/api/compress` for compression + token metrics + `removedSpans` (`PARITOK_API_KEY` as secret)
- Lovable AI Gateway: `openai/gpt-5.5` for side-by-side outputs and evaluator scoring; `google/gemini-3-pro-image` (streaming) for meme rewards
- No DB, no auth — single-session React state only

## Routes / files
- `src/routes/index.tsx` — replace placeholder with full Diet Coach UI + SEO head()
- `src/routes/api/compress.ts` — POST → Paritok, returns `{ compressed, originalTokens, compressedTokens, savingsPct, removedSpans }` (character-offset ranges into the original text)
- `src/routes/api/compare.ts` — POST → parallel `openai/gpt-5.5` calls for original and compressed prompts, then a third `openai/gpt-5.5` call scoring each output for **relevance (0–100)** and **conciseness (0–100)** vs the original intent; returns `{ bloated: {text, relevance, conciseness, total}, trimmed: {...} }`
- `src/routes/api/reward-image.ts` — streams `google/gemini-3-pro-image` SSE (per `ai-image-generation-tanstack`)
- `src/lib/streamImage.ts` — canonical SSE parser with `flushSync` and blur-on-partial
- `src/components/DietCoach.tsx` — main UI
- `src/components/PromptDiffView.tsx` — overlays `removedSpans` as struck-through `<mark>` behind textarea (toggle: edit / diff view)
- `src/components/ScoreDial.tsx` — animated savings dial + letter grade
- `src/components/OutputCard.tsx` — one side of the comparison, with output text + score chips
- `src/components/RewardMeme.tsx` — streaming meme
- `src/components/ShareCard.tsx` — Paritok-branded downloadable PNG (html2canvas) with meme + savings + app URL

## UX flow
```text
┌──────────────────────────────────────────────┐
│  Prompt Diet Coach 🥗   powered by Paritok   │
├──────────────────────────────────────────────┤
│  [ textarea ]      [ Edit | Diff ] toggle    │
│  Diff: ~~As an AI model, I must state~~      │
│        Quantum physics is...                 │
│                                              │
│  Original 412 tok · Trimmed 138 tok          │
│  Savings ████████░░ 66%  Grade A             │
│                                              │
│  [ Run side-by-side ▶ ]                      │
│  ┌─── Bloated ────┐  ┌─── Trimmed ────┐      │
│  │ output…        │  │ output…        │      │
│  │ Score 72/100   │  │ Score 89/100   │      │
│  │ Rel 78 Conc 66 │  │ Rel 91 Conc 87 │      │
│  └────────────────┘  └────────────────┘      │
│                                              │
│  🎉 50%+ → [meme streams in]                 │
│  [ Download share card ] [ Copy tweet ]      │
└──────────────────────────────────────────────┘
```

## Behavior
1. **Live compression** — debounced 500ms → `/api/compress`. Update tokens, savings bar, grade (F<20, C<40, B<60, A<80, A+≥80).
2. **Hybrid coach / diff view** — client-side regex chips flag common filler ("as an AI model", "please note", "in order to", "it is important to"); toggling **Diff** view overlays Paritok's actual `removedSpans` as struck-through highlights on the original text — proves semantic (not keyword) trimming.
3. **Side-by-side + objective scores** — `Promise.all` runs both prompts through `openai/gpt-5.5`; a third gpt-5.5 call scores both outputs on relevance to the original intent and conciseness (structured JSON output). Totals shown as chips beneath each card.
4. **Reward** — first time savings ≥50%, auto-stream a Paritok-branded meme:
   *"Isometric pixel art: a chibi robot lifting a barbell labeled 'TOKENS', sweat droplets are tiny Paritok logos, bright colors, meme style."*
   Extra milestones (70%, 90%) unlock additional memes.
5. **Share card** — html2canvas snapshot combining meme + "I trimmed X tokens (Y%) with Paritok → [app URL]"; download PNG + copy pre-filled tweet text.

## Secrets
- `PARITOK_API_KEY` — request via `add_secret` secure form (never hardcode the pasted key).
- `LOVABLE_API_KEY` — ensure via `ai_gateway--create`.

## Out of scope
- No accounts, persistence, or leaderboard
- No streaming for the chat comparison (non-streaming keeps parallel calls + scoring simple)
- No custom evaluator model — reuse gpt-5.5 as judge instead of flan-t5 (Paritok's `/api/compress` is the only Paritok endpoint used; adding a separate GPU-eval endpoint isn't documented in the provided spec)

## SEO
`index.tsx` head(): title "Prompt Diet Coach — Trim AI Tokens with Paritok", matching description, og/twitter tags.
