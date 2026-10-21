/**
 * Same-origin reverse proxy: the browser only talks to SvelteKit; /api, /auth and /health go to the Go API.
 * Headers are forwarded by allowlist so nothing unexpected (e.g. a spoofed X-Forwarded-For) reaches Go.
 */

export const PROXY_PREFIXES = ['/api/', '/auth/', '/health/'] as const;

export const MAX_PROXY_BODY_BYTES = 1 << 20;

const FORWARD_REQUEST_HEADERS = [
	'accept',
	'accept-language',
	'content-type',
	'cookie',
	'idempotency-key',
	'if-match',
	'origin',
	'user-agent',
	'x-csrf-token',
	'x-request-id'
];

const DROP_RESPONSE_HEADERS = new Set([
	'connection',
	'content-encoding',
	'content-length',
	'keep-alive',
	'set-cookie',
	'transfer-encoding'
]);

export function isProxied(pathname: string): boolean {
	return PROXY_PREFIXES.some((p) => pathname.startsWith(p));
}

/** Builds the headers sent to Go. X-Forwarded-For is replaced, never appended to. */
export function forwardRequestHeaders(
	src: Headers,
	clientAddress: string,
	publicUrl: URL
): Headers {
	const out = new Headers();
	for (const name of FORWARD_REQUEST_HEADERS) {
		const v = src.get(name);
		if (v !== null) out.set(name, v);
	}
	out.set('x-forwarded-for', clientAddress);
	out.set('x-forwarded-proto', publicUrl.protocol.replace(':', ''));
	out.set('x-forwarded-host', publicUrl.host);
	return out;
}

/** Copies Go's response headers, preserving every Set-Cookie individually. */
export function copyResponseHeaders(src: Headers): Headers {
	const out = new Headers();
	src.forEach((value, name) => {
		if (!DROP_RESPONSE_HEADERS.has(name.toLowerCase())) out.set(name, value);
	});
	for (const cookie of src.getSetCookie()) out.append('set-cookie', cookie);
	return out;
}

export async function proxyRequest(
	request: Request,
	url: URL,
	clientAddress: string,
	backendUrl: string
): Promise<Response> {
	const declared = Number(request.headers.get('content-length') ?? '0');
	if (declared > MAX_PROXY_BODY_BYTES) {
		return jsonError(413, 'payload_too_large', 'Request body is too large');
	}
	let body: ArrayBuffer | undefined;
	if (request.method !== 'GET' && request.method !== 'HEAD') {
		body = await request.arrayBuffer();
		if (body.byteLength > MAX_PROXY_BODY_BYTES) {
			return jsonError(413, 'payload_too_large', 'Request body is too large');
		}
	}

	let upstream: Response;
	try {
		upstream = await fetch(backendUrl + url.pathname + url.search, {
			method: request.method,
			headers: forwardRequestHeaders(request.headers, clientAddress, url),
			body,
			redirect: 'manual',
			signal: AbortSignal.timeout(30_000)
		});
	} catch {
		return jsonError(502, 'backend_unavailable', 'The service is temporarily unavailable');
	}
	return new Response(upstream.body, {
		status: upstream.status,
		statusText: upstream.statusText,
		headers: copyResponseHeaders(upstream.headers)
	});
}

function jsonError(status: number, code: string, message: string): Response {
	return new Response(JSON.stringify({ error: { code, message } }), {
		status,
		headers: { 'content-type': 'application/json', 'cache-control': 'no-store' }
	});
}
