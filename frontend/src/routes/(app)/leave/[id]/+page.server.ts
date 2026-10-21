import { error } from '@sveltejs/kit';
import { actionResult, api, int, load as apiLoad } from '#lib/server/api.ts';
import type { LeaveRequest, TimelineEvent, VersionSnapshot } from '#lib/leave.ts';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	if (!/^\d+$/.test(event.params.id)) error(404, 'Leave request not found');
	// The API only returns the caller's own request; any other id is a 404.
	const [request, history] = await Promise.all([
		apiLoad<LeaveRequest>(event, `/api/v1/leave-requests/${event.params.id}`),
		apiLoad<{ events: TimelineEvent[]; versions: VersionSnapshot[] }>(
			event,
			`/api/v1/leave-requests/${event.params.id}/history`
		)
	]);
	return { request, history, submitted: event.url.searchParams.has('submitted') };
};

export const actions = {
	cancel: async (event) => {
		const f = await event.request.formData();
		const res = await api(event, 'POST', `/api/v1/leave-requests/${event.params.id}/cancel`, {
			version: int(f, 'version')
		});
		return actionResult(res, 'toast.cancelled');
	}
} satisfies Actions;
