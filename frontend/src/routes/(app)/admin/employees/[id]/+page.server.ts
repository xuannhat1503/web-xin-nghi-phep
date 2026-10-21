import { error, fail } from '@sveltejs/kit';
import { api, actionResult, int, load as apiLoad, str } from '#lib/server/api.ts';
import type { Candidate, EmployeeDetail } from '#lib/admin-types.ts';
import type { MessageKey } from '#lib/i18n/index.ts';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	const id = event.params.id;
	if (!/^\d+$/.test(id)) error(404, 'Employee not found');
	const mq = (event.url.searchParams.get('mq') ?? '').slice(0, 100);
	const [employee, candidates] = await Promise.all([
		apiLoad<EmployeeDetail>(event, `/api/v1/admin/employees/${id}`),
		apiLoad<{ items: Candidate[] }>(
			event,
			`/api/v1/admin/employees/${id}/manager-candidates?` + new URLSearchParams({ q: mq })
		)
	]);
	return { employee, candidates: candidates.items, mq, me: event.locals.user!.id };
};

const ACCESS: Record<string, { body: Record<string, unknown>; msg: MessageKey }> = {
	activate: { body: { active: true }, msg: 'toast.activated' },
	deactivate: { body: { active: false }, msg: 'toast.deactivated' },
	make_admin: { body: { role: 'ADMIN' }, msg: 'toast.madeAdmin' },
	make_employee: { body: { role: 'EMPLOYEE' }, msg: 'toast.madeEmployee' }
};

export const actions = {
	access: async (event) => {
		const form = await event.request.formData();
		const op = ACCESS[str(form, 'op')];
		if (!op) return fail(400, { errorKey: 'toast.unknownAction' });
		const res = await api(event, 'PATCH', `/api/v1/admin/employees/${event.params.id}`, {
			version: int(form, 'version'),
			...op.body
		});
		return actionResult(res, op.msg);
	},

	manager: async (event) => {
		const form = await event.request.formData();
		const raw = str(form, 'manager_id');
		const managerId = raw === '' || raw === 'none' ? null : Number(raw);
		if (managerId !== null && !Number.isInteger(managerId)) {
			return fail(400, { errorKey: 'toast.chooseManager' });
		}
		const res = await api(event, 'PUT', `/api/v1/admin/employees/${event.params.id}/manager`, {
			version: int(form, 'version'),
			manager_id: managerId,
			reason: str(form, 'reason')
		});
		return actionResult(res, managerId === null ? 'toast.managerRemoved' : 'toast.managerAssigned');
	}
} satisfies Actions;
