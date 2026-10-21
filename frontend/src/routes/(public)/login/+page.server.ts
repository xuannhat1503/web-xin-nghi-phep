import { loginErrorKey, loginHref } from '#lib/login.ts';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = ({ url }) => ({
	loginHref: loginHref(url.searchParams.get('return_to')),
	errorKey: loginErrorKey(url.searchParams.get('error')),
	signedOut: url.searchParams.has('signed_out')
});
