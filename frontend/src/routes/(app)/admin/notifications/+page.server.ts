import { actionResult, api, int, load as apiLoad } from '#lib/server/api.ts';
import type { Paged } from '#lib/admin-types.ts';
import type { Actions, PageServerLoad } from './$types';

interface OutboxItem {
	id: number;
	event_type: string;
	status: string;
	attempts: number;
	next_retry_at: string;
	last_error: string | null;
	request_code: string | null;
	created_at: string;
	processed_at: string | null;
}

export const load: PageServerLoad = async (event) => {
	const sp = event.url.searchParams;
	const q = new URLSearchParams();
	const status = sp.get('status') ?? '';
	if (['PENDING', 'PROCESSING', 'SENT', 'FAILED'].includes(status)) q.set('status', status);
	q.set('page', String(Math.max(1, Number(sp.get('page')) || 1)));
	const res = await apiLoad<{ page: Paged<OutboxItem>; counts: Record<string, number> }>(
		event,
		'/api/v1/admin/outbox?' + q
	);
	return { ...res, status };
};

export const actions = {
	retry: async (event) => {
		const f = await event.request.formData();
		const res = await api(event, 'POST', `/api/v1/admin/outbox/${int(f, 'outbox_id')}/retry`);
		return actionResult(res, 'toast.retried');
	}
} satisfies Actions;
