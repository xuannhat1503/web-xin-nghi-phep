import type { MessageKey } from '#lib/i18n/index.ts';

const CODES = [
	'access_denied',
	'login_expired',
	'provider_unavailable',
	'login_failed',
	'not_allowed'
] as const;

/** Maps the generic error code from the auth callback to a message key. Unknown codes get the generic text. */
export function loginErrorKey(code: string | null): MessageKey | null {
	if (!code) return null;
	const c = (CODES as readonly string[]).includes(code) ? code : 'login_failed';
	return `login.err.${c}` as MessageKey;
}

/** Builds the Google sign-in link. The Go API re-validates return_to (same-site paths only). */
export function loginHref(returnTo: string | null): string {
	return '/auth/google/login' + (returnTo ? '?return_to=' + encodeURIComponent(returnTo) : '');
}
