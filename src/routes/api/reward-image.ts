import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/api/reward-image")({
  server: {
    handlers: {
      POST: async ({ request }) => {
        const key = process.env.LOVABLE_API_KEY;
        if (!key) return new Response("Missing LOVABLE_API_KEY", { status: 500 });

        const { prompt } = (await request.json()) as { prompt?: string };
        const finalPrompt =
          prompt?.trim() ||
          "Isometric pixel art meme: a chibi robot lifting a barbell labeled 'TOKENS', sweat droplets shaped like tiny leaves, bright colors, meme style, celebrating token savings.";

        const upstream = await fetch(
          "https://ai.gateway.lovable.dev/v1/images/generations",
          {
            method: "POST",
            headers: { Authorization: `Bearer ${key}`, "Content-Type": "application/json" },
            body: JSON.stringify({
              model: "google/gemini-3-pro-image",
              messages: [{ role: "user", content: finalPrompt }],
              modalities: ["image", "text"],
              stream: true,
            }),
          },
        );

        if (!upstream.ok || !upstream.body) {
          return new Response(await upstream.text().catch(() => "image error"), {
            status: upstream.status || 500,
          });
        }
        return new Response(upstream.body, {
          headers: { "Content-Type": "text/event-stream", "Cache-Control": "no-cache" },
        });
      },
    },
  },
});
