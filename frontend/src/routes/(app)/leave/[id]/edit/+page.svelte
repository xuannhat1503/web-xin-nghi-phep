<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount, untrack } from 'svelte';
	import Alert from '#lib/components/Alert.svelte';
	import DatePicker from '#lib/components/DatePicker.svelte';
	import PageHeader from '#lib/components/PageHeader.svelte';
	import { useI18n, type MessageKey } from '#lib/i18n/index.ts';
	import {
		formatDate,
		formatDateTime,
		formatDuration,
		formatMinutes,
		issueText,
		kindLabel,
		leaveTypeLabel,
		parseDisplayDate,
		reasonLabel,
		weekday
	} from '#lib/format.ts';
	import { detectMode, MODES, modeTimes, type Evaluation, type Mode } from '#lib/leave.ts';
	import { toast } from '#lib/toast.svelte.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const i18n = useI18n();

	// Initial values are taken once; afterwards the form owns the state and auto-saves it.
	const init = untrack(() => data.request);
	const policy = untrack(() => data.context.policy);
	const id = init.id;

	// Name written on this request and its emails only; defaults to the account name.
	let employeeName = $state(init.employee_name ?? untrack(() => data.context.employee.full_name));
	let leaveType = $state(init.leave_type ?? '');
	let startDate = $state(formatDate(init.start_date));
	let endDate = $state(formatDate(init.end_date));
	let mode = $state<Mode>(
		detectMode(init.start_time, init.end_time, init.start_date === init.end_date, policy)
	);
	let startTime = $state(init.start_time ?? policy.work_start);
	let endTime = $state(init.end_time ?? policy.work_end);
	let reasonCode = $state(init.reason_code ?? '');
	let reasonText = $state(init.reason_text);
	let version = $state(init.version);

	type SaveState = 'idle' | 'saving' | 'saved' | 'failed' | 'offline' | 'conflict' | 'locked';
	let saveState = $state<SaveState>('idle');
	let savedAt = $state<string | null>(null);
	let evaluation = $state<Evaluation | null>(untrack(() => data.preview));
	let calculating = $state(false);
	let hydrated = $state(false);

	const halfDay = $derived(mode === 'MORNING' || mode === 'AFTERNOON');
	const reason = $derived(data.context.reasons.find((r) => r.code === reasonCode));
	const issues = $derived(evaluation?.issues ?? []);
	const fieldIssue = (field: string) => issues.find((i) => i.field === field);
	const calc = $derived(evaluation?.calculation ?? null);

	// Quick picks set the times (and a half day is a single day).
	$effect(() => {
		const t = modeTimes(mode, policy);
		if (t) {
			startTime = t.start;
			endTime = t.end;
		}
		if (mode === 'MORNING' || mode === 'AFTERNOON') endDate = startDate;
	});

	function fields() {
		const sd = parseDisplayDate(startDate);
		const ed = halfDay ? sd : (parseDisplayDate(endDate) ?? (endDate.trim() ? null : sd));
		return {
			employee_name: employeeName.trim() || null,
			leave_type: leaveType || null,
			start_date: sd,
			end_date: ed,
			start_time: startTime || null,
			end_time: endTime || null,
			reason_code: reasonCode || null,
			reason_text: reasonText
		};
	}

	const headers = () => ({
		'content-type': 'application/json',
		'x-csrf-token': data.csrfToken ?? ''
	});

	// ---- auto-save (debounced, versioned, retried) ----
	let saveTimer: ReturnType<typeof setTimeout> | undefined;
	let saving = false;
	let pending = false;
	let retries = 0;
	let dirty = false;

	function scheduleSave() {
		if (saveState === 'conflict' || saveState === 'locked') return;
		dirty = true;
		clearTimeout(saveTimer);
		saveTimer = setTimeout(save, 1500);
	}

	async function save(): Promise<boolean> {
		clearTimeout(saveTimer);
		if (saveState === 'conflict' || saveState === 'locked') return false;
		if (saving) {
			pending = true;
			return false;
		}
		saving = true;
		dirty = false;
		saveState = 'saving';
		let ok = false;
		try {
			const res = await fetch(`/api/v1/leave-requests/${id}`, {
				method: 'PATCH',
				headers: headers(),
				body: JSON.stringify({ version, ...fields() })
			});
			if (res.ok) {
				const r = await res.json();
				version = r.version;
				saveState = 'saved';
				savedAt = r.updated_at;
				retries = 0;
				ok = true;
			} else if (res.status === 409) {
				const j = await res.json().catch(() => null);
				saveState = j?.error?.code === 'not_editable' ? 'locked' : 'conflict';
			} else if (res.status >= 500 || res.status === 429) {
				throw new Error(String(res.status));
			} else {
				saveState = 'failed';
			}
		} catch {
			retries++;
			if (retries <= 3) {
				saveState = 'failed';
				dirty = true;
				saveTimer = setTimeout(save, 2000 * 2 ** (retries - 1));
			} else {
				saveState = 'offline';
			}
		} finally {
			saving = false;
			if (pending) {
				pending = false;
				void save();
			}
		}
		return ok;
	}

	// ---- live duration preview (debounced; stale responses ignored) ----
	let previewTimer: ReturnType<typeof setTimeout> | undefined;
	let seq = 0;
	function schedulePreview() {
		clearTimeout(previewTimer);
		previewTimer = setTimeout(preview, 400);
	}
	async function preview() {
		const my = ++seq;
		calculating = true;
		try {
			const res = await fetch('/api/v1/leave-requests/preview', {
				method: 'POST',
				headers: headers(),
				body: JSON.stringify({ request_id: id, ...fields() })
			});
			if (my !== seq) return;
			if (res.ok) evaluation = await res.json();
		} catch {
			// keep the last estimate
		} finally {
			if (my === seq) calculating = false;
		}
	}

	// Any change to the request triggers both.
	$effect(() => {
		JSON.stringify(fields()); // track every field
		if (!untrack(() => hydrated)) return;
		untrack(() => {
			scheduleSave();
			schedulePreview();
		});
	});

	onMount(() => {
		hydrated = true;
		const flush = () => {
			if (document.visibilityState === 'hidden' && dirty) void save();
		};
		document.addEventListener('visibilitychange', flush);
		return () => {
			document.removeEventListener('visibilitychange', flush);
			clearTimeout(saveTimer);
			clearTimeout(previewTimer);
		};
	});

	async function onSubmit(e: SubmitEvent) {
		// With JavaScript we save through the API; the form action is the no-JS fallback.
		e.preventDefault();
		const toReview = (e.submitter as HTMLButtonElement | null)?.value === 'review';
		const ok = dirty || saveState !== 'saved' ? await save() : true;
		if (!ok && saveState !== 'saved') {
			toast(
				i18n.t(saveState === 'conflict' ? 'leave.save.conflict' : 'leave.save.offline'),
				'error',
				6000
			);
			return;
		}
		if (toReview) await goto(resolve('/(app)/leave/[id]/review', { id: String(id) }));
		else toast(i18n.t('leave.save.saved'));
	}

	const statusText = $derived(
		saveState === 'saved' && savedAt
			? `${i18n.t('leave.save.saved')} · ${formatDateTime(savedAt).slice(11)}`
			: i18n.t(`leave.save.${saveState === 'locked' ? 'conflict' : saveState}` as MessageKey)
	);
	const input = 'mt-1 w-full rounded-lg border-slate-300 text-sm';
	const ro = 'mt-1 w-full rounded-lg border-slate-200 bg-slate-50 text-sm text-slate-700';
	const timeInput = 'mt-1 w-full rounded-lg border-slate-300 text-sm tabular-nums';
