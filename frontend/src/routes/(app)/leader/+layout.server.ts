import { error } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';

// UI convenience only: the reviewer APIs scope every query to requests assigned to the caller.
export const load: LayoutServerLoad = ({ locals }) => {
	if (!locals.isReviewer) error(403, 'You do not have access to this page.');
};
