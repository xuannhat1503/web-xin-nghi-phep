import { fail } from '@sveltejs/kit';
import { actionResult, api, int, load as apiLoad, str } from '#lib/server/api.ts';
import type { SettingsOverview } from '#lib/admin-types.ts';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => ({
	settings: await apiLoad<SettingsOverview>(event, '/api/v1/admin/settings')
});

const list = (s: string) => s.split(/[\s,;]+/).filter(Boolean);
const CODE = /^[A-Z][A-Z0-9_]{1,39}$/;

export const actions = {
	policy: async (event) => {
		const f = await event.request.formData();
		// The rules table posts one hidden "rule_types" per active leave type.
		const leaveTypes: Record<string, unknown> = {};
		for (const t of f.getAll('rule_types').map(String)) {
			const allowPast = f.get(`${t}.allow_past_dates`) === 'on';
			leaveTypes[t] = {
				advance_notice_days: int(f, `${t}.advance_notice_days`),
				allow_past_dates: allowPast,
				max_past_days: allowPast ? int(f, `${t}.max_past_days`) : 0,
				reason_required: f.get(`${t}.reason_required`) === 'on'
			};
		}
		const hours = int(f, 'reminder_interval_hours');
		const res = await api(event, 'PUT', '/api/v1/admin/settings/policy', {
			version: int(f, 'version'),
			timezone: str(f, 'timezone'),
			work_start: str(f, 'work_start'),
			work_end: str(f, 'work_end'),
			lunch_start: str(f, 'lunch_start'),
			lunch_end: str(f, 'lunch_end'),
			workdays: f.getAll('workdays').map(Number),
			min_leave_minutes: int(f, 'min_leave_minutes'),
			leave_unit_minutes: int(f, 'leave_unit_minutes'),
			reminder_basis: str(f, 'reminder_basis'),
			reminder_interval_minutes: Number.isFinite(hours) ? hours * 60 : NaN,
			reminder_max_count: int(f, 'reminder_max_count'),
			leave_types: leaveTypes,
			change_note: str(f, 'change_note')
		});
		return actionResult(res, 'toast.settingsSaved');
	},

	email: async (event) => {
		const f = await event.request.formData();
		const res = await api(event, 'PUT', '/api/v1/admin/settings/email', {
			version: int(f, 'version'),
			mode: str(f, 'mode'),
			test_to: str(f, 'test_to'),
			test_cc: list(str(f, 'test_cc'))
		});
		return actionResult(res, 'toast.emailSaved');
	},

	finance: async (event) => {
		const f = await event.request.formData();
		const res = await api(event, 'PUT', '/api/v1/admin/settings/finance-cc', {
			version: int(f, 'version'),
			finance_cc_emails: list(str(f, 'finance_cc_emails'))
		});
		return actionResult(res, 'toast.ccSaved');
	},

	createType: async (event) => {
		const f = await event.request.formData();
		const res = await api(event, 'POST', '/api/v1/admin/settings/leave-types', {
			code: str(f, 'code'),
			name_vi: str(f, 'name_vi'),
			name_en: str(f, 'name_en'),
			sort_order: int(f, 'sort_order') || 0
		});
		return actionResult(res, 'toast.typeAdded', { typeForm: 'new' });
	},

	updateType: async (event) => {
		const f = await event.request.formData();
		const code = str(f, 'type_code');
		if (!CODE.test(code)) return fail(400, { errorKey: 'toast.unknownAction' });
		const res = await api(event, 'PATCH', `/api/v1/admin/settings/leave-types/${code}`, {
			name_vi: str(f, 'name_vi'),
			name_en: str(f, 'name_en'),
			sort_order: int(f, 'sort_order') || 0,
			active: f.get('active') === 'on'
		});
		return actionResult(res, 'toast.typeUpdated', { typeForm: code });
	},

	deleteType: async (event) => {
		const f = await event.request.formData();
		const code = str(f, 'type_code');
		if (!CODE.test(code)) return fail(400, { errorKey: 'toast.unknownAction' });
		const res = await api<{ deleted: boolean }>(
			event,
			'DELETE',
			`/api/v1/admin/settings/leave-types/${code}`
		);
		const msg = res.ok && !res.data.deleted ? 'toast.typeDeactivated' : 'toast.typeDeleted';
		return actionResult(res, msg, { typeForm: code });
	},

	createReason: async (event) => {
		const f = await event.request.formData();
		const res = await api(event, 'POST', '/api/v1/admin/settings/reasons', {
			code: str(f, 'code'),
			label: str(f, 'label'),
			label_en: str(f, 'label_en'),
			requires_text: f.get('requires_text') === 'on',
			sort_order: int(f, 'sort_order') || 0
		});
		return actionResult(res, 'toast.reasonAdded', { reasonForm: 'new' });
	},

	updateReason: async (event) => {
		const f = await event.request.formData();
		const id = int(f, 'reason_id');
		if (!Number.isInteger(id)) return fail(400, { errorKey: 'toast.unknownAction' });
		const res = await api(event, 'PATCH', `/api/v1/admin/settings/reasons/${id}`, {
			label: str(f, 'label'),
			label_en: str(f, 'label_en'),
			requires_text: f.get('requires_text') === 'on',
			sort_order: int(f, 'sort_order') || 0,
			active: f.get('active') === 'on'
		});
		return actionResult(res, 'toast.reasonUpdated', { reasonForm: String(id) });
	},

	deleteReason: async (event) => {
		const f = await event.request.formData();
		const id = int(f, 'reason_id');
		if (!Number.isInteger(id)) return fail(400, { errorKey: 'toast.unknownAction' });
		const res = await api<{ deleted: boolean }>(
			event,
			'DELETE',
			`/api/v1/admin/settings/reasons/${id}`
		);
		const msg = res.ok && !res.data.deleted ? 'toast.reasonDeactivated' : 'toast.reasonDeleted';
		return actionResult(res, msg, { reasonForm: String(id) });
	}
} satisfies Actions;