</script>

<svelte:head><title>{i18n.t('leave.newTitle')} · G-ASIAPACIFIC</title></svelte:head>

<PageHeader
	title={init.request_code
		? i18n.t('req.title', { code: init.request_code })
		: i18n.t('leave.newTitle')}
/>

{#if init.status === 'NEEDS_CHANGES' && init.change_request_reason}
	<div class="mt-4">
		<Alert tone="warning" title={i18n.t('st.NEEDS_CHANGES')}>“{init.change_request_reason}”</Alert>
	</div>
{/if}
{#if saveState === 'conflict' || saveState === 'locked'}
	<div class="mt-4">
		<Alert
			tone="error"
			title={i18n.t(saveState === 'locked' ? 'leave.notEditable' : 'leave.save.conflict')}
		>
			<a
				href={resolve('/(app)/leave/[id]/edit', { id: String(id) })}
				data-sveltekit-reload
				class="font-medium underline">{i18n.t('leave.reload')}</a
			>
		</Alert>
	</div>
{/if}

<form
	method="POST"
	action="?/save"
	class="mt-6 grid gap-6 lg:grid-cols-[1fr_20rem]"
	onsubmit={onSubmit}
	novalidate
>
	<input type="hidden" name="version" value={version} />

	<div class="space-y-6">
		<section class="rounded-xl border border-slate-200 bg-white p-5" aria-labelledby="who-h">
			<h2 id="who-h" class="font-semibold">{i18n.t('leave.applicant')}</h2>
			<div class="mt-4 grid gap-4 sm:grid-cols-2">
				<div>
					<label for="emp-name" class="block text-sm font-medium"
						>{i18n.t('leave.employeeName')}</label
					>
					<input
						id="emp-name"
						name="employee_name"
						class={input}
						maxlength="100"
						bind:value={employeeName}
						aria-invalid={!!fieldIssue('employee_name')}
					/>
					{#if hydrated && fieldIssue('employee_name')}<p class="mt-1 text-sm text-red-700">
							{issueText(i18n, fieldIssue('employee_name')!)}
						</p>{/if}
				</div>
				<div>
					<label for="emp-email" class="block text-sm font-medium"
						>{i18n.t('leave.employeeEmail')}</label
					><input id="emp-email" readonly class={ro} value={data.context.employee.email} />
				</div>
				<div class="sm:col-span-2">
					<label for="mgr" class="block text-sm font-medium">{i18n.t('leave.manager')}</label>
					{#if data.context.manager}
						<input
							id="mgr"
							readonly
							class={ro}
							value="{data.context.manager.full_name} ({data.context.manager.email}){data.context
								.manager.active
								? ''
								: ' ' + i18n.t('leave.managerInactive')}"
						/>
					{:else}
						<p
							id="mgr"
							class="mt-1 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900"
						>
							{i18n.t('leave.noManager')}
						</p>
					{/if}
				</div>
			</div>
		</section>

		<section class="rounded-xl border border-slate-200 bg-white p-5" aria-labelledby="what-h">
			<h2 id="what-h" class="font-semibold">{i18n.t('leave.details')}</h2>
			<div class="mt-4 space-y-4">
				<div>
					<label for="leave_type" class="block text-sm font-medium">{i18n.t('leave.type')}</label>
					<select
						id="leave_type"
						name="leave_type"
						class={input}
						bind:value={leaveType}
						aria-invalid={!!fieldIssue('leave_type')}
					>
						<option value="">{i18n.t('leave.chooseType')}</option>
						{#each i18n.leaveTypes.filter((t) => t.active || t.code === leaveType) as t (t.code)}<option
								value={t.code}>{leaveTypeLabel(i18n, t.code)}</option
							>{/each}
					</select>
					{#if hydrated && fieldIssue('leave_type')}<p class="mt-1 text-sm text-red-700">
							{issueText(i18n, fieldIssue('leave_type')!)}
						</p>{/if}
				</div>

				<fieldset>
					<legend class="text-sm font-medium">{i18n.t('leave.mode')}</legend>
					<div class="mt-2 grid grid-cols-2 gap-2 sm:grid-cols-4">
						{#each MODES as m (m)}
							<label
								class="flex cursor-pointer items-center justify-center rounded-lg border border-slate-300 px-2 py-2 text-center text-sm has-[:checked]:border-brand-600 has-[:checked]:bg-brand-50 has-[:checked]:font-medium has-[:checked]:text-brand-800"
							>
								<input
									type="radio"
									name="mode"
									value={m}
									bind:group={mode}
									class="sr-only"
								/>{i18n.t(`leave.mode.${m}` as MessageKey)}
							</label>
						{/each}
					</div>
				</fieldset>

				<div class="grid gap-4 sm:grid-cols-2">
					<div>
						<DatePicker
							id="start_date"
							name="start_date"
							label={halfDay ? i18n.t('day.date') : i18n.t('leave.fromDate')}
							bind:value={startDate}
							days={data.days}
							workdays={policy.workdays}
							invalid={!!fieldIssue('start_date')}
							describedby="start-err"
						/>
						{#if hydrated && fieldIssue('start_date')}<p
								id="start-err"
								class="mt-1 text-sm text-red-700"
							>
								{issueText(i18n, fieldIssue('start_date')!)}
							</p>{/if}
					</div>
					{#if !halfDay}
						<div>
							<DatePicker
								id="end_date"
								name="end_date"
								label={i18n.t('leave.toDate')}
								bind:value={endDate}
								days={data.days}
								workdays={policy.workdays}
								invalid={!!fieldIssue('end_date')}
								describedby="end-err"
							/>
							{#if hydrated && fieldIssue('end_date')}<p
									id="end-err"
									class="mt-1 text-sm text-red-700"
								>
									{issueText(i18n, fieldIssue('end_date')!)}
								</p>{/if}
						</div>
					{/if}
				</div>

				<div class="grid grid-cols-2 gap-4">
					<div>
						<label for="start_time" class="block text-sm font-medium"
							>{i18n.t('leave.fromTime')}</label
						>
						<input
							id="start_time"
							name="start_time"
							type="text"
							inputmode="numeric"
							placeholder="HH:mm"
							maxlength="5"
							autocomplete="off"
							readonly={mode !== 'CUSTOM'}
							class={mode === 'CUSTOM' ? timeInput : ro + ' tabular-nums'}
							bind:value={startTime}
							aria-invalid={!!fieldIssue('start_time')}
						/>
						{#if hydrated && fieldIssue('start_time')}<p class="mt-1 text-sm text-red-700">
								{issueText(i18n, fieldIssue('start_time')!)}
							</p>{/if}
					</div>
					<div>
						<label for="end_time" class="block text-sm font-medium">{i18n.t('leave.toTime')}</label>
						<input
							id="end_time"
							name="end_time"
							type="text"
							inputmode="numeric"
							placeholder="HH:mm"
							maxlength="5"
							autocomplete="off"
							readonly={mode !== 'CUSTOM'}
							class={mode === 'CUSTOM' ? timeInput : ro + ' tabular-nums'}
							bind:value={endTime}
							aria-invalid={!!fieldIssue('end_time')}
						/>
						{#if hydrated && fieldIssue('end_time')}<p class="mt-1 text-sm text-red-700">
								{issueText(i18n, fieldIssue('end_time')!)}
							</p>{/if}
					</div>
				</div>
				<p class="text-xs text-slate-500">
					{policy.work_start}–{policy.lunch_start} · {policy.lunch_end}–{policy.work_end} · {i18n.t(
						'issue.time_step',
						{ minutes: policy.leave_unit_minutes }
					)}
				</p>

				<div>
					<label for="reason_code" class="block text-sm font-medium">{i18n.t('leave.reason')}</label
					>
					<select
						id="reason_code"
						name="reason_code"
						class={input}
						bind:value={reasonCode}
						aria-invalid={!!fieldIssue('reason_code')}
					>
						<option value="">{i18n.t('leave.chooseReason')}</option>
						{#each data.context.reasons as r (r.code)}<option value={r.code}
								>{reasonLabel(i18n, r.code, r.label)}</option
							>{/each}
					</select>
					{#if hydrated && fieldIssue('reason_code')}<p class="mt-1 text-sm text-red-700">
							{issueText(i18n, fieldIssue('reason_code')!)}
						</p>{/if}
				</div>
				<div>
					<label for="reason_text" class="block text-sm font-medium">
						{i18n.t('leave.reasonText')}
						{#if !reason?.requires_text}<span class="font-normal text-slate-500"
								>{i18n.t('common.optional')}</span
							>{/if}
					</label>
					<textarea
						id="reason_text"
						name="reason_text"
						rows="3"
						maxlength="1000"
						class={input}
						placeholder={i18n.t('leave.reasonTextPh')}
						bind:value={reasonText}
						aria-invalid={!!fieldIssue('reason_text')}></textarea>
					<p class="mt-1 text-right text-xs text-slate-400">{reasonText.length}/1000</p>
					{#if hydrated && fieldIssue('reason_text')}<p class="text-sm text-red-700">
							{issueText(i18n, fieldIssue('reason_text')!)}
						</p>{/if}
				</div>
			</div>
		</section>
	</div>

	<aside class="space-y-4 lg:sticky lg:top-20 lg:self-start">
		<section
			class="rounded-xl border border-brand-100 bg-brand-50 p-5"
			aria-live="polite"
			aria-labelledby="total-h"
		>
			<h2 id="total-h" class="text-sm font-medium text-brand-900">{i18n.t('leave.total')}</h2>
			<p class="mt-1 text-2xl font-semibold text-brand-900 tabular-nums">
				{#if calculating && !calc}{i18n.t('leave.calculating')}{:else if calc}{formatDuration(
						i18n,
						calc.total_minutes,
						evaluation?.minutes_per_day ?? policy.minutes_per_day
					)}{:else}—{/if}
			</p>

			{#if calc && calc.days.length > 0}
				<details class="mt-3 text-sm" open={calc.days.length <= 5}>
					<summary class="cursor-pointer font-medium">{i18n.t('leave.breakdown')}</summary>
					<ul class="mt-2 space-y-1">
						{#each calc.days as d (d.date)}
							<li class="flex justify-between gap-2">
								<span
									>{formatDate(d.date)}
									<span class="text-slate-500">{weekday(i18n, d.date)}</span>{#if d.note}<span
											class="block text-xs text-slate-500"
											>{i18n.t(`leave.note.${d.note}` as MessageKey, { name: d.name ?? '' })}</span
										>{/if}</span
								>
								<span class="tabular-nums">{formatMinutes(i18n, d.minutes)}</span>
							</li>
						{/each}
					</ul>
				</details>
			{/if}
			{#if calc && calc.excluded.length > 0}
				<details class="mt-2 text-sm">
					<summary class="cursor-pointer font-medium"
						>{i18n.t('leave.excluded')} ({calc.excluded.length})</summary
					>
					<ul class="mt-2 space-y-1 text-slate-600">
						{#each calc.excluded as x (x.date)}
							<li>
								{formatDate(x.date)} · {x.reason === 'rest_day'
									? i18n.t('leave.ex.rest_day')
									: `${x.name} (${kindLabel(i18n, x.kind ?? '')})`}
							</li>
						{/each}
					</ul>
				</details>
			{/if}
		</section>

		{#if hydrated && issues.length > 0}
			<Alert tone="warning" title={i18n.t('leave.problems')}>
				<ul class="list-disc space-y-1 pl-5">
					{#each issues as is, i (i)}<li>{issueText(i18n, is)}</li>{/each}
				</ul>
			</Alert>
		{/if}

		<div class="rounded-xl border border-slate-200 bg-white p-4">
			<p
				class="text-sm {saveState === 'failed' ||
				saveState === 'offline' ||
				saveState === 'conflict'
					? 'text-red-700'
					: 'text-slate-600'}"
				role="status"
			>
				{#if saveState === 'saving'}<span
						class="mr-1 inline-block h-2 w-2 animate-pulse rounded-full bg-brand-600"
					></span>{/if}{statusText}
			</p>
			<div class="mt-3 flex flex-col gap-2">
				<button
					name="next"
					value="review"
					class="rounded-lg bg-brand-700 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-800 disabled:opacity-50"
					disabled={saveState === 'conflict' || saveState === 'locked'}
					>{i18n.t('leave.review')}</button
				>
				<button
					name="next"
					value="stay"
					class="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
					disabled={saveState === 'conflict' || saveState === 'locked'}
					>{i18n.t('leave.saveDraft')}</button
				>
			</div>
		</div>
	</aside>
</form>
