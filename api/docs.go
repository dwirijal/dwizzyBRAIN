package api

import (
	"net/http"
	"strings"
)

const docsHTMLTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>dwizzyBRAIN API Docs</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui.css">
  <style>
    :root {
      color-scheme: dark;
      --bg: #0b1020;
      --panel: #111936;
      --text: #e8ecff;
      --muted: #96a0c8;
      --accent: #73e0a9;
    }
    html, body {
      margin: 0;
      min-height: 100%;
      background:
        radial-gradient(circle at top left, rgba(115, 224, 169, 0.14), transparent 32%%),
        radial-gradient(circle at top right, rgba(113, 132, 255, 0.16), transparent 28%%),
        linear-gradient(180deg, #0b1020 0%%, #080c17 100%%);
      color: var(--text);
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    .hero {
      padding: 28px 24px 12px;
      border-bottom: 1px solid rgba(255,255,255,0.06);
      backdrop-filter: blur(10px);
    }
    .hero h1 {
      margin: 0;
      font-size: 28px;
      letter-spacing: -0.03em;
    }
    .hero p {
      margin: 8px 0 0;
      color: var(--muted);
      max-width: 860px;
      line-height: 1.5;
    }
    .meta {
      margin-top: 10px;
      font-size: 12px;
      color: rgba(232, 236, 255, 0.72);
      letter-spacing: 0.02em;
    }
    .hero a {
      color: var(--accent);
      text-decoration: none;
    }
    #swagger-ui {
      background: rgba(8, 12, 23, 0.72);
      min-height: calc(100vh - 92px);
    }
  </style>
</head>
<body>
  <header class="hero">
    <h1>dwizzyBRAIN API Docs</h1>
    <p>
      Live contract for the market, DeFi, news, and auth APIs. Spec source: <a href="/openapi.json">/openapi.json</a>.
      Discord OAuth and Web3 login share the same auth contract.
    </p>
    <div class="meta">Contract version __OPENAPI_VERSION__ | sha256 __OPENAPI_SHA256__</div>
  </header>
  <main id="swagger-ui"></main>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      window.ui = SwaggerUIBundle({
        url: '/openapi.json',
        dom_id: '#swagger-ui',
        deepLinking: true,
        persistAuthorization: true,
        displayRequestDuration: true,
        docExpansion: 'list',
        defaultModelsExpandDepth: 1,
        syntaxHighlight: {
          activated: true,
          theme: 'agate'
        }
      });
    };
  </script>
</body>
</html>`

var docsHTML = strings.NewReplacer(
	"__OPENAPI_VERSION__", openAPIContractVersion,
	"__OPENAPI_SHA256__", openAPISpecSHA256,
).Replace(docsHTMLTemplate)

func serveDocsHTML(w http.ResponseWriter, r *http.Request) {
	setOpenAPIContractHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(docsHTML))
}
