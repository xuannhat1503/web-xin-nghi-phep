import { error, fail, type RequestEvent } from '@sveltejs/kit';
import { BACKEND_URL } from '$app/env/private';
import { forwardRequestHeaders } from './proxy.ts';
import type { Msg } from '#lib/i18n/index.ts';

export interface ApiError {
	code: string;
	message: string;
	fields?: Record<string, string>;
}

export type ApiResult<T> =
	{ ok: true; status: number; data: T } | { ok: false; status: number; error: ApiError };

/**
 * Calls the Go API from server code on behalf of the signed-in user: forwards the session cookie,
 * the app Origin and the CSRF token. Authorization is decided by Go, never here.
 */
export async function api<T>(
	event: RequestEvent,
	method: string,
	path: string,
	body?: unknown,
	extraHeaders: Record<string, string> = {}
): Promise<ApiResult<T>> {
	const headers = forwardRequestHeaders(event.request.headers, event.getClientAddress(), event.url);
	headers.set('accept', 'application/json');
	headers.set('origin', event.url.origin);
	if (event.locals.csrfToken) headers.set('x-csrf-token', event.locals.csrfToken);
	if (body !== undefined) headers.set('content-type', 'application/json');
	headers.delete('if-match');
	headers.delete('idempotency-key');
	for (const [k, v] of Object.entries(extraHeaders)) headers.set(k, v);

	let res: Response;
	try {
		res = await fetch(BACKEND_URL + path, {
			method,
			headers,
			body: body === undefined ? undefined : JSON.stringify(body),
			signal: AbortSignal.timeout(15_000)
		});
	} catch {
		return {
			ok: false,
			status: 503,
			error: { code: 'unavailable', message: 'The service is temporarily unavailable.' }
		};
	}
	const text = await res.text();
	let json: unknown;
	try {
		json = text ? JSON.parse(text) : null;
	} catch {
		json = null;
	}
	if (res.ok) return { ok: true, status: res.status, data: json as T };
	const err = (json as { error?: ApiError } | null)?.error;
	return {
		ok: false,
		status: res.status,
		error: err ?? { code: 'error', message: 'Something went wrong. Please try again.' }
	};
}

/** Loads data or throws the matching SvelteKit error page. */
export async function load<T>(event: RequestEvent, path: string): Promise<T> {
	const res = await api<T>(event, 'GET', path);
	if (!res.ok) error(res.status === 401 ? 403 : res.status, res.error.message);
	return res.data;
}

/** Converts an API result into a form-action result (success message or fail()). */
export function actionResult<T>(
	res: ApiResult<T>,
	success: Msg,
	extra: Record<string, unknown> = {}
) {
	if (res.ok) return { success, ...extra };
	return fail(res.status, {
		error: res.error.message,
		code: res.error.code,
		fields: res.error.fields ?? {},
		...extra
	});
}

/** Reads a form field as a trimmed string. */
export function str(form: FormData, name: string): string {
	const v = form.get(name);
	return typeof v === 'string' ? v.trim() : '';
}

export function int(form: FormData, name: string): number {
	const n = Number(str(form, name));
	return Number.isFinite(n) ? Math.trunc(n) : NaN;
}
