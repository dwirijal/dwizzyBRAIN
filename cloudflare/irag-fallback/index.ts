export interface Env {
  IRAG_PRIMARY_ORIGIN: string
  IRAG_SECONDARY_ORIGIN?: string
  IRAG_ORIGIN_HOST_HEADER?: string
  IRAG_RESOLVE_OVERRIDE?: string
  IRAG_ALLOWED_ORIGINS?: string
  IRAG_TIMEOUT_MS?: string
}

type AttemptResult = {
  response?: Response
  error?: Error
  retryable: boolean
  origin: string
}

const DEFAULT_TIMEOUT_MS = 4000

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    return handleRequest(request, env)
  },
}

export async function handleRequest(request: Request, env: Env): Promise<Response> {
  const url = new URL(request.url)
  const origin = request.headers.get('Origin')
  const corsOrigin = selectCorsOrigin(origin, env.IRAG_ALLOWED_ORIGINS ?? '')

  if (request.method === 'OPTIONS') {
    return withSecurityHeaders(
      new Response(null, {
        status: 204,
        headers: corsHeaders(corsOrigin),
      }),
    )
  }

  if (url.pathname === '/healthz') {
    return withSecurityHeaders(
      jsonResponse(
        {
          ok: true,
          service: 'irag-fallback',
          primary_configured: Boolean(normalizeOrigin(env.IRAG_PRIMARY_ORIGIN)),
          secondary_configured: Boolean(normalizeOrigin(env.IRAG_SECONDARY_ORIGIN)),
          timeout_ms: parseTimeoutMS(env.IRAG_TIMEOUT_MS),
        },
        200,
        corsHeaders(corsOrigin),
      ),
    )
  }

  if (!url.pathname.startsWith('/v1/')) {
    return withSecurityHeaders(
      jsonResponse(
        {
          error: {
            code: 'not_found',
            message: 'route not found',
          },
        },
        404,
        corsHeaders(corsOrigin),
      ),
    )
  }

  return withSecurityHeaders(await proxyWithFallback(request, env, corsOrigin))
}

async function proxyWithFallback(request: Request, env: Env, corsOrigin: string | null): Promise<Response> {
  const primaryOrigin = normalizeOrigin(env.IRAG_PRIMARY_ORIGIN)
  if (!primaryOrigin) {
    return jsonResponse(
      {
        error: {
          code: 'service_unavailable',
          message: 'IRAG_PRIMARY_ORIGIN is not configured',
        },
      },
      503,
      corsHeaders(corsOrigin),
    )
  }

  const secondaryOrigin = normalizeOrigin(env.IRAG_SECONDARY_ORIGIN)
  const timeoutMs = parseTimeoutMS(env.IRAG_TIMEOUT_MS)
  const originHostHeader = normalizeHostHeader(env.IRAG_ORIGIN_HOST_HEADER)
  const resolveOverride = normalizeHostHeader(env.IRAG_RESOLVE_OVERRIDE)
  const bodyBuffer = shouldForwardBody(request.method) ? await request.arrayBuffer() : undefined

  const primaryResult = await attemptProxyWithRetries(
    request,
    primaryOrigin,
    originHostHeader,
    resolveOverride,
    bodyBuffer,
    timeoutMs,
    3,
  )
  if (!shouldFallback(primaryResult, primaryOrigin, secondaryOrigin)) {
    if (primaryResult.response) {
      return decorateProxyResponse(primaryResult.response, corsOrigin, primaryOrigin, false)
    }
    return upstreamUnavailable(corsOrigin)
  }

  const secondaryResult = await attemptProxyWithRetries(
    request,
    secondaryOrigin,
    originHostHeader,
    resolveOverride,
    bodyBuffer,
    timeoutMs,
    2,
  )
  if (secondaryResult.response) {
    return decorateProxyResponse(secondaryResult.response, corsOrigin, secondaryOrigin, true)
  }

  if (primaryResult.response) {
    return decorateProxyResponse(primaryResult.response, corsOrigin, primaryOrigin, false)
  }

  return upstreamUnavailable(corsOrigin)
}

async function attemptProxyWithRetries(
  request: Request,
  upstreamOrigin: string,
  originHostHeader: string,
  resolveOverride: string,
  bodyBuffer: ArrayBuffer | undefined,
  timeoutMs: number,
  attempts: number,
): Promise<AttemptResult> {
  let result: AttemptResult = {
    retryable: true,
    origin: upstreamOrigin,
  }

  for (let attempt = 1; attempt <= attempts; attempt++) {
    result = await attemptProxy(
      request,
      upstreamOrigin,
      originHostHeader,
      resolveOverride,
      bodyBuffer,
      timeoutMs,
    )
    if (!result.retryable) {
      return result
    }
    if (attempt < attempts) {
      await sleep(100 * attempt)
    }
  }

  return result
}

function shouldFallback(result: AttemptResult, primaryOrigin: string, secondaryOrigin: string): boolean {
  if (!secondaryOrigin || secondaryOrigin === primaryOrigin) {
    return false
  }
  return result.retryable
}

async function attemptProxy(
  request: Request,
  upstreamOrigin: string,
  originHostHeader: string,
  resolveOverride: string,
  bodyBuffer: ArrayBuffer | undefined,
  timeoutMs: number,
): Promise<AttemptResult> {
  try {
    const response = await fetchWithTimeout(
      buildUpstreamRequest(request, upstreamOrigin, originHostHeader, bodyBuffer),
      resolveOverride,
      timeoutMs,
    )
    return {
      response,
      retryable: isRetryableStatus(response.status),
      origin: upstreamOrigin,
    }
  } catch (error) {
    return {
      error: error instanceof Error ? error : new Error('upstream request failed'),
      retryable: true,
      origin: upstreamOrigin,
    }
  }
}

