export interface Env {
  AUTH_API_ORIGIN: string
  JWT_SECRET: string
  JWT_ISSUER?: string
  JWT_AUDIENCE?: string
  AUTH_ALLOWED_ORIGINS?: string
}

type Claims = {
  sub: string
  sid: string
  plan?: string
  roles?: string[]
  iss: string
  aud: string
  iat: number
  exp: number
}

const ACCESS_COOKIE_NAME = 'dwizzy_access_token'
const REFRESH_COOKIE_NAME = 'dwizzy_refresh_token'
const DEFAULT_ISSUER = 'dwizzyBRAIN'
const DEFAULT_AUDIENCE = 'dwizzyOS-api'

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    return handleRequest(request, env)
  },
}

export async function handleRequest(request: Request, env: Env): Promise<Response> {
  const url = new URL(request.url)
  const origin = request.headers.get('Origin')
  const corsOrigin = selectCorsOrigin(origin, env.AUTH_ALLOWED_ORIGINS ?? '')

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
        { ok: true, service: 'auth-worker' },
        200,
        corsHeaders(corsOrigin),
      ),
    )
  }

  if (url.pathname === '/v1/auth/edge/session') {
    return withSecurityHeaders(await handleEdgeSession(request, env, corsOrigin))
  }

  if (url.pathname.startsWith('/v1/auth/')) {
    return withSecurityHeaders(await proxyAuthRequest(request, env, corsOrigin))
  }

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

async function handleEdgeSession(request: Request, env: Env, corsOrigin: string | null): Promise<Response> {
  const token = tokenFromRequest(request, ACCESS_COOKIE_NAME)
  if (!token) {
    return jsonResponse(
      {
        error: {
          code: 'unauthorized',
          message: 'access token is required',
        },
      },
      401,
      corsHeaders(corsOrigin),
    )
  }

  try {
    const claims = await verifyJWT(
      token,
      env.JWT_SECRET,
      env.JWT_ISSUER || DEFAULT_ISSUER,
      env.JWT_AUDIENCE || DEFAULT_AUDIENCE,
    )
    return jsonResponse(
      {
        data: {
          claims,
        },
      },
      200,
      corsHeaders(corsOrigin),
    )
  } catch {
    return jsonResponse(
      {
        error: {
          code: 'unauthorized',
          message: 'invalid auth token',
        },
      },
      401,
      corsHeaders(corsOrigin),
    )
  }
}

async function proxyAuthRequest(request: Request, env: Env, corsOrigin: string | null): Promise<Response> {
  const upstreamBase = normalizeOrigin(env.AUTH_API_ORIGIN)
  if (!upstreamBase) {
    return jsonResponse(
      {
        error: {
          code: 'service_unavailable',
          message: 'AUTH_API_ORIGIN is not configured',
        },
      },
      503,
      corsHeaders(corsOrigin),
    )
  }

  const url = new URL(request.url)
  const upstreamURL = new URL(url.pathname + url.search, upstreamBase)
  const headers = new Headers(request.headers)
  headers.set('X-Forwarded-Host', url.host)
  headers.set('X-Forwarded-Proto', url.protocol.replace(':', ''))
  headers.set('X-Forwarded-For', request.headers.get('CF-Connecting-IP') || request.headers.get('X-Forwarded-For') || '')
  headers.delete('Host')

  const upstreamInit: RequestInit & { duplex?: 'half' } = {
    method: request.method,
    headers,
    body: shouldForwardBody(request.method) ? request.body : undefined,
    redirect: 'manual',
  }
  if (upstreamInit.body) {
    upstreamInit.duplex = 'half'
  }

  const upstreamRequest = new Request(upstreamURL.toString(), upstreamInit)

  const upstreamResponse = await fetch(upstreamRequest)
  const responseHeaders = new Headers(upstreamResponse.headers)
  applyCorsHeaders(responseHeaders, corsOrigin)
  return new Response(upstreamResponse.body, {
    status: upstreamResponse.status,
    statusText: upstreamResponse.statusText,
    headers: responseHeaders,
  })
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
  headers.set('Access-Control-Allow-Methods', 'GET,POST,OPTIONS')
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
  headers.set('Cache-Control', 'no-store')
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

function normalizeOrigin(value: string | undefined): string {
  const trimmed = value?.trim() || ''
  if (!trimmed) {
    return ''
  }
  return trimmed.endsWith('/') ? trimmed : trimmed + '/'
}

export function tokenFromRequest(request: Request, cookieName: string): string {
  const cookieHeader = request.headers.get('Cookie') || ''
  const cookies = parseCookies(cookieHeader)
  const cookieValue = cookies[cookieName]
  if (cookieValue) {
    return cookieValue
  }

  const authorization = request.headers.get('Authorization') || ''
  if (authorization.toLowerCase().startsWith('bearer ')) {
    return authorization.slice(7).trim()
  }
  return ''
}

export function parseCookies(cookieHeader: string): Record<string, string> {
  const values: Record<string, string> = {}
  for (const item of cookieHeader.split(';')) {
    const part = item.trim()
    if (!part) {
      continue
    }
    const index = part.indexOf('=')
    if (index <= 0) {
      continue
    }
    const key = part.slice(0, index).trim()
    const value = part.slice(index + 1).trim()
    if (!key) {
      continue
    }
    values[key] = decodeURIComponent(value)
  }
  return values
}

export async function verifyJWT(
  token: string,
  secret: string,
  expectedIssuer: string,
  expectedAudience: string,
): Promise<Claims> {
  const parts = token.split('.')
  if (parts.length !== 3) {
    throw new Error('invalid token')
  }

  const [encodedHeader, encodedClaims, encodedSignature] = parts
  const header = JSON.parse(decodeBase64URL(encodedHeader))
  if (header.alg !== 'HS256' || header.typ !== 'JWT') {
    throw new Error('invalid token')
  }

  const unsigned = `${encodedHeader}.${encodedClaims}`
  const key = await crypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(secret),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['verify'],
  )

  const valid = await crypto.subtle.verify(
    'HMAC',
    key,
    base64URLToBytes(encodedSignature),
    new TextEncoder().encode(unsigned),
  )
  if (!valid) {
    throw new Error('invalid token')
  }

  const claims = JSON.parse(decodeBase64URL(encodedClaims)) as Claims
  const now = Math.floor(Date.now() / 1000)
  if (claims.iss !== expectedIssuer || claims.aud !== expectedAudience) {
    throw new Error('invalid token')
  }
  if (!claims.sub || !claims.sid || !claims.exp || claims.exp <= now) {
    throw new Error('invalid token')
  }
  return claims
}

function decodeBase64URL(value: string): string {
  return new TextDecoder().decode(base64URLToBytes(value))
}

function base64URLToBytes(value: string): Uint8Array {
  const base64 = value.replace(/-/g, '+').replace(/_/g, '/')
  const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=')
  const binary = atob(padded)
  return Uint8Array.from(binary, (char) => char.charCodeAt(0))
}

export const INTERNALS = {
  ACCESS_COOKIE_NAME,
  REFRESH_COOKIE_NAME,
  DEFAULT_ISSUER,
  DEFAULT_AUDIENCE,
  selectCorsOrigin,
  normalizeOrigin,
}
