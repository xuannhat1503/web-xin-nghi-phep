import { load as apiLoad } from '#lib/server/api.ts';
import type { Paged } from '#lib/admin-types.ts';
import type { PageServerLoad } from './$types';

interface EmailSummary {
	id: number;
	email_type: string;
	to: string;
	subject: string;
	created_at: string;
}

export const load: PageServerLoad = async (event) => {
	const page = Math.max(1, Number(event.url.searchParams.get('page')) || 1);
	return {
		list: await apiLoad<Paged<EmailSummary>>(event, `/api/v1/admin/mock-emails?page=${page}`)
	};
};
