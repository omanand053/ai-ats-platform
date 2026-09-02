import { NextRequest, NextResponse } from "next/server";

type JsonResponse = Record<string, unknown>;

type OpenRouterResponse = {
  choices?: unknown;
  reply?: unknown;
  answer?: unknown;
  data?: unknown;
  error?: unknown;
  message?: unknown;
};

const OPENROUTER_URL = "https://openrouter.ai/api/v1/chat/completions";
const OPENROUTER_FALLBACK_PATH = "/api/v1/llm/gemini";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isString(value: unknown): value is string {
  return typeof value === "string";
}

function parseJson<T>(text: string): T | null {
  if (!text) return null;
  try {
    return JSON.parse(text) as T;
  } catch {
    return null;
  }
}

function shouldFallbackOpenRouter(status: number, body: OpenRouterResponse | null, rawBody: string) {
  if (status >= 500 || status === 429 || status === 408) {
    return true;
  }
  if (status === 401 || status === 403) {
    return true;
  }
  if (status === 0) {
    return true;
  }
  if (!rawBody) {
    return true;
  }
  return false;
}

async function callGeminiFallback(request: NextRequest, prompt: string) {
  const fallbackUrl = new URL(OPENROUTER_FALLBACK_PATH, request.url).toString();

  const response = await fetch(fallbackUrl, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ prompt }),
  });

  const responseBody = await response.text();
  const data = parseJson<OpenRouterResponse>(responseBody);

  if (!response.ok) {
    let errorMessage: string | null = null;

    if (isRecord(data)) {
      const errorValue = data.error;
      if (isRecord(errorValue) && isString(errorValue.message)) {
        errorMessage = errorValue.message;
      } else if (isString(errorValue)) {
        errorMessage = errorValue;
      } else if (isString(data.message)) {
        errorMessage = data.message;
      }
    }

    return NextResponse.json(
      { error: String(errorMessage ?? responseBody ?? "Gemini fallback failed.") },
      { status: response.status }
    );
  }

  let reply: string | null = null;
  if (isRecord(data)) {
    if (isString(data.reply)) reply = data.reply;
    else if (isRecord(data.data) && isString(data.data.reply)) reply = data.data.reply;
    else if (isRecord(data.data) && isString(data.data.answer)) reply = data.data.answer;
    else if (isString(data.answer)) reply = data.answer;
  }

  const chatResp = {
    id: "gemini-fallback",
    object: "chat.completion",
    choices: [
      {
        index: 0,
        message: {
          role: "assistant",
          content: reply ?? "",
        },
        finish_reason: "stop",
      },
    ],
  };

  return NextResponse.json(chatResp, { status: 200 });
}

export async function POST(request: NextRequest) {
  const OPENROUTER_API_KEY = process.env.OPENROUTER_API_KEY?.trim();
  const OPENROUTER_MODEL = process.env.OPENROUTER_MODEL?.trim() || "gpt-4o-mini";
  const apiKeyLoaded = Boolean(OPENROUTER_API_KEY);

  const body = (await request.json().catch(() => null)) as unknown;
  const prompt = isRecord(body) && isString(body.prompt) ? body.prompt.trim() : "";

  if (!prompt) {
    return NextResponse.json(
      { error: "Invalid request body. Expected { prompt: string }." },
      { status: 400 }
    );
  }

  if (!apiKeyLoaded) {
    return callGeminiFallback(request, prompt);
  }

  try {
    const response = await fetch(OPENROUTER_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${OPENROUTER_API_KEY}`,
      },
      body: JSON.stringify({
        model: OPENROUTER_MODEL,
        messages: [{ role: "user", content: prompt }],
        temperature: 0.2,
      }),
    });

    const responseBody = await response.text();
    const data = parseJson<OpenRouterResponse>(responseBody);

    if (!response.ok || shouldFallbackOpenRouter(response.status, data, responseBody)) {
      return callGeminiFallback(request, prompt);
    }

    if (Array.isArray(data?.choices)) {
      return NextResponse.json(data, { status: response.status });
    }

    let reply: string | null = null;
    if (isRecord(data)) {
      if (isString(data.reply)) reply = data.reply;
      else if (isRecord(data.data) && isString(data.data.reply)) reply = data.data.reply;
      else if (isRecord(data.data) && isString(data.data.answer)) reply = data.data.answer;
      else if (isString(data.answer)) reply = data.answer;
    }

    const chatResp = {
      id: "openrouter-response",
      object: "chat.completion",
      choices: [
        {
          index: 0,
          message: {
            role: "assistant",
            content: reply ?? responseBody,
          },
          finish_reason: "stop",
        },
      ],
    };

    return NextResponse.json(chatResp, { status: response.status });
  } catch {
    return callGeminiFallback(request, prompt);
  }
}
