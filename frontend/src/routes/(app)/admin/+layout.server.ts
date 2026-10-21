import { error } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';

// Defence in depth for the UI; every admin API endpoint is also authorized by the Go backend.
export const load: LayoutServerLoad = ({ locals }) => {
	if (locals.user?.role !== 'ADMIN' || !locals.user.active) {
		error(403, 'You do not have access to this page.');
	}
};
