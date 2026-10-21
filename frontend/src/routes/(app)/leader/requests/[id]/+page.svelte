<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import Alert from '#lib/components/Alert.svelte';
	import ConfirmButton from '#lib/components/ConfirmButton.svelte';
	import StatusBadge from '#lib/components/StatusBadge.svelte';
	import Timeline from '#lib/components/Timeline.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import {
		formatDate,
		formatDateTime,
		formatDuration,
		formatMinutes,
		kindLabel,
		leaveTypeLabel,
		reasonLabel,
		weekday
	} from '#lib/format.ts';
	import { withToast, type ActionData } from '#lib/forms.ts';
	import type { PageProps } from './$types';

	let { data, form: rawForm }: PageProps = $props();
	const form = $derived(rawForm as ActionData);
	const i18n = useI18n();
	const enhanced = withToast(i18n);
	const r = $derived(data.request);
	const calc = $derived(r.calculation);
	let comment = $state('');
	const btn = 'rounded-lg px-4 py-2.5 text-sm font-medium';
</script>

<svelte:head><title>{r.request_code} · {i18n.t('lr.title')} · G-ASIAPACIFIC</title></svelte:head>

<a href={resolve('/(app)/leader/requests')} class="text-sm text-brand-700 hover:underline"
	>{i18n.t('lrd.back')}</a
>
<div class="mt-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
	<h1 class="text-2xl font-semibold">{i18n.t('req.title', { code: r.request_code ?? '' })}</h1>
	<StatusBadge status={r.status} />
</div>

{#if form?.error || form?.errorKey}
	<div class="mt-4">
		<Alert tone="error">{form.error ? i18n.server(form.error) : i18n.msg(form.errorKey)}</Alert>
	</div>
{/if}
{#if r.calendar_changed}
	<div class="mt-4"><Alert tone="warning">{i18n.t('req.calendarChanged')}</Alert></div>
{/if}

<div class="mt-6 grid gap-6 lg:grid-cols-[1fr_22rem]">
	<section class="rounded-xl border border-slate-200 bg-white p-5">
		<dl class="grid grid-cols-[auto_1fr] gap-x-6 gap-y-3 text-sm">
			<dt class="text-slate-500">{i18n.t('lrd.employee')}</dt>
			<dd class="font-medium">
				{r.employee.full_name} <span class="font-normal text-slate-500">({r.employee.email})</span>
			</dd>
			<dt class="text-slate-500">{i18n.t('leave.type')}</dt>
			<dd>{leaveTypeLabel(i18n, r.leave_type)}</dd>
			<dt class="text-slate-500">{i18n.t('review.from')}</dt>
			<dd>
				{r.start_date
					? `${formatDate(r.start_date)} (${weekday(i18n, r.start_date)}) · ${r.start_time}`
					: '—'}
			</dd>
			<dt class="text-slate-500">{i18n.t('review.to')}</dt>
			<dd>
				{r.end_date
					? `${formatDate(r.end_date)} (${weekday(i18n, r.end_date)}) · ${r.end_time}`
					: '—'}
			</dd>
			<dt class="text-slate-500">{i18n.t('leave.total')}</dt>
			<dd class="font-semibold">
				{calc
					? formatDuration(i18n, calc.total_minutes, calc.minutes_per_day)
					: formatMinutes(i18n, r.total_minutes)}
			</dd>
			<dt class="text-slate-500">{i18n.t('leave.reason')}</dt>
			<dd class="break-words whitespace-pre-line">
				{reasonLabel(i18n, r.reason_code, r.reason_label)}{r.reason_text
					? `\n${r.reason_text}`
					: ''}
			</dd>
			<dt class="text-slate-500">{i18n.t('req.submittedAt')}</dt>
			<dd>{formatDateTime(r.submitted_at)}</dd>
		</dl>
		{#if calc?.excluded.length}
			<h3 class="mt-5 text-sm font-medium">{i18n.t('leave.excluded')}</h3>
			<ul class="mt-1 space-y-0.5 text-sm text-slate-600">
				{#each calc.excluded as x (x.date)}<li>
						{formatDate(x.date)} · {x.reason === 'rest_day'
							? i18n.t('leave.ex.rest_day')
							: `${x.name} (${kindLabel(i18n, x.kind ?? '')})`}
					</li>{/each}
			</ul>
		{/if}
	</section>

	<section
		class="rounded-xl border border-slate-200 bg-white p-5 lg:self-start"
		aria-labelledby="dec-h"
	>
		<h2 id="dec-h" class="font-semibold">{i18n.t('lrd.decideTitle')}</h2>
		{#if r.status !== 'PENDING'}
			<p class="mt-2 text-sm text-slate-600">{i18n.t('lrd.closed')}</p>
			{#if r.rejection_reason}<p class="mt-2 text-sm">“{r.rejection_reason}”</p>{/if}
			{#if r.status === 'NEEDS_CHANGES' && r.change_request_reason}<p class="mt-2 text-sm">
					“{r.change_request_reason}”
				</p>{/if}
		{:else}
			<form method="POST" action="?/decide" use:enhance={enhanced} class="mt-3 space-y-3">
				<input type="hidden" name="version" value={r.version} />
				<div>
					<label for="comment" class="block text-sm font-medium">{i18n.t('lrd.comment')}</label>
					<textarea
						id="comment"
						name="comment"
						rows="3"
						maxlength="1000"
						class="mt-1 w-full rounded-lg border-slate-300 text-sm"
						placeholder={i18n.t('lrd.commentPh')}
						bind:value={comment}></textarea>
				</div>
				<div class="flex flex-col gap-2">
					<ConfirmButton
						name="decision"
						value="approve"
						label={i18n.t('lrd.approve')}
						title={i18n.t('lrd.approveTitle')}
						message={i18n.t('lrd.approveMsg')}
						confirmLabel={i18n.t('lrd.approve')}
						class="{btn} bg-emerald-700 text-white hover:bg-emerald-800"
					/>
					<ConfirmButton
						name="decision"
						value="request_changes"
						label={i18n.t('lrd.changes')}
						title={i18n.t('lrd.changesTitle')}
						message={comment ? `“${comment}”` : i18n.t('lrd.commentRequired')}
						confirmLabel={i18n.t('lrd.changes')}
						class="{btn} border border-violet-300 text-violet-800 hover:bg-violet-50"
					/>
					<ConfirmButton
						name="decision"
						value="reject"
						tone="danger"
						label={i18n.t('lrd.reject')}
						title={i18n.t('lrd.rejectTitle')}
						message={comment ? `“${comment}”` : i18n.t('lrd.commentRequired')}
						confirmLabel={i18n.t('lrd.reject')}
						class="{btn} border border-red-300 text-red-700 hover:bg-red-50"
					/>
				</div>
			</form>
		{/if}
	</section>
</div>

<div class="mt-6"><Timeline events={r.events} versions={r.versions} /></div>
