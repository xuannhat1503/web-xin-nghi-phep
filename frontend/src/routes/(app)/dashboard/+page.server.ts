import { load as apiLoad } from '#lib/server/api.ts';
import type { ListItem } from '#lib/leave.ts';
import type { PageServerLoad } from './$types';

interface PublicCalendar {
	year: number;
	year_confirmed: boolean;
	days: {
		date: string;
		name: string;
		kind: string;
		start_time: string | null;
		end_time: string | null;
	}[];
}

export const load: PageServerLoad = async (event) => {
	const today = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Ho_Chi_Minh' }).format(
		new Date()
	);
	const year = Number(today.slice(0, 4));
	// Own requests only + the company calendar (CONFIRMED days only). Nothing about other employees.
	const [calendar, summary] = await Promise.all([
		apiLoad<PublicCalendar>(event, `/api/v1/calendar?year=${year}`),
		apiLoad<{ counts: Record<string, number>; recent: ListItem[] | null }>(
			event,
			'/api/v1/leave-summary'
		)
	]);
	return {
		calendar,
		summary,
		upcoming: calendar.days
			.filter((d) => d.date >= today && d.kind !== 'MAKEUP_WORKDAY')
			.slice(0, 8)
	};
};
