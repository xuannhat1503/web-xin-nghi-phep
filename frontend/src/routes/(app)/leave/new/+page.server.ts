import { redirect } from '@sveltejs/kit';
import { actionResult, api, int, load as apiLoad } from '#lib/server/api.ts';
import type { LeaveRequest } from '#lib/leave.ts';
import type { Actions, PageServerLoad } from './$types';

interface DraftSummary {
	id: number;
	leave_type: string | null;
	start_date: string | null;
	end_date: string | null;
	total_minutes: number;
	version: number;
	updated_at: string;
}

export const load: PageServerLoad = async (event) => ({
	drafts: (await apiLoad<{ items: DraftSummary[] }>(event, '/api/v1/leave-requests/drafts')).items
});

export const actions = {
	// POST, never GET: creating a draft is a write.
	create: async (event) => {
		const res = await api<LeaveRequest>(event, 'POST', '/api/v1/leave-requests');
		if (!res.ok) return actionResult(res, []);
		redirect(303, `/leave/${res.data.id}/edit`);
	},
	delete: async (event) => {
		const f = await event.request.formData();
		const res = await api(
			event,
			'DELETE',
			`/api/v1/leave-requests/${int(f, 'draft_id')}?version=${int(f, 'version')}`
		);
		return actionResult(res, 'toast.draftDeleted');
	}
} satisfies Actions;
