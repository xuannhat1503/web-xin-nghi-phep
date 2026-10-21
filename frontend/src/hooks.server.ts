import { error, redirect } from '@sveltejs/kit';
import { sequence, type Handle } from '@sveltejs/kit/hooks';
import { BACKEND_URL } from '$app/env/private';
import { forwardRequestHeaders, isProxied, proxyRequest } from '#lib/server/proxy.ts';
import { guard } from '#lib/server/guard.ts';
import type { MeResponse } from '#lib/types.ts';
import { DEFAULT_LANG, isLang, LANG_COOKIE } from '#lib/i18n/index.ts';

const SESSION_COOKIES = ['elm_session', '__Host-elm_session'];

const securityHeaders: Handle = async ({ event, resolve }) => {
	const cookieLang = event.cookies.get(LANG_COOKIE);
	event.locals.lang = isLang(cookieLang) ? cookieLang : DEFAULT_LANG;
	const response = await resolve(event, {
		transformPageChunk: ({ html }) => html.replace('%lang%', event.locals.lang)
	});
	if (response.headers.get('content-type')?.startsWith('text/html')) {
		response.headers.set('X-Frame-Options', 'DENY');
		response.headers.set('X-Content-Type-Options', 'nosniff');
		response.headers.set('Referrer-Policy', 'strict-origin-when-cross-origin');
		response.headers.set('Permissions-Policy', 'camera=(), microphone=(), geolocation=()');
		// Pages show personal data; never let shared caches keep them.
		response.headers.set('Cache-Control', 'no-store');
	}
	return response;
};

const proxy: Handle = async ({ event, resolve }) => {
	if (isProxied(event.url.pathname)) {
		return proxyRequest(event.request, event.url, event.getClientAddress(), BACKEND_URL);
	}
	return resolve(event);
};

const session: Handle = async ({ event, resolve }) => {
	event.locals.user = null;
	event.locals.isReviewer = false;
	event.locals.unread = 0;
	event.locals.csrfToken = null;
	event.locals.backendUnavailable = false;

	if (SESSION_COOKIES.some((name) => event.cookies.get(name))) {
		try {
			const res = await fetch(BACKEND_URL + '/api/v1/me', {
				headers: forwardRequestHeaders(event.request.headers, event.getClientAddress(), event.url),
				signal: AbortSignal.timeout(5_000)
			});
			if (res.ok) {
				const me = (await res.json()) as MeResponse;
				event.locals.user = me.employee;
				event.locals.isReviewer = me.is_reviewer;
				event.locals.unread = me.unread_notifications ?? 0;
				event.locals.csrfToken = me.csrf_token;
			} else if (res.status !== 401) {
				event.locals.backendUnavailable = true;
			}
		} catch {
			event.locals.backendUnavailable = true;
		}
	}
	return resolve(event);
};

const routeGuard: Handle = async ({ event, resolve }) => {
	const result = guard(event.url, event.locals.user, event.locals.backendUnavailable);
	if (result.kind === 'redirect') redirect(303, result.location);
	if (result.kind === 'error') {
		error(result.status, 'The service is temporarily unavailable. Please try again shortly.');
	}
	return resolve(event);
};

export const handle = sequence(securityHeaders, proxy, session, routeGuard);
