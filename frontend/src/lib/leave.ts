// Shapes returned by the Go leave API and helpers shared by the leave pages.

export interface Issue {
	code: string;
	field?: string;
	message: string;
	params?: Record<string, unknown>;
}

export interface CalcDay {
	date: string;
	minutes: number;
	note?: string;
	name?: string;
}

export interface CalcExcluded {
	date: string;
	reason: 'rest_day' | 'holiday';
	kind?: string;
	name?: string;
}

export interface Calculation {
	total_minutes: number;
	days: CalcDay[];
	excluded: CalcExcluded[];
}

export interface Evaluation {
	calculation: Calculation | null;
	minutes_per_day: number;
	issues: Issue[];
	can_submit: boolean;
}

export interface LeaveRequest {
	id: number;
	request_code: string | null;
	status: 'DRAFT' | 'PENDING' | 'NEEDS_CHANGES' | 'APPROVED' | 'REJECTED' | 'CANCELLED';
	/** Name written on this request and its emails (null = the account name). */
	employee_name: string | null;
	leave_type: string | null;
	start_date: string | null;
	end_date: string | null;
	start_time: string | null;
	end_time: string | null;
	total_minutes: number;
	reason_code: string | null;
	reason_label: string | null;
	reason_text: string;
	reviewer: { full_name: string; email: string } | null;
	rejection_reason: string | null;
	change_request_reason: string | null;
	calculation: (Calculation & { minutes_per_day: number; policy_version_no: number }) | null;
	calendar_changed: boolean;
	submit_count: number;
	version: number;
	editable: boolean;
	created_at: string;
	first_submitted_at: string | null;
	submitted_at: string | null;
	reviewed_at: string | null;
	updated_at: string;
}

export interface LeavePolicy {
	work_start: string;
	work_end: string;
	lunch_start: string;
	lunch_end: string;
	workdays: number[];
	minutes_per_day: number;
	min_leave_minutes: number;
	leave_unit_minutes: number;
	leave_types: Record<
		string,
		{
			advance_notice_days: number;
			allow_past_dates: boolean;
			max_past_days: number;
			reason_required: boolean;
		}
	>;
	timezone: string;
}

export interface LeaveContext {
	employee: { id: number; full_name: string; email: string };
	manager: { id: number; full_name: string; email: string; active: boolean } | null;
	policy: LeavePolicy;
	reasons: { id: number; code: string; label: string; requires_text: boolean }[];
}

export interface ConfirmedDay {
	date: string;
	name: string;
	kind: string;
	start_time: string | null;
	end_time: string | null;
}

export interface ReviewData {
	request: LeaveRequest;
	evaluation: Evaluation;
	manager: { full_name: string; email: string } | null;
	email_preview: { to: string; cc: string[]; subject: string; body: string } | null;
}

export type Mode = 'FULL' | 'MORNING' | 'AFTERNOON' | 'CUSTOM';
export const MODES: Mode[] = ['FULL', 'MORNING', 'AFTERNOON', 'CUSTOM'];

/** Times implied by a quick-pick mode (null for custom hours). */
export function modeTimes(mode: Mode, p: LeavePolicy): { start: string; end: string } | null {
	switch (mode) {
		case 'FULL':
			return { start: p.work_start, end: p.work_end };
		case 'MORNING':
			return { start: p.work_start, end: p.lunch_start };
		case 'AFTERNOON':
			return { start: p.lunch_end, end: p.work_end };
		default:
			return null;
	}
}

/** Recognises the quick-pick mode of stored times so a resumed draft shows the right choice. */
export function detectMode(
	start: string | null,
	end: string | null,
	sameDay: boolean,
	p: LeavePolicy
): Mode {
	if (!start || !end) return 'FULL';
	if (start === p.work_start && end === p.work_end) return 'FULL';
	if (sameDay && start === p.work_start && end === p.lunch_start) return 'MORNING';
	if (sameDay && start === p.lunch_end && end === p.work_end) return 'AFTERNOON';
	return 'CUSTOM';
}

export interface ListItem {
	id: number;
	request_code: string | null;
	status: LeaveRequest['status'];
	leave_type: string | null;
	start_date: string | null;
	end_date: string | null;
	start_time: string | null;
	end_time: string | null;
	total_minutes: number;
	employee_name: string;
	reviewer_name: string | null;
	created_at: string;
	submitted_at: string | null;
	reviewed_at: string | null;
}

export interface TimelineEvent {
	action: string;
	actor: string;
	from_status: string | null;
	to_status: string | null;
	submit_no: number | null;
	comment: string | null;
	created_at: string;
}

export interface VersionSnapshot {
	submit_no: number;
	leave_type: string;
	start_date: string;
	end_date: string;
	start_time: string;
	end_time: string;
	total_minutes: number;
	reason_code: string | null;
	reason_text: string;
	reviewer: string;
	created_at: string;
}

export const STATUS_TONES = {
	DRAFT: 'slate',
	PENDING: 'amber',
	NEEDS_CHANGES: 'violet',
	APPROVED: 'green',
	REJECTED: 'red',
	CANCELLED: 'slate'
} as const;

/** Query params understood by the list APIs (kept in the page URL). */
export const FILTER_KEYS = [
	'year',
	'month',
	'status',
	'leave_type',
	'code',
	'q',
	'from',
	'to',
	'submitted_from',
	'submitted_to',
	'reviewed_from',
	'reviewed_to',
	'sort',
	'page'
] as const;

export function filterQuery(url: URL): string {
	const out = new URLSearchParams();
	for (const k of FILTER_KEYS) {
		const v = url.searchParams.get(k);
		if (v) out.set(k, v.slice(0, 100));
	}
	return out.toString();
}
