import type { SubmitFunction } from '$app/forms';
import type { I18n, Msg } from '#lib/i18n/index.ts';
import { toast } from '#lib/toast.svelte.ts';

/**
 * use:enhance callback: shows the action's success message (a translation key) or the API error
 * (translated) as a toast, then applies the result (re-running loads for fresh data and versions).
 */
export function withToast(i18n: I18n): SubmitFunction {
	return () => {
		return async ({ result, update }) => {
			if (result.type === 'success' && result.data?.success) {
				toast(i18n.msg(result.data.success as Msg), 'success');
			} else if (result.type === 'failure' && typeof result.data?.error === 'string') {
				toast(i18n.server(result.data.error), 'error', 6000);
			} else if (result.type === 'failure' && result.data?.errorKey) {
				toast(i18n.msg(result.data.errorKey as Msg), 'error', 6000);
			} else if (result.type === 'error') {
				toast(i18n.t('common.genericError'), 'error', 6000);
			}
			await update({ reset: result.type === 'success' });
		};
	};
}

/** Loose shape of what our form actions return. success/errorKey are translation keys; error is API text. */
export type ActionData =
	| {
			success?: Msg;
			error?: string;
			errorKey?: Msg;
			code?: string;
			fields?: Record<string, string>;
			values?: unknown;
			reasonForm?: string;
			typeForm?: string;
	  }
	| null
	| undefined;
