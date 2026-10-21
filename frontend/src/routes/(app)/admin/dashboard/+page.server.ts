import { load as apiLoad } from '#lib/server/api.ts';
import type { PageServerLoad } from './$types';

export interface Analytics {
	year: number;
	month: number;
	minutes_per_day: number;
	kpis: {
		total: number;
		by_status: Record<string, number>;
		approved_minutes: number;
		avg_processing_hours: number | null;
		overdue_pending: number;
		inactive_employees: number;
		employees_without_manager: number;
		unconfirmed_years: number[];
	};
	monthly_counts: number[];
	approved_by_month_type: { month: number; leave_type: string; minutes: number }[];
}

export const load: PageServerLoad = async (event) => {
	const sp = event.url.searchParams;
	const q = new URLSearchParams();
	const y = Number(sp.get('year'));
	const m = Number(sp.get('month'));
	if (Number.isInteger(y) && y >= 2000 && y <= 2200) q.set('year', String(y));
	if (Number.isInteger(m) && m >= 1 && m <= 12) q.set('month', String(m));
	// Aggregates only (the API never returns names, emails or reasons).
	return { analytics: await apiLoad<Analytics>(event, '/api/v1/admin/analytics?' + q) };
};
