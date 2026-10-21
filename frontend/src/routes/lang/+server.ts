import { redirect } from '@sveltejs/kit';
import { isLang, LANG_COOKIE } from '#lib/i18n/index.ts';
import type { RequestHandler } from './$types';

// Language preference only (not security-sensitive). A POST form works without JavaScript.
export const POST: RequestHandler = async ({ request, cookies, url }) => {
	const form = await request.formData();
	const lang = form.get('lang');
	if (isLang(lang)) {
		cookies.set(LANG_COOKIE, lang, {
			path: '/',
			maxAge: 60 * 60 * 24 * 365,
			sameSite: 'lax',
			httpOnly: false,
			secure: url.protocol === 'https:'
		});
	}
	const back = String(form.get('redirect_to') ?? '/');
	// Same-site paths only (no open redirect).
	const safe =
		back.startsWith('/') && !back.startsWith('//') && !back.startsWith('/\\') ? back : '/';
	redirect(303, safe);
};
