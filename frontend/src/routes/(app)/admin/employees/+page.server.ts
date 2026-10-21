import { load as apiLoad } from '#lib/server/api.ts';
import type { EmployeeList } from '#lib/admin-types.ts';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	const sp = event.url.searchParams;
	const q = (sp.get('q') ?? '').slice(0, 100);
	const status = ['active', 'inactive'].includes(sp.get('status') ?? '')
		? sp.get('status')!
		: 'all';
	const noManager = sp.get('no_manager') === '1';
	const page = Math.max(1, Number(sp.get('page')) || 1);

	const params = new URLSearchParams({ q, status, page: String(page) });
	if (noManager) params.set('no_manager', '1');
	const list = await apiLoad<EmployeeList>(event, '/api/v1/admin/employees?' + params);
	return { list, filters: { q, status, noManager } };
};
