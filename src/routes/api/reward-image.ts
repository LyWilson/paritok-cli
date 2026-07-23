import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/api/reward-image")({
  server: {
    handlers: {
      POST: async () => {
        return new Response(
          JSON.stringify({ error: "Image generation unavailable — NVIDIA endpoint is text-only." }),
          { status: 501, headers: { "Content-Type": "application/json" } },
        );
      },
    },
  },
});
