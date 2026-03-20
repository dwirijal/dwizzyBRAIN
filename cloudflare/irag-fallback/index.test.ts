import test from 'node:test'
import assert from 'node:assert/strict'

import worker, { handleRequest } from './index.ts'

test('healthz reports fallback configuration state', async () => {
  const response = await handleRequest(new Request('https://irag-fallback.example/healthz'), {
    IRAG_PRIMARY_ORIGIN: 'https://primary.example',
    IRAG_SECONDARY_ORIGIN: 'https://secondary.example',
    IRAG_TIMEOUT_MS: '2500',
  })

  assert.equal(response.status, 200)
  const payload = (await response.json()) as {
    service: string
    primary_configured: boolean
    secondary_configured: boolean
    timeout_ms: number
  }
  assert.equal(payload.service, 'irag-fallback')
  assert.equal(payload.primary_configured, true)
  assert.equal(payload.secondary_configured, true)
  assert.equal(payload.timeout_ms, 2500)
})

test('worker proxies requests to primary upstream and forwards headers', async () => {
  const originalFetch = globalThis.fetch
  let forwardedURL = ''
  let forwardedForwardedFor = ''
  let forwardedHost = ''

  globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
    const req = input instanceof Request ? input : new Request(String(input), init)
    forwardedURL = req.url
    forwardedForwardedFor = req.headers.get('X-Forwarded-For') || ''
    forwardedHost = req.headers.get('Host') || ''
    return new Response(JSON.stringify({ ok: true }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  }

  try {
    const response = await worker.fetch(
      new Request('https://api2.dwizzy.my.id/v1/tools/ping?x=1', {
        method: 'POST',
        headers: {
          Origin: 'https://app.dwizzy.my.id',
          'CF-Connecting-IP': '203.0.113.20',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ hello: 'world' }),
      }),
      {
        IRAG_PRIMARY_ORIGIN: 'https://primary.example',
        IRAG_ORIGIN_HOST_HEADER: 'api.dwizzy.my.id',
        IRAG_ALLOWED_ORIGINS: 'https://app.dwizzy.my.id',
      },
    )

    assert.equal(response.status, 200)
    assert.equal(forwardedURL, 'https://primary.example/v1/tools/ping?x=1')
    assert.equal(forwardedForwardedFor, '203.0.113.20')
    assert.equal(forwardedHost, 'api.dwizzy.my.id')
    assert.equal(response.headers.get('Access-Control-Allow-Origin'), 'https://app.dwizzy.my.id')
    assert.equal(response.headers.get('X-IRAG-Upstream'), 'https://primary.example')
    assert.equal(response.headers.get('X-IRAG-Fallback-Used'), 'false')
  } finally {
    globalThis.fetch = originalFetch
  }
})

test('worker falls back to secondary origin on retryable primary status', async () => {
  const originalFetch = globalThis.fetch
  const called: string[] = []

  globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
    const req = input instanceof Request ? input : new Request(String(input), init)
    called.push(req.url)
    if (req.url.startsWith('https://primary.example/')) {
      return new Response(JSON.stringify({ error: 'temporary' }), {
        status: 503,
        headers: { 'Content-Type': 'application/json' },
      })
    }
    return new Response(JSON.stringify({ ok: true, source: 'secondary' }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  }

  try {
    const response = await handleRequest(
      new Request('https://irag-fallback.example/v1/ai/text/qwen'),
      {
        IRAG_PRIMARY_ORIGIN: 'https://primary.example',
        IRAG_SECONDARY_ORIGIN: 'https://secondary.example',
      },
    )

    assert.equal(response.status, 200)
    assert.deepEqual(called, [
      'https://primary.example/v1/ai/text/qwen',
      'https://primary.example/v1/ai/text/qwen',
      'https://primary.example/v1/ai/text/qwen',
      'https://secondary.example/v1/ai/text/qwen',
    ])
    assert.equal(response.headers.get('X-IRAG-Upstream'), 'https://secondary.example')
    assert.equal(response.headers.get('X-IRAG-Fallback-Used'), 'true')
  } finally {
    globalThis.fetch = originalFetch
  }
})

test('worker does not fall back on non-retryable upstream status', async () => {
  const originalFetch = globalThis.fetch
  let calls = 0

  globalThis.fetch = async () => {
    calls += 1
    return new Response(JSON.stringify({ error: 'not found' }), {
      status: 404,
      headers: { 'Content-Type': 'application/json' },
    })
  }

  try {
    const response = await handleRequest(
      new Request('https://irag-fallback.example/v1/tools/unknown'),
      {
        IRAG_PRIMARY_ORIGIN: 'https://primary.example',
        IRAG_SECONDARY_ORIGIN: 'https://secondary.example',
      },
    )

    assert.equal(response.status, 404)
    assert.equal(calls, 1)
    assert.equal(response.headers.get('X-IRAG-Upstream'), 'https://primary.example')
    assert.equal(response.headers.get('X-IRAG-Fallback-Used'), 'false')
  } finally {
    globalThis.fetch = originalFetch
  }
})

test('worker returns 503 when primary origin is missing', async () => {
  const response = await handleRequest(
    new Request('https://irag-fallback.example/v1/tools/ping'),
    {
      IRAG_PRIMARY_ORIGIN: '',
    },
  )

  assert.equal(response.status, 503)
  const payload = (await response.json()) as { error: { code: string } }
  assert.equal(payload.error.code, 'service_unavailable')
})

test('worker sustains concurrent proxy requests', async () => {
  const originalFetch = globalThis.fetch
  let calls = 0

  globalThis.fetch = async (input: RequestInfo | URL) => {
    const req = input instanceof Request ? input : new Request(String(input))
    if (!req.url.startsWith('https://primary.example/')) {
      throw new Error(`unexpected upstream: ${req.url}`)
    }
    calls += 1
    await new Promise((resolve) => setTimeout(resolve, 10))
    return new Response(JSON.stringify({ ok: true }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  }

  try {
    const requests = Array.from({ length: 50 }, (_, index) =>
      handleRequest(
        new Request(`https://irag-fallback.example/v1/tools/ping?i=${index}`, {
          method: 'GET',
        }),
        {
          IRAG_PRIMARY_ORIGIN: 'https://primary.example',
          IRAG_ALLOWED_ORIGINS: 'https://app.dwizzy.my.id',
        },
      ),
    )

    const responses = await Promise.all(requests)
    assert.equal(responses.length, 50)
    for (const response of responses) {
      assert.equal(response.status, 200)
      assert.equal(response.headers.get('X-IRAG-Upstream'), 'https://primary.example')
      assert.equal(response.headers.get('X-IRAG-Fallback-Used'), 'false')
    }
    assert.equal(calls, 50)
  } finally {
    globalThis.fetch = originalFetch
  }
})
