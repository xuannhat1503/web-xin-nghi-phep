import { load as apiLoad } from '#lib/server/api.ts';
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

export const load: PageServerLoad = async (event) => ({
	status: await apiLoad<SystemStatus>(event, '/api/v1/admin/system')
});
