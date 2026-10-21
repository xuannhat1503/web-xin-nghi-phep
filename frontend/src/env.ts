import { defineEnvVars } from '@sveltejs/kit/env';

function loopbackUrl(value: string | undefined): string {
	const raw = value || 'http://127.0.0.1:8080';
	const url = new URL(raw);
	if (url.protocol !== 'http:' && url.protocol !== 'https:') {
		throw new Error('BACKEND_URL must be http(s)');
	}
	if (!['127.0.0.1', 'localhost', '[::1]'].includes(url.hostname)) {
		// The Go API trusts X-Forwarded-For only from loopback; it must not be exposed elsewhere.
		throw new Error('BACKEND_URL must point to a loopback address');
	}
	return url.origin;
}

export const variables = defineEnvVars({
	BACKEND_URL: {
		description: 'Origin of the Go API (loopback only), e.g. http://127.0.0.1:8080',
		schema: loopbackUrl
	}
});
