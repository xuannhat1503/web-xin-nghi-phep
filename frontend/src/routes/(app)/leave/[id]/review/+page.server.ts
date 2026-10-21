import { error, fail, redirect } from '@sveltejs/kit';
import { api, int, str } from '#lib/server/api.ts';
import type { Issue, LeaveRequest, ReviewData } from '#lib/leave.ts';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	const { id } = event.params;
	if (!/^\d+$/.test(id)) error(404, 'Leave request not found');
	const res = await api<ReviewData>(event, 'GET', `/api/v1/leave-requests/${id}/review`);
	if (!res.ok) {
		if (res.error.code === 'not_editable') redirect(303, `/leave/${id}`);
		error(res.status, res.error.message);
	}
	// One key per rendered review: double clicks and retries cannot create a second submission.
	return { review: res.data, idempotencyKey: crypto.randomUUID() };
};

export const actions = {
	submit: async (event) => {
		const f = await event.request.formData();
		const key = str(f, 'idempotency_key');
		const res = await api<LeaveRequest>(
			event,
			'POST',
			`/api/v1/leave-requests/${event.params.id}/submit`,
			{ version: int(f, 'version') },
			{ 'idempotency-key': key }
		);
		if (res.ok) redirect(303, `/leave/${event.params.id}?submitted=1`);
		const issues = (res.error as unknown as { issues?: Issue[] }).issues;
		return fail(res.status, {
			error: res.error.message,
			code: res.error.code,
			issues: issues ?? []
		});
	}
} satisfies Actions;
