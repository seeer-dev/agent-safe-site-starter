// /api/* -> Go API pass-through.
//
// This Function is the only edge compute the architecture permits (see
// specs/changes/edge-api-proxy and the AGENTS.md hard-boundary note). It
// exists so browsers never learn the API origin: every Pages project that
// deploys this repo (storefront and admin alike) serves /api/* same-origin,
// forwards it to env.API_ORIGIN, and stamps the hop credential
// env.EDGE_SECRET that the origin requires.
//
// It holds no logic — no routing tables, no caching, no transforms beyond
// the credential. If a request needs different handling, that belongs in
// the Go API, not here.

export async function onRequest(context) {
  const { request, env } = context;
  const origin = (env.API_ORIGIN || "").replace(/\/+$/, "");
  if (!origin) {
    // Fail closed: a missing upstream is a deployment misconfiguration,
    // never a reason to fall through to static assets.
    return new Response("API_ORIGIN is not configured", { status: 502 });
  }

  const url = new URL(request.url);
  const target = new URL(url.pathname + url.search, origin);

  // new Request(target, request) preserves method, body (streamed), and
  // headers while retargeting the URL.
  const upstream = new Request(target, request);
  if (env.EDGE_SECRET) {
    upstream.headers.set("X-Edge-Secret", env.EDGE_SECRET);
  }
  // The API never reads cookies; do not leak first-party site cookies to
  // the upstream hop.
  upstream.headers.delete("Cookie");

  return fetch(upstream);
}
