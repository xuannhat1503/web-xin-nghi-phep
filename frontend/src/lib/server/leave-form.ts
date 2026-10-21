import { parseDisplayDate } from '#lib/format.ts';
import { modeTimes, type LeavePolicy, type Mode, MODES } from '#lib/leave.ts';

/** API "fields" payload for drafts and previews. */
export interface LeaveFields {
	leave_type: string | null;
	start_date: string | null;
	end_date: string | null;
	start_time: string | null;
	end_time: string | null;
	reason_code: string | null;
	reason_text: string;
}

const s = (f: FormData, k: string) => {
	const v = f.get(k);
	return typeof v === 'string' ? v.trim() : '';
};

/**
 * Reads the leave form as submitted without JavaScript: dd/MM/yyyy dates, quick-pick mode
 * (whose times come from the policy) or custom HH:mm times. The API validates everything again.
 */
export function fieldsFromForm(f: FormData, policy: LeavePolicy): LeaveFields {
	const mode = (MODES as string[]).includes(s(f, 'mode')) ? (s(f, 'mode') as Mode) : 'CUSTOM';
	const start = parseDisplayDate(s(f, 'start_date'));
	let end = parseDisplayDate(s(f, 'end_date'));
	let startTime = s(f, 'start_time') || null;
	let endTime = s(f, 'end_time') || null;
	const preset = modeTimes(mode, policy);
	if (preset) {
		startTime = preset.start;
		endTime = preset.end;
		if (mode !== 'FULL') end = start; // half days are single-day requests
	}
	if (start && !end) end = start;
	return {
		leave_type: s(f, 'leave_type') || null,
		start_date: start,
		end_date: end,
		start_time: startTime,
		end_time: endTime,
		reason_code: s(f, 'reason_code') || null,
		reason_text: s(f, 'reason_text')
	};
}
