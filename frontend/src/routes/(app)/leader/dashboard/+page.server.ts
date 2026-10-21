import { load as apiLoad } from '#lib/server/api.ts';
import type { ListItem } from '#lib/leave.ts';
import type { PageServerLoad } from './$types';

interface ReviewSummary {
	counts: Record<string, number>;
	overdue: number;
	pending: ListItem[] | null;
	needs_changes: ListItem[] | null;
}

export const load: PageServerLoad = async (event) => ({
	summary: await apiLoad<ReviewSummary>(event, '/api/v1/reviews/summary')
});
