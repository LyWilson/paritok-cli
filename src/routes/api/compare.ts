import { createFileRoute } from "@tanstack/react-router";
import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "https://integrate.api.nvidia.com/v1",
  apiKey: process.env.NVIDIA_API_KEY,
});
const MODEL = "openai/gpt-oss-20b";

async function chat(prompt: string): Promise<string> {
  const r = await client.chat.completions.create({
    model: MODEL,
    messages: [{ role: "user", content: prompt }],
    temperature: 1,
    top_p: 1,
    max_tokens: 4096,
    stream: false,
  });
  return r.choices?.[0]?.message?.content ?? "";
}

async function judge(
  originalIntent: string,
  outputA: string,
  outputB: string,
): Promise<{
  bloated: { relevance: number; conciseness: number };
  trimmed: { relevance: number; conciseness: number };
}> {
  const r = await client.chat.completions.create({
    model: MODEL,
    messages: [
      {
        role: "system",
        content:
          "You are a strict AI-output evaluator. Rate two answers to the same user intent on relevance (0-100, how well it addresses the intent) and conciseness (0-100, higher = leaner without losing meaning). Respond with JSON only.",
      },
      {
        role: "user",
        content: `Original user intent:\n"""\n${originalIntent}\n"""\n\nAnswer A (bloated prompt output):\n"""\n${outputA}\n"""\n\nAnswer B (trimmed prompt output):\n"""\n${outputB}\n"""\n\nReturn JSON: {"bloated":{"relevance":<0-100>,"conciseness":<0-100>},"trimmed":{"relevance":<0-100>,"conciseness":<0-100>}}`,
      },
    ],
    response_format: { type: "json_object" },
    temperature: 1,
    top_p: 1,
    max_tokens: 4096,
    stream: false,
  });
  const raw = r.choices?.[0]?.message?.content ?? "{}";
  try {
    const parsed = JSON.parse(raw);
    return {
      bloated: {
        relevance: Number(parsed.bloated?.relevance) || 0,
        conciseness: Number(parsed.bloated?.conciseness) || 0,
      },
      trimmed: {
        relevance: Number(parsed.trimmed?.relevance) || 0,
        conciseness: Number(parsed.trimmed?.conciseness) || 0,
      },
    };
  } catch {
    return {
      bloated: { relevance: 0, conciseness: 0 },
      trimmed: { relevance: 0, conciseness: 0 },
    };
  }
}

export const Route = createFileRoute("/api/compare")({
  server: {
    handlers: {
      POST: async ({ request }) => {
        if (!process.env.NVIDIA_API_KEY) {
          return Response.json({ error: "Missing NVIDIA_API_KEY" }, { status: 500 });
        }

        const body = (await request.json()) as {
          original?: string;
          compressed?: string;
          intent?: string;
        };
        const original = (body.original ?? "").trim();
        const compressed = (body.compressed ?? "").trim();
        const intent = (body.intent ?? original).trim();
        if (!original || !compressed) {
          return Response.json({ error: "original and compressed required" }, { status: 400 });
        }

        try {
          const [bloatedText, trimmedText] = await Promise.all([
            chat(original),
            chat(compressed),
          ]);
          const scores = await judge(intent, bloatedText, trimmedText);
          const total = (r: number, c: number) => Math.round(r * 0.7 + c * 0.3);
          return Response.json({
            bloated: {
              text: bloatedText,
              relevance: scores.bloated.relevance,
              conciseness: scores.bloated.conciseness,
              total: total(scores.bloated.relevance, scores.bloated.conciseness),
            },
            trimmed: {
              text: trimmedText,
              relevance: scores.trimmed.relevance,
              conciseness: scores.trimmed.conciseness,
              total: total(scores.trimmed.relevance, scores.trimmed.conciseness),
            },
          });
        } catch (e) {
          return Response.json(
            { error: e instanceof Error ? e.message : "Comparison failed" },
            { status: 500 },
          );
        }
      },
    },
  },
});
