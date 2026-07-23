# Prompt Diet Coach

An AI-powered prompt optimization tool that compresses verbose prompts into lean, effective versions — then scores both to show you the savings.

[![Built with Paritok](https://img.shields.io/badge/Built%20with-Paritok-1f2d3d)](https://github.com/Paritok-official/paritok-4b-v1)

## What it does

1. **Paste a prompt** — drop in any AI prompt you want to optimize
2. **Compress** — sends it through [Paritok](https://github.com/Paritok-official/paritok-4b-v1) to strip bloat while preserving intent
3. **Compare** — runs both the original and compressed versions through a judge LLM
4. **Score** — see relevance and conciseness scores for both versions side by side

## Tech Stack

- [TanStack Start](https://tanstack.com/start) — React meta-framework
- [React 19](https://react.dev) — UI
- [Tailwind CSS 4](https://tailwindcss.com) — styling
- [shadcn/ui](https://ui.shadcn.com) — component library
- [Paritok](https://github.com/Paritok-official/paritok-4b-v1) — prompt compression API
- [NVIDIA OpenAI endpoint](https://build.nvidia.com) — LLM judge

## Getting Started

Requires [Node.js](https://nodejs.org) 18+.

```sh
git clone https://github.com/LyWilson/prompt-diet-coach.git
cd prompt-diet-coach
npm install
```

Create a `.env` file:

```
PARITOK_API_KEY=your-paritok-key
NVIDIA_API_KEY=your-nvidia-key
```

Run the dev server:

```sh
npm run dev
```

## License

[Apache 2.0](LICENSE)

## Built with

- [Paritok](https://github.com/Paritok-official/paritok-4b-v1) — prompt compression engine
