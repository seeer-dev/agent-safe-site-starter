// /api/* -> Go API pass-through.
//
// This Function is the only edge compute the architecture permits (see
// specs/changes/edge-api-proxy and the AGENTS.md hard-boundary note). It
// exists so browsers never learn the API origin: every Pages project that
// deploys this repo (storefront and admin alike) serves /api/* same-origin,
// forwards it to env.API_ORIGIN, and stamps the hop credential
// env.EDGE_SECRET that the origin requires.
//
// It holds no routing, caching, or business logic. Its only transforms are
// credentials: the hop secret, and an opaque per-client key for the shared
// abuse limiter (see specs/changes/prelaunch-security-hardening).

const EDGE_CLIENT_KEY_HEADER = "X-Edge-Client-Key";
// Domain separation tag: the derived key is scoped to the abuse limiter so
// the same secret cannot mint identities for another contract.
const EDGE_CLIENT_KEY_DOMAIN = "edge-client-key/v1";

function hex(buffer) {
  return [...new Uint8Array(buffer)].map((b) => b.toString(16).padStart(2, "0")).join("");
}

// deriveClientKey turns Cloudflare's edge-observed client address into an
// opaque HMAC-SHA256 identifier. The raw address is never forwarded and
// never logged; the origin can only correlate equal keys, not recover IPs.
// Returns "" when the edge provides no address or no secret is configured —
// the origin then fails the protected route closed.
async function deriveClientKey(request, secret) {
  const addr = request.headers.get("CF-Connecting-IP");
  if (!secret || !addr) return "";
  const enc = new TextEncoder();
  const key = await crypto.subtle.importKey(
    "raw",
    enc.encode(EDGE_CLIENT_KEY_DOMAIN + "" + secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  return hex(await crypto.subtle.sign("HMAC", key, enc.encode(addr)));
}

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

  // A caller may supply the identity header itself; the only value allowed
  // to reach the origin is the one derived here.
  upstream.headers.delete(EDGE_CLIENT_KEY_HEADER);
  const clientKey = await deriveClientKey(request, env.EDGE_SECRET);
  if (clientKey) {
    upstream.headers.set(EDGE_CLIENT_KEY_HEADER, clientKey);
  }

  return fetch(upstream);
}
