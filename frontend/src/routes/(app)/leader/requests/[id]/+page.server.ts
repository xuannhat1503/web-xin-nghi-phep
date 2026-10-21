import { error, fail } from '@sveltejs/kit';
import { actionResult, api, int, load as apiLoad, str } from '#lib/server/api.ts';
import type { Calculation, TimelineEvent, VersionSnapshot } from '#lib/leave.ts';
import type { MessageKey } from '#lib/i18n/index.ts';
import type { Actions, PageServerLoad } from './$types';

export interface ReviewDetail {
	id: number;
	request_code: string | null;
	status: 'PENDING' | 'NEEDS_CHANGES' | 'APPROVED' | 'REJECTED' | 'CANCELLED';
	employee: { id: number; full_name: string; email: string };
	leave_type: string | null;
	start_date: string | null;
	end_date: string | null;
	start_time: string | null;
	end_time: string | null;
	total_minutes: number;
	reason_code: string | null;
	reason_label: string | null;
	reason_text: string;
	rejection_reason: string | null;
	change_request_reason: string | null;
	calculation: (Calculation & { minutes_per_day: number; policy_version_no: number }) | null;
	calendar_changed: boolean;
	version: number;
	submitted_at: string | null;
	reviewed_at: string | null;
	events: TimelineEvent[];
	versions: VersionSnapshot[];
}

export const load: PageServerLoad = async (event) => {
	if (!/^\d+$/.test(event.params.id)) error(404, 'Leave request not found');
	// 404 unless the request is currently assigned to the viewer.
	return { request: await apiLoad<ReviewDetail>(event, `/api/v1/reviews/${event.params.id}`) };
};

const DECISIONS: Record<string, { path: string; msg: MessageKey }> = {
	approve: { path: 'approve', msg: 'toast.approved' },
	reject: { path: 'reject', msg: 'toast.rejected' },
	request_changes: { path: 'request-changes', msg: 'toast.changesRequested' }
};

// Decisions are POST form actions (never GET, never from an email link); the API enforces CSRF,
// reviewer assignment, status and version.
export const actions = {
	decide: async (event) => {
		const f = await event.request.formData();
		const d = DECISIONS[str(f, 'decision')];
		if (!d) return fail(400, { errorKey: 'toast.unknownAction' });
		const comment = str(f, 'comment');
		if (d.path !== 'approve' && !comment)
			return fail(422, { errorKey: 'lrd.commentRequired', comment });
		const res = await api(event, 'POST', `/api/v1/reviews/${event.params.id}/${d.path}`, {
			version: int(f, 'version'),
			comment
		});
		return actionResult(res, d.msg);
	}
} satisfies Actions;