function buildUpstreamRequest(
  request: Request,
  upstreamOrigin: string,
  originHostHeader: string,
  bodyBuffer: ArrayBuffer | undefined,
): Request {
  const requestURL = new URL(request.url)
  const upstreamURL = new URL(requestURL.pathname + requestURL.search, upstreamOrigin)
  const headers = new Headers(request.headers)
  headers.set('X-Forwarded-Host', requestURL.host)
  headers.set('X-Forwarded-Proto', requestURL.protocol.replace(':', ''))
  headers.set(
    'X-Forwarded-For',
    request.headers.get('CF-Connecting-IP') || request.headers.get('X-Forwarded-For') || '',
  )
  headers.set('X-IRAG-Edge-Proxy', 'cloudflare')
  headers.delete('Host')
  if (originHostHeader) {
    headers.set('Host', originHostHeader)
  }

  return new Request(upstreamURL.toString(), {
    method: request.method,
    headers,
    body: bodyBuffer,
    redirect: 'follow',
  })
}

async function fetchWithTimeout(
  request: Request,
  resolveOverride: string,
  timeoutMs: number,
): Promise<Response> {
  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(new Error('upstream timeout')), timeoutMs)
  try {
    const init: RequestInit & { cf?: { resolveOverride?: string } } = {
      signal: controller.signal,
    }
    if (resolveOverride) {
      init.cf = { resolveOverride }
    }
    return await fetch(request, init)
  } finally {
    clearTimeout(timeout)
  }
}

function shouldForwardBody(method: string): boolean {
  switch (method.toUpperCase()) {
    case 'GET':
    case 'HEAD':
      return false
    default:
      return true
  }
}

function isRetryableStatus(status: number): boolean {
  return status === 408 || status === 429 || status >= 500
}

function upstreamUnavailable(corsOrigin: string | null): Response {
  return jsonResponse(
    {
      error: {
        code: 'bad_gateway',
        message: 'IRAG upstream unavailable',
      },
    },
    502,
    corsHeaders(corsOrigin),
  )
}

function decorateProxyResponse(
  upstreamResponse: Response,
  corsOrigin: string | null,
  upstreamOrigin: string,
  fallbackUsed: boolean,
): Response {
  const responseHeaders = new Headers(upstreamResponse.headers)
  applyCorsHeaders(responseHeaders, corsOrigin)
  responseHeaders.set('X-IRAG-Upstream', trimTrailingSlash(upstreamOrigin))
  responseHeaders.set('X-IRAG-Fallback-Used', String(fallbackUsed))
  return new Response(upstreamResponse.body, {
    status: upstreamResponse.status,
    statusText: upstreamResponse.statusText,
    headers: responseHeaders,
  })
}

function selectCorsOrigin(requestOrigin: string | null, configured: string): string | null {
  const origin = requestOrigin?.trim()
  if (!origin) {
    return null
  }

  const allowed = configured
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)

  if (allowed.length === 0) {
    return origin
  }

  if (allowed.includes(origin)) {
    return origin
  }

  return null
}

function corsHeaders(origin: string | null): HeadersInit {
  const headers = new Headers()
  applyCorsHeaders(headers, origin)
  return headers
}

function applyCorsHeaders(headers: Headers, origin: string | null): void {
  if (!origin) {
    return
  }

  headers.set('Access-Control-Allow-Origin', origin)
  headers.set('Vary', appendVary(headers.get('Vary'), 'Origin'))
  headers.set('Access-Control-Allow-Credentials', 'true')
  headers.set('Access-Control-Allow-Methods', 'GET,POST,PUT,PATCH,DELETE,OPTIONS')
  headers.set('Access-Control-Allow-Headers', 'Content-Type, Authorization')
}

function appendVary(current: string | null, value: string): string {
  const parts = (current || '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
  if (!parts.includes(value)) {
    parts.push(value)
  }
  return parts.join(', ')
}

function withSecurityHeaders(response: Response): Response {
  const headers = new Headers(response.headers)
  headers.set('X-Content-Type-Options', 'nosniff')
  headers.set('X-Frame-Options', 'DENY')
  headers.set('Referrer-Policy', 'strict-origin-when-cross-origin')
  headers.set('Cross-Origin-Resource-Policy', 'same-site')
  headers.set(
    'Content-Security-Policy',
    "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'",
  )
  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers,
  })
}

function jsonResponse(payload: unknown, status: number, extraHeaders?: HeadersInit): Response {
  const headers = new Headers(extraHeaders)
  headers.set('Content-Type', 'application/json; charset=utf-8')
  return new Response(JSON.stringify(payload), { status, headers })
}

function parseTimeoutMS(value: string | undefined): number {
  const parsed = Number.parseInt(value || '', 10)
  if (Number.isFinite(parsed) && parsed > 0) {
    return parsed
  }
  return DEFAULT_TIMEOUT_MS
}

function normalizeHostHeader(value: string | undefined): string {
  return value?.trim() || ''
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function normalizeOrigin(value: string | undefined): string {
  const trimmed = value?.trim() || ''
  if (!trimmed) {
    return ''
  }
  return trimmed.endsWith('/') ? trimmed : trimmed + '/'
}

function trimTrailingSlash(value: string): string {
  if (value.endsWith('/')) {
    return value.slice(0, -1)
  }
  return value
}
