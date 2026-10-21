// See https://svelte.dev/docs/kit/types#app.d.ts
import type { CurrentEmployee } from '#lib/types.ts';
import type { Lang } from '#lib/i18n/index.ts';

declare global {
	namespace App {
		interface Error {
			message: string;
			code?: string;
		}
		interface Locals {
			user: CurrentEmployee | null;
			isReviewer: boolean;
			unread: number;
			csrfToken: string | null;
			backendUnavailable: boolean;
			lang: Lang;
		}
	}
}

export {};
