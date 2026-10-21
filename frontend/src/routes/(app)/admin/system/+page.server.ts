import { load as apiLoad } from '#lib/server/api.ts';
import type { SettingsOverview } from '#lib/admin-types.ts';
import type { PageServerLoad } from './$types';

export interface SystemStatus {
	server_time: string;
	database: 'ok' | 'fail';
	email_mode?: string;
	migration?: { version: number; latest: number; dirty: boolean; up_to_date: boolean };
	workers?: {
		name: string;
		last_run_at: string | null;
		last_success_at: string | null;
		last_error: string | null;
	}[];
	outbox?: Record<string, number>;
	recent_errors?: { component: string; code: string; message: string; created_at: string }[];
}

export const load: PageServerLoad = async (event) => {
	const status = await apiLoad<SystemStatus>(event, '/api/v1/admin/system');
	// Email delivery details (SMTP / EMAIL_LIVE_ENABLED) come from the settings API; skipped when the DB is down.
	let email: SettingsOverview['email'] | null = null;
	if (status.database === 'ok') {
		email = (await apiLoad<SettingsOverview>(event, '/api/v1/admin/settings')).email;
	}
	return { status, email };
};
