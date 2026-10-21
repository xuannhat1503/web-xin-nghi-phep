import { error, redirect } from '@sveltejs/kit';
import { actionResult, api, int, load as apiLoad } from '#lib/server/api.ts';
import { fieldsFromForm } from '#lib/server/leave-form.ts';
import type { ConfirmedDay, Evaluation, LeaveContext, LeaveRequest } from '#lib/leave.ts';
import type { Actions, PageServerLoad } from './$types';

function currentYear(): number {
	return Number(
		new Intl.DateTimeFormat('en', { timeZone: 'Asia/Ho_Chi_Minh', year: 'numeric' }).format(
			new Date()
		)
	);
}

export const load: PageServerLoad = async (event) => {
	const { id } = event.params;
	if (!/^\d+$/.test(id)) error(404, 'Leave request not found');
	const [request, context] = await Promise.all([
		apiLoad<LeaveRequest>(event, `/api/v1/leave-requests/${id}`),
		apiLoad<LeaveContext>(event, '/api/v1/leave-context')
	]);
	if (!request.editable) redirect(303, `/leave/${id}`);

	// CONFIRMED days for the date picker (this year, next year and the draft's own years).
	const y = currentYear();
	const years = new Set([y, y + 1]);
	for (const d of [request.start_date, request.end_date]) if (d) years.add(Number(d.slice(0, 4)));
	const calendars = await Promise.all(
		[...years].map((yr) => apiLoad<{ days: ConfirmedDay[] }>(event, `/api/v1/calendar?year=${yr}`))
	);

	// Initial estimate so the page is useful without JavaScript.
	const preview = await api<Evaluation>(event, 'POST', '/api/v1/leave-requests/preview', {
		request_id: request.id,
		leave_type: request.leave_type,
		start_date: request.start_date,
		end_date: request.end_date,
		start_time: request.start_time,
		end_time: request.end_time,
		reason_code: request.reason_code,
		reason_text: request.reason_text
	});
	return {
		request,
		context,
		days: calendars.flatMap((c) => c.days),
		preview: preview.ok ? preview.data : null
	};
};

export const actions = {
	// Fallback save for browsers without JavaScript (the page auto-saves through the API otherwise).
	save: async (event) => {
		const f = await event.request.formData();
		const ctx = await apiLoad<LeaveContext>(event, '/api/v1/leave-context');
		const res = await api<LeaveRequest>(
			event,
			'PATCH',
			`/api/v1/leave-requests/${event.params.id}`,
			{
				version: int(f, 'version'),
				...fieldsFromForm(f, ctx.policy)
			}
		);
		if (res.ok && f.get('next') === 'review') redirect(303, `/leave/${event.params.id}/review`);
		return actionResult(res, 'leave.save.saved');
	}
} satisfies Actions;
