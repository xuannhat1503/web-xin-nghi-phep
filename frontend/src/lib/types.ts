export type Role = 'ADMIN' | 'EMPLOYEE';

export interface CurrentEmployee {
	id: number;
	full_name: string;
	email: string;
	role: Role;
	active: boolean;
}

/** Response of GET /api/v1/me. */
export interface MeResponse {
	employee: CurrentEmployee;
	is_reviewer: boolean;
	unread_notifications: number;
	csrf_token: string;
	session: { idle_expires_at: string; absolute_expires_at: string };
}
