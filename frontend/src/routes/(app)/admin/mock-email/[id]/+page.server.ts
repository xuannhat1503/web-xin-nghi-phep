import { error } from '@sveltejs/kit';
import { load as apiLoad } from '#lib/server/api.ts';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	if (!/^\d+$/.test(event.params.id)) error(404, 'Not found');
	return {
		mail: await apiLoad<{
			email: { id: number; email_type: string; to: string; subject: string; created_at: string };
			cc: string[];
			body: string;
			transport: string;
		}>(event, `/api/v1/admin/mock-emails/${event.params.id}`)
	};
};
