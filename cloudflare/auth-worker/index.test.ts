import test from 'node:test'
import assert from 'node:assert/strict'

import worker, { handleRequest, parseCookies, tokenFromRequest, verifyJWT } from './index.ts'

const encoder = new TextEncoder()

test('parseCookies parses cookie header values', () => {
  const cookies = parseCookies('a=1; dwizzy_access_token=abc123; other=hello%20world')
  assert.equal(cookies.a, '1')
  assert.equal(cookies.dwizzy_access_token, 'abc123')
  assert.equal(cookies.other, 'hello world')
})

test('tokenFromRequest prefers auth cookie before authorization header', () => {
  const request = new Request('https://auth.dwizzy.my.id/v1/auth/edge/session', {
    headers: {
      Cookie: 'dwizzy_access_token=cookie-token',
      Authorization: 'Bearer header-token',
    },
  })
  assert.equal(tokenFromRequest(request, 'dwizzy_access_token'), 'cookie-token')
})

test('verifyJWT accepts valid hs256 token', async () => {
  const token = await signToken({
    sub: 'user-1',
    sid: 'session-1',
    plan: 'premium',
    roles: ['user'],
    iss: 'dwizzyBRAIN',
    aud: 'dwizzyOS-api',
    iat: Math.floor(Date.now() / 1000) - 10,
    exp: Math.floor(Date.now() / 1000) + 60,
  }, 'secret')

  const claims = await verifyJWT(token, 'secret', 'dwizzyBRAIN', 'dwizzyOS-api')
  assert.equal(claims.sub, 'user-1')
  assert.equal(claims.plan, 'premium')
})

test('edge session returns claims from valid cookie token', async () => {
  const token = await signToken({
    sub: 'user-2',
    sid: 'session-2',
    plan: 'free',
    roles: ['user'],
    iss: 'dwizzyBRAIN',
    aud: 'dwizzyOS-api',
    iat: Math.floor(Date.now() / 1000) - 10,
    exp: Math.floor(Date.now() / 1000) + 60,
  }, 'secret')

  const response = await handleRequest(
    new Request('https://auth.dwizzy.my.id/v1/auth/edge/session', {
      headers: { Cookie: `dwizzy_access_token=${token}` },
    }),
    {
      AUTH_API_ORIGIN: 'https://api.dwizzy.my.id',
      JWT_SECRET: 'secret',
    },
  )

  assert.equal(response.status, 200)
  const payload = await response.json() as { data: { claims: { sub: string } } }
  assert.equal(payload.data.claims.sub, 'user-2')
})

test('worker proxies auth requests to upstream origin and applies cors', async () => {
  const originalFetch = globalThis.fetch
  let forwardedURL = ''
  let forwardedMethod = ''
  let forwardedForwardedFor = ''

  globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
    const req = input instanceof Request ? input : new Request(String(input), init)
    forwardedURL = req.url
    forwardedMethod = req.method
    forwardedForwardedFor = req.headers.get('X-Forwarded-For') || ''
    return new Response(JSON.stringify({ ok: true }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  }

  try {
    const response = await worker.fetch(
      new Request('https://auth.dwizzy.my.id/v1/auth/refresh', {
        method: 'POST',
        headers: {
          Origin: 'https://app.dwizzy.my.id',
          'CF-Connecting-IP': '203.0.113.10',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ refresh_token: 'abc' }),
      }),
      {
        AUTH_API_ORIGIN: 'https://api.dwizzy.my.id',
        JWT_SECRET: 'secret',
        AUTH_ALLOWED_ORIGINS: 'https://app.dwizzy.my.id',
      },
    )

    assert.equal(response.status, 200)
    assert.equal(forwardedURL, 'https://api.dwizzy.my.id/v1/auth/refresh')
    assert.equal(forwardedMethod, 'POST')
    assert.equal(forwardedForwardedFor, '203.0.113.10')
    assert.equal(response.headers.get('Access-Control-Allow-Origin'), 'https://app.dwizzy.my.id')
    assert.equal(response.headers.get('X-Frame-Options'), 'DENY')
  } finally {
    globalThis.fetch = originalFetch
  }
})

async function signToken(claims: Record<string, unknown>, secret: string): Promise<string> {
  const header = { alg: 'HS256', typ: 'JWT' }
  const encodedHeader = encodeBase64URL(JSON.stringify(header))
  const encodedClaims = encodeBase64URL(JSON.stringify(claims))
  const unsigned = `${encodedHeader}.${encodedClaims}`
  const key = await crypto.subtle.importKey(
    'raw',
    encoder.encode(secret),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign'],
  )
  const signature = await crypto.subtle.sign('HMAC', key, encoder.encode(unsigned))
  return `${unsigned}.${encodeBase64URLBytes(new Uint8Array(signature))}`
}

function encodeBase64URL(value: string): string {
  return encodeBase64URLBytes(encoder.encode(value))
}

function encodeBase64URLBytes(value: Uint8Array): string {
  const binary = String.fromCharCode(...value)
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}
