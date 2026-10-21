import { error } from '@sveltejs/kit';
import { load as apiLoad } from '#lib/server/api.ts';
import type { EmailSummary } from '../+page.server.ts';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	if (!/^\d+$/.test(event.params.id)) error(404, 'Not found');
	return {
		mail: await apiLoad<{
			email: EmailSummary;
			cc: string[];
			body: string;
			delivered_to: string;
			error: string | null;
		}>(event, `/api/v1/admin/email-log/${event.params.id}`)
	};
};
