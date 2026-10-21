import { load as apiLoad } from '#lib/server/api.ts';
import type { Paged } from '#lib/admin-types.ts';
import type { PageServerLoad } from './$types';

export interface NotificationItem {
	id: number;
	type: string;
	params: string;
	link: string;
	read_at: string | null;
	created_at: string;
}

// Opening the page marks everything as read (done from the browser after render, so hover-preloading
// the page never clears the badge by accident).
export const load: PageServerLoad = async (event) => {
	const q = new URLSearchParams();
	const page = Number(event.url.searchParams.get('page'));
	if (Number.isInteger(page) && page > 1) q.set('page', String(page));
	return { list: await apiLoad<Paged<NotificationItem>>(event, '/api/v1/notifications?' + q) };
};
