<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import ConfirmButton from '#lib/components/ConfirmButton.svelte';
	import Timeline from '#lib/components/Timeline.svelte';
	import { withToast } from '#lib/forms.ts';
	import Alert from '#lib/components/Alert.svelte';
	import Badge from '#lib/components/Badge.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import {
		formatDate,
		formatDateTime,
		formatDuration,
		formatMinutes,
		kindLabel,
		leaveTypeLabel,
		reasonLabel,
		statusLabel,
		weekday
	} from '#lib/format.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const i18n = useI18n();
	const enhanced = withToast(i18n);
	const r = $derived(data.request);
	const cancellable = $derived(r.status === 'PENDING' || r.status === 'NEEDS_CHANGES');
	const calc = $derived(r.calculation);
	const tones = {
		DRAFT: 'slate',
		PENDING: 'amber',
		NEEDS_CHANGES: 'violet',
		APPROVED: 'green',
		REJECTED: 'red',
		CANCELLED: 'slate'
	} as const;
</script>

<svelte:head
	><title>{r.request_code ?? i18n.t('req.draftTitle')} · G-ASIAPACIFIC</title></svelte:head
>

<div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
	<h1 class="text-2xl font-semibold">
		{r.request_code ? i18n.t('req.title', { code: r.request_code }) : i18n.t('req.draftTitle')}
	</h1>
	<Badge tone={tones[r.status]} label={statusLabel(i18n, r.status)} />
</div>

{#if data.submitted && r.request_code}
	<div class="mt-4">
		<Alert tone="success">{i18n.t('toast.submitted', { code: r.request_code })}</Alert>
	</div>
{/if}
{#if r.calendar_changed}
	<div class="mt-4"><Alert tone="warning">{i18n.t('req.calendarChanged')}</Alert></div>
{/if}
{#if r.status === 'NEEDS_CHANGES' && r.change_request_reason}
	<div class="mt-4">
		<Alert tone="warning" title={statusLabel(i18n, r.status)}>“{r.change_request_reason}”</Alert>
	</div>
{/if}
{#if r.status === 'REJECTED' && r.rejection_reason}
	<div class="mt-4">
		<Alert tone="error" title={statusLabel(i18n, r.status)}>“{r.rejection_reason}”</Alert>
	</div>
{/if}

<div class="mt-6 grid gap-6 lg:grid-cols-[1fr_20rem]">
	<section class="rounded-xl border border-slate-200 bg-white p-5">
		<dl class="grid grid-cols-[auto_1fr] gap-x-6 gap-y-3 text-sm">
			<dt class="text-slate-500">{i18n.t('leave.type')}</dt>
			<dd class="font-medium">{leaveTypeLabel(i18n, r.leave_type) || '—'}</dd>
			<dt class="text-slate-500">{i18n.t('review.from')}</dt>
			<dd>
				{r.start_date
					? `${formatDate(r.start_date)} (${weekday(i18n, r.start_date)}) · ${r.start_time ?? ''}`
					: '—'}
			</dd>
			<dt class="text-slate-500">{i18n.t('review.to')}</dt>
			<dd>
				{r.end_date
					? `${formatDate(r.end_date)} (${weekday(i18n, r.end_date)}) · ${r.end_time ?? ''}`
					: '—'}
			</dd>
			<dt class="text-slate-500">{i18n.t('leave.total')}</dt>
			<dd class="font-semibold">
				{calc
					? formatDuration(i18n, calc.total_minutes, calc.minutes_per_day)
					: r.total_minutes
						? formatMinutes(i18n, r.total_minutes)
						: '—'}
			</dd>
			<dt class="text-slate-500">{i18n.t('leave.reason')}</dt>
			<dd class="break-words whitespace-pre-line">
				{reasonLabel(i18n, r.reason_code, r.reason_label) || '—'}{r.reason_text
					? `\n${r.reason_text}`
					: ''}
			</dd>
			<dt class="text-slate-500">{i18n.t('req.reviewer')}</dt>
			<dd>{r.reviewer ? `${r.reviewer.full_name} (${r.reviewer.email})` : '—'}</dd>
			<dt class="text-slate-500">{i18n.t('req.createdAt')}</dt>
			<dd>{formatDateTime(r.created_at)}</dd>
			{#if r.submitted_at}
				<dt class="text-slate-500">{i18n.t('req.submittedAt')}</dt>
				<dd>{formatDateTime(r.submitted_at)}</dd>
			{/if}
		</dl>
		<div class="mt-6 flex flex-wrap gap-2">
			{#if r.editable}
				<a
					href={resolve('/(app)/leave/[id]/edit', { id: String(r.id) })}
					class="rounded-lg bg-brand-700 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-800"
					>{i18n.t(r.status === 'NEEDS_CHANGES' ? 'req.resubmit' : 'req.editDraft')}</a
				>
			{/if}
			{#if cancellable}
				<form method="POST" action="?/cancel" use:enhance={enhanced}>
					<input type="hidden" name="version" value={r.version} />
					<ConfirmButton
						label={i18n.t('req.cancel')}
						tone="danger"
						title={i18n.t('req.cancelTitle')}
						message={i18n.t('req.cancelMsg')}
						confirmLabel={i18n.t('req.cancel')}
						class="rounded-lg border border-red-300 px-4 py-2.5 text-sm font-medium text-red-700 hover:bg-red-50"
					/>
				</form>
			{/if}
		</div>
	</section>

	{#if calc}
		<aside class="rounded-xl border border-slate-200 bg-white p-5 text-sm lg:self-start">
			<h2 class="font-semibold">{i18n.t('leave.breakdown')}</h2>
			<p class="text-xs text-slate-500">{i18n.t('req.calcFrom', { v: calc.policy_version_no })}</p>
			<ul class="mt-3 space-y-1">
				{#each calc.days as d (d.date)}
					<li class="flex justify-between">
						<span
							>{formatDate(d.date)}
							<span class="text-slate-500">{weekday(i18n, d.date)}</span></span
						><span class="tabular-nums">{formatMinutes(i18n, d.minutes)}</span>
					</li>
				{/each}
			</ul>
			{#if calc.excluded.length}
				<h3 class="mt-4 font-medium">{i18n.t('leave.excluded')}</h3>
				<ul class="mt-1 space-y-0.5 text-slate-600">
					{#each calc.excluded as x (x.date)}<li>
							{formatDate(x.date)} · {x.reason === 'rest_day'
								? i18n.t('leave.ex.rest_day')
								: `${x.name} (${kindLabel(i18n, x.kind ?? '')})`}
						</li>{/each}
				</ul>
			{/if}
		</aside>
	{/if}
</div>

{#if data.history.events.length}
	<div class="mt-6"><Timeline events={data.history.events} versions={data.history.versions} /></div>
{/if}
