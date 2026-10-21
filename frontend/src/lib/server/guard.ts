import type { CurrentEmployee } from '#lib/types.ts';

/**
 * UI routing rules only. These decide which page to show; the Go API enforces every permission itself.
 */
export type GuardResult =
	{ kind: 'allow' } | { kind: 'redirect'; location: string } | { kind: 'error'; status: 503 };

const PUBLIC_PATHS = new Set(['/login']);
// Reachable by everyone, signed in or not (language preference).
const OPEN_PATHS = new Set(['/lang']);

export function guard(
	url: URL,
	user: CurrentEmployee | null,
	backendUnavailable = false
): GuardResult {
	const path = url.pathname;
	const home = user ? (user.active ? homeOf(user) : '/pending-activation') : '/login';

	if (OPEN_PATHS.has(path)) return { kind: 'allow' };
	if (PUBLIC_PATHS.has(path)) {
		return user ? { kind: 'redirect', location: home } : { kind: 'allow' };
	}
	if (backendUnavailable) return { kind: 'error', status: 503 };
	if (path === '/') return { kind: 'redirect', location: home };
	if (!user) {
		const returnTo = path + url.search;
		return { kind: 'redirect', location: '/login?return_to=' + encodeURIComponent(returnTo) };
	}
	if (path === '/pending-activation') {
		return user.active ? { kind: 'redirect', location: home } : { kind: 'allow' };
	}
	if (!user.active) return { kind: 'redirect', location: '/pending-activation' };
	// Role checks (e.g. /admin) live in route layouts so the 403 renders inside the app shell.
	return { kind: 'allow' };
}

/** Landing page after sign-in: Admins land on the admin dashboard (their own leave pages stay in the menu); everyone else on theirs. */
export function homeOf(user: CurrentEmployee): string {
	return user.role === 'ADMIN' ? '/admin/dashboard' : '/dashboard';
}
