import { load as apiLoad } from '#lib/server/api.ts';
import type { Paged } from '#lib/admin-types.ts';
import { filterQuery, type ListItem } from '#lib/leave.ts';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => ({
	list: await apiLoad<Paged<ListItem>>(event, '/api/v1/leave-requests?' + filterQuery(event.url))
});
