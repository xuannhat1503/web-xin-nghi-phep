import { api } from '#lib/server/api.ts';
import type { Catalog } from '#lib/admin-types.ts';
import type { LayoutServerLoad } from './$types';

const EMPTY: Catalog = { leave_types: [], reasons: [] };

// The CSRF token is meant to be readable by page code (synchronizer pattern); the session cookie is HttpOnly.
// The catalog (leave types, reasons) is Admin-managed; form actions re-run this load, so edits show at once.
export const load: LayoutServerLoad = async (event) => {
	const { locals } = event;
	let catalog = EMPTY;
	if (locals.user?.active) {
		const res = await api<Catalog>(event, 'GET', '/api/v1/catalog');
		if (res.ok) catalog = res.data;
	}
	return {
		user: locals.user,
		isReviewer: locals.isReviewer,
		unread: locals.unread,
		csrfToken: locals.csrfToken,
		lang: locals.lang,
		catalog
	};
};
