/** Display helpers. Dates are shown as dd/MM/yyyy and times in Asia/Ho_Chi_Minh (the company time zone). */
import type { I18n, MessageKey } from '#lib/i18n/index.ts';

const TZ = 'Asia/Ho_Chi_Minh';

/** "2026-04-30" -> "30/04/2026" (no time-zone conversion: it is a calendar date). */
export function formatDate(iso: string | null | undefined): string {
	if (!iso) return '';
	const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(iso);
	return m ? `${m[3]}/${m[2]}/${m[1]}` : iso;
}

/** "30/04/2026" -> "2026-04-30"; returns null when the text is not a real date. */
export function parseDisplayDate(text: string): string | null {
	const m = /^\s*(\d{1,2})\/(\d{1,2})\/(\d{4})\s*$/.exec(text);
	if (!m) return null;
	const [d, mo, y] = [Number(m[1]), Number(m[2]), Number(m[3])];
	const dt = new Date(Date.UTC(y, mo - 1, d));
	if (dt.getUTCFullYear() !== y || dt.getUTCMonth() !== mo - 1 || dt.getUTCDate() !== d)
		return null;
	return `${y}-${String(mo).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
}

const dtf = new Intl.DateTimeFormat('en-GB', {
	timeZone: TZ,
	day: '2-digit',
	month: '2-digit',
	year: 'numeric',
	hour: '2-digit',
	minute: '2-digit',
	hour12: false
});

/** UTC timestamp -> "30/04/2026 14:05" in company time. */
export function formatDateTime(ts: string | null | undefined): string {
	if (!ts) return '';
	const d = new Date(ts);
	if (Number.isNaN(d.getTime())) return '';
	return dtf.format(d).replace(',', '');
}

/** ISO weekday 1..7 (Mon..Sun) of a calendar date (computed in UTC so it never shifts). */
export function isoWeekday(iso: string): number {
	const [y, m, d] = iso.split('-').map(Number);
	return ((new Date(Date.UTC(y!, m! - 1, d!)).getUTCDay() + 6) % 7) + 1;
}

/** Localized short weekday of a calendar date (T2 / Mon …). */
export function weekday(i18n: I18n, iso: string): string {
	return i18n.t(`wd.${isoWeekday(iso)}` as MessageKey);
}

export const ISO_WEEKDAYS = [1, 2, 3, 4, 5, 6, 7] as const;

/** 90 -> "1 giờ 30 phút" / "1h 30m". */
export function formatMinutes(i18n: I18n, min: number): string {
	const h = Math.floor(min / 60);
	const m = min % 60;
	if (h && m) return i18n.t('common.hoursMinutes', { h, m });
	if (h) return i18n.t('common.hours', { h });
	return i18n.t('common.minutes', { m });
}

/** 960 min with an 8h day -> "16 giờ (2 ngày)". */
export function formatDuration(i18n: I18n, min: number, minutesPerDay: number): string {
	const hours = formatMinutes(i18n, min);
	if (!minutesPerDay) return hours;
	const days = Math.round((min / minutesPerDay) * 100) / 100;
	return i18n.t('leave.totalValue', { hours, days: i18n.t('common.days', { d: days }) });
}

export const KINDS = [
	'NATIONAL_HOLIDAY',
	'COMPANY_HOLIDAY',
	'COMPENSATORY_HOLIDAY',
	'MAKEUP_WORKDAY'
] as const;

export const kindLabel = (i18n: I18n, k: string) => i18n.t(`kind.${k}` as MessageKey);
/** Leave types are Admin-managed: names come from the catalog in the current language. */
export const leaveTypeLabel = (i18n: I18n, t: string | null | undefined) => i18n.leaveType(t);
export const statusLabel = (i18n: I18n, s: string) => i18n.t(`st.${s}` as MessageKey);

/** Reason label from the catalog in the current language; `stored` covers codes no longer listed. */
export function reasonLabel(
	i18n: I18n,
	code: string | null | undefined,
	stored: string | null | undefined
): string {
	return i18n.reason(code, stored);
}

/** Translates a coded API message (validation issue, calendar warning) with its params. */
export function codedText(
	i18n: I18n,
	prefix: 'issue' | 'warn',
	item: { code: string; message: string; params?: Record<string, unknown> | null }
): string {
	const key = `${prefix}.${item.code}`;
	const params = { ...(item.params ?? {}) } as Record<string, string | number>;
	if (typeof params.date === 'string') params.date = formatDate(params.date);
	if (typeof params.status === 'string') params.status = statusLabel(i18n, params.status);
	const text = i18n.t(key as MessageKey, params);
	return text === key ? i18n.server(item.message) : text;
}

/** Translates an API validation issue (leave form). */
export function issueText(
	i18n: I18n,
	issue: { code: string; message: string; params?: Record<string, unknown> }
): string {
	return codedText(i18n, 'issue', issue);
}

/** Current year in company time. */
export function companyYear(): number {
	return Number(
		new Intl.DateTimeFormat('en', { timeZone: 'Asia/Ho_Chi_Minh', year: 'numeric' }).format(
			new Date()
		)
	);
}

/**
 * Year choices for filters: next year down to `back` years ago, computed from today (so 2028 appears
 * automatically), plus the selected year if it falls outside that range. Newest first.
 */
export function yearOptions(selected?: number | null, back = 5): number[] {
	const now = companyYear();
	const years: number[] = [];
	for (let y = now + 1; y >= now - back; y--) years.push(y);
	if (selected && selected >= 2000 && selected <= 2200 && !years.includes(selected))
		years.push(selected);
	return years.sort((a, b) => b - a);
}

type Tone = 'green' | 'amber' | 'red' | 'blue' | 'slate' | 'violet';

/** Email kinds (APPROVAL_REQUIRED, FINANCE_APPROVED, …) in the current language. */
export function emailTypeLabel(i18n: I18n, t: string): string {
	const key = `mailtype.${t}`;
	return hasMsg(i18n, key) ? i18n.t(key as MessageKey) : t;
}

/** Email delivery state: SENT | NOT_SENT | PENDING | PROCESSING | FAILED. */
export function deliveryLabel(i18n: I18n, d: string): string {
	const key = `delivery.${d}`;
	return hasMsg(i18n, key) ? i18n.t(key as MessageKey) : d;
}

export function deliveryTone(d: string): Tone {
	return d === 'SENT' ? 'green' : d === 'FAILED' ? 'red' : d === 'NOT_SENT' ? 'slate' : 'amber';
}

/** Email queue task status: PENDING | PROCESSING | SENT | FAILED. */
export function queueLabel(i18n: I18n, s: string): string {
	const key = `queue.${s}`;
	return hasMsg(i18n, key) ? i18n.t(key as MessageKey) : s;
}

export function queueTone(s: string): Tone {
	return s === 'SENT' ? 'green' : s === 'FAILED' ? 'red' : s === 'PROCESSING' ? 'blue' : 'amber';
}

const hasMsg = (i18n: I18n, key: string) => i18n.t(key as MessageKey) !== key;
