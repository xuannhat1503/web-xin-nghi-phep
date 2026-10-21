// Shapes returned by the Go admin API.

export interface Paged<T> {
	items: T[];
	page: number;
	page_size: number;
	total_items: number;
	total_pages: number;
}

export interface PersonRef {
	id: number;
	full_name: string;
	email?: string;
}

export interface EmployeeRow {
	id: number;
	full_name: string;
	email: string;
	role: 'ADMIN' | 'EMPLOYEE';
	active: boolean;
	manager: PersonRef | null;
	manager_active?: boolean;
	last_login_at: string | null;
	created_at: string;
	version: number;
}

export interface EmployeeList extends Paged<EmployeeRow> {
	summary: { inactive: number; active_without_manager: number; total: number };
}

export interface HistoryEntry {
	kind: 'account' | 'manager';
	action: string;
	actor: string;
	before?: Record<string, unknown> | null;
	after?: Record<string, unknown> | null;
	reason?: string;
	created_at: string;
}

export interface EmployeeDetail extends EmployeeRow {
	updated_at: string;
	active_reports: number;
	history: HistoryEntry[] | null;
}

export interface Candidate {
	id: number;
	full_name: string;
	email: string;
}

export type DayKind =
	'NATIONAL_HOLIDAY' | 'COMPANY_HOLIDAY' | 'COMPENSATORY_HOLIDAY' | 'MAKEUP_WORKDAY';

export interface CalendarDay {
	id: number;
	date: string;
	name: string;
	kind: DayKind;
	status: 'DRAFT' | 'CONFIRMED';
	start_time: string | null;
	end_time: string | null;
	source: string;
	note: string;
	seeded: boolean;
	admin_modified: boolean;
	version: number;
	confirmed_at: string | null;
	confirmed_by: string | null;
}

export interface CalendarYearView {
	year: number;
	exists: boolean;
	status: 'DRAFT' | 'CONFIRMED';
	version: number;
	confirmed_at: string | null;
	confirmed_by: string | null;
	days: CalendarDay[];
	warnings: { code: string; message: string; params?: Record<string, unknown> }[];
	workdays: number[];
	affected_requests: Record<string, number>;
	events: { day_id: number | null; action: string; actor: string; created_at: string }[];
	years:
		| {
				year: number;
				status: string;
				version: number;
				total_days: number;
				draft_days: number;
		  }[]
		| null;
}

export interface LeaveTypeRule {
	advance_notice_days: number;
	allow_past_dates: boolean;
	max_past_days: number;
	reason_required: boolean;
}

export interface Policy {
	version_id: number;
	version_no: number;
	timezone: string;
	work_start: string;
	work_end: string;
	lunch_start: string;
	lunch_end: string;
	workdays: number[];
	minutes_per_day: number;
	min_leave_minutes: number;
	leave_unit_minutes: number;
	reminder_basis: 'WORKING_HOURS' | 'CLOCK_HOURS';
	reminder_interval_minutes: number;
	reminder_max_count: number;
	leave_types: Record<string, LeaveTypeRule>;
	change_note: string;
	created_at: string;
}

export interface Reason {
	id: number;
	code: string;
	label: string;
	label_en: string;
	requires_text: boolean;
	sort_order: number;
	active: boolean;
}

export interface SettingsOverview {
	settings_version: number;
	policy: Policy;
	finance_cc_emails: string[];
	reasons: Reason[];
	leave_types: LeaveType[];
	history:
		{ version_no: number; change_note: string; created_at: string; created_by: string }[] | null;
	open_requests: Record<string, number>;
	email: {
		mode: 'MOCK' | 'TEST_EMAIL' | 'LIVE';
		test_to: string;
		test_cc: string[];
		smtp_configured: boolean;
		smtp_from: string;
		live_enabled: boolean;
		allowed_emails: string[];
		allowed_domain: string;
	};
}

export interface LeaveType {
	code: string;
	name_vi: string;
	name_en: string;
	sort_order: number;
	active: boolean;
}

/** Every leave type and reason (active or not), used to label requests in the current language. */
export interface Catalog {
	leave_types: LeaveType[];
	reasons: Reason[];
}
