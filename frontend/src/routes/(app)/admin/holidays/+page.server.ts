import { error, fail, redirect } from '@sveltejs/kit';
import { actionResult, api, int, load as apiLoad, str } from '#lib/server/api.ts';
import type { CalendarYearView } from '#lib/admin-types.ts';
import type { Msg } from '#lib/i18n/index.ts';
import type { Actions, PageServerLoad } from './$types';

function currentYear(): number {
	return Number(
		new Intl.DateTimeFormat('en', { timeZone: 'Asia/Ho_Chi_Minh', year: 'numeric' }).format(
			new Date()
		)
	);
}

function yearParam(url: URL): number {
	const y = Number(url.searchParams.get('year'));
	return Number.isInteger(y) && y >= 2000 && y <= 2200 ? y : currentYear();
}

export const load: PageServerLoad = async (event) => {
	const year = yearParam(event.url);
	const view = await apiLoad<CalendarYearView>(event, `/api/v1/admin/calendar/years/${year}`);
	const sp = event.url.searchParams;
	return {
		view,
		mode: sp.get('view') === 'calendar' ? 'calendar' : 'list',
		kind: sp.get('kind') ?? '',
		status: sp.get('status') ?? '',
		thisYear: currentYear()
	};
};

function dayBody(form: FormData) {
	const start = str(form, 'start_time');
	const end = str(form, 'end_time');
	return {
		date: str(form, 'date'),
		name: str(form, 'name'),
		kind: str(form, 'kind'),
		start_time: start || null,
		end_time: end || null,
		source: str(form, 'source'),
		note: str(form, 'note')
	};
}

/** Extra notes after a calendar change: reopened year and flagged requests. */
function changeNotes(data: unknown): Msg[] {
	const d = data as { year_reopened?: boolean; affected_requests?: Record<string, number> } | null;
	const notes: Msg[] = [];
	if (d?.year_reopened) notes.push('toast.yearReopened');
	const n = Object.values(d?.affected_requests ?? {}).reduce((a, b) => a + b, 0);
	if (n > 0) notes.push({ key: 'toast.affected', params: { n } });
	return notes;
}

export const actions = {
	seed: async (event) => {
		const year = yearParam(event.url);
		const res = await api<{ inserted: number; year_confirmed: boolean }>(
			event,
			'POST',
			`/api/v1/admin/calendar/years/${year}/seed`
		);
		if (!res.ok) return actionResult(res, []);
		if (res.data.year_confirmed)
			return { success: { key: 'toast.alreadyConfirmed', params: { year } } as Msg };
		return { success: { key: 'toast.seeded', params: { n: res.data.inserted } } as Msg };
	},

	confirmYear: async (event) => {
		const year = yearParam(event.url);
		const form = await event.request.formData();
		const res = await api(event, 'POST', `/api/v1/admin/calendar/years/${year}/confirm`, {
			version: int(form, 'version')
		});
		return actionResult(res, { key: 'toast.yearConfirmed', params: { year } });
	},

	create: async (event) => {
		const form = await event.request.formData();
		const res = await api(event, 'POST', '/api/v1/admin/calendar/days', dayBody(form));
		if (!res.ok) return actionResult(res, [], { values: dayBody(form) });
		return { success: ['toast.dayAdded', ...changeNotes(res.data)] as Msg };
	},

	update: async (event) => {
		const form = await event.request.formData();
		const id = int(form, 'day_id');
		const res = await api(event, 'PUT', `/api/v1/admin/calendar/days/${id}`, {
			version: int(form, 'version'),
			...dayBody(form)
		});
		if (!res.ok) return actionResult(res, []);
		const year = str(form, 'date').slice(0, 4);
		redirect(303, `/admin/holidays?year=${year}&updated=1`);
	},

	setStatus: async (event) => {
		const form = await event.request.formData();
		const id = int(form, 'day_id');
		const op = str(form, 'op');
		if (op !== 'confirm' && op !== 'unconfirm')
			return fail(400, { errorKey: 'toast.unknownAction' });
		const res = await api(event, 'POST', `/api/v1/admin/calendar/days/${id}/${op}`, {
			version: int(form, 'version')
		});
		const msg = op === 'confirm' ? 'toast.dayConfirmed' : 'toast.dayReverted';
		if (!res.ok) return actionResult(res, msg);
		return { success: [msg, ...changeNotes(res.data)] as Msg };
	},

	delete: async (event) => {
		const form = await event.request.formData();
		const id = int(form, 'day_id');
		const version = int(form, 'version');
		if (!Number.isInteger(id) || !Number.isInteger(version)) error(400, 'Invalid request');
		const res = await api(event, 'DELETE', `/api/v1/admin/calendar/days/${id}?version=${version}`);
		return actionResult(res, 'toast.dayDeleted');
	}
} satisfies Actions;
