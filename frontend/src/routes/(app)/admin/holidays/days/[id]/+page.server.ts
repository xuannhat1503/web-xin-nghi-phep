import { error } from '@sveltejs/kit';
import { load as apiLoad } from '#lib/server/api.ts';
import type { CalendarYearView } from '#lib/admin-types.ts';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	const year = Number(event.url.searchParams.get('year'));
	if (!/^\d+$/.test(event.params.id) || !Number.isInteger(year) || year < 2000 || year > 2200) {
		error(404, 'Calendar day not found');
	}
	const view = await apiLoad<CalendarYearView>(event, `/api/v1/admin/calendar/years/${year}`);
	const day = view.days.find((d) => d.id === Number(event.params.id));
	if (!day) error(404, 'Calendar day not found');
	return { day, year };
};
