import { type NextRequest, NextResponse } from "next/server";

// Disable Next.js response caching — required for SSE streaming to work.
export const dynamic = "force-dynamic";

const API_BASE = process.env.API_URL ?? "http://api:8080";

const HOP_BY_HOP = new Set([
  "connection",
  "keep-alive",
  "proxy-authenticate",
  "proxy-authorization",
  "te",
  "trailers",
  "transfer-encoding",
  "upgrade",
  "host",
]);

async function proxy(
  req: NextRequest,
  { params }: { params: Promise<{ path: string[] }> },
): Promise<NextResponse> {
  const { path } = await params;
  const search = req.nextUrl.search;
  const url = `${API_BASE}/api/v1/${path.join("/")}${search}`;

  const forwardHeaders: Record<string, string> = {};
  req.headers.forEach((value, key) => {
    if (!HOP_BY_HOP.has(key.toLowerCase())) {
      forwardHeaders[key] = value;
    }
  });

  const hasBody = req.method !== "GET" && req.method !== "HEAD";

  const upstream = await fetch(url, {
    method: req.method,
    headers: forwardHeaders,
    body: hasBody ? await req.arrayBuffer() : undefined,
    // @ts-expect-error — Node.js fetch supports duplex for streaming
    duplex: hasBody ? "half" : undefined,
  });

  const responseHeaders: Record<string, string> = {};
  upstream.headers.forEach((value, key) => {
    if (!HOP_BY_HOP.has(key.toLowerCase())) {
      responseHeaders[key] = value;
    }
  });

  return new NextResponse(upstream.body, {
    status: upstream.status,
    headers: responseHeaders,
  });
}

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const DELETE = proxy;
export const PATCH = proxy;
