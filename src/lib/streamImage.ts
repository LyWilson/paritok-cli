import { flushSync } from "react-dom";

type ImageEvent =
  | { type: "image_generation.partial_image"; b64_json: string; partial_image_index: number }
  | { type: "image_generation.completed"; b64_json: string }
  | { type: "error"; error: { message: string } };

// Minimal SSE parser sufficient for the gateway (single-line data: frames).
export async function streamImage(
  endpoint: string,
  prompt: string,
  onFrame: (dataUrl: string, isFinal: boolean) => void,
): Promise<void> {
  const res = await fetch(endpoint, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ prompt }),
  });
  if (!res.ok || !res.body) {
    throw new Error(`Image request failed: ${res.status} ${await res.text().catch(() => "")}`);
  }
  const reader = res.body.pipeThrough(new TextDecoderStream()).getReader();
  let buf = "";
  let sawCompleted = false;
  let streamError: string | undefined;
  let currentEvent: string | undefined;

  const handleEvent = (eventName: string | undefined, dataStr: string) => {
    if (!dataStr) return;
    let payload: ImageEvent | undefined;
    try {
      payload = JSON.parse(dataStr) as ImageEvent;
    } catch {
      return;
    }
    if (eventName === "error" || payload.type === "error") {
      streamError =
        (payload as { error?: { message?: string } }).error?.message ?? "Image generation failed";
      return;
    }
    if (
      payload.type !== "image_generation.partial_image" &&
      payload.type !== "image_generation.completed"
    ) {
      return;
    }
    const isFinal = payload.type === "image_generation.completed";
    flushSync(() => {
      onFrame(`data:image/png;base64,${payload!.b64_json}`, isFinal);
    });
    if (isFinal) sawCompleted = true;
  };

  try {
    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      buf += value;
      let idx: number;
      // Split on SSE record boundaries (blank line).
      while ((idx = buf.indexOf("\n\n")) !== -1) {
        const record = buf.slice(0, idx);
        buf = buf.slice(idx + 2);
        let evt: string | undefined;
        let data = "";
        for (const line of record.split("\n")) {
          if (line.startsWith("event:")) evt = line.slice(6).trim();
          else if (line.startsWith("data:")) data += (data ? "\n" : "") + line.slice(5).trim();
        }
        currentEvent = evt ?? currentEvent;
        if (data === "[DONE]") continue;
        handleEvent(evt, data);
      }
    }
  } finally {
    reader.cancel().catch(() => {});
  }
  if (streamError) throw new Error(streamError);
  if (!sawCompleted) throw new Error("Stream ended without completed frame");
}
