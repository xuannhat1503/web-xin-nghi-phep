<script lang="ts">
	import { resolve } from '$app/paths';
	import Alert from '#lib/components/Alert.svelte';
	import PageHeader from '#lib/components/PageHeader.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import {
		formatDate,
		formatDuration,
		issueText,
		kindLabel,
		leaveTypeLabel,
		reasonLabel,
		weekday
	} from '#lib/format.ts';
	import type { Issue } from '#lib/leave.ts';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	const i18n = useI18n();
	const r = $derived(data.review.request);
	const ev = $derived(data.review.evaluation);
	const calc = $derived(ev.calculation);
	const issues = $derived<Issue[]>(
		((form as { issues?: Issue[] } | null)?.issues?.length
			? (form as { issues: Issue[] }).issues
			: ev.issues) ?? []
	);
	const blocked = $derived(issues.length > 0 || !ev.can_submit);
	const editHref = $derived(resolve('/(app)/leave/[id]/edit', { id: String(r.id) }));
	let submitting = $state(false);
</script>

<svelte:head><title>{i18n.t('review.title')} · G-ASIAPACIFIC</title></svelte:head>

<PageHeader title={i18n.t('review.title')} description={i18n.t('review.desc')} />

{#if form?.error && !issues.length}
	<div class="mt-4"><Alert tone="error">{i18n.server(form.error)}</Alert></div>
{/if}
{#if blocked}
	<div class="mt-4">
		<Alert tone="error" title={i18n.t('review.blocking')}>
			<ul class="list-disc space-y-1 pl-5">
				{#each issues as is, i (i)}<li>{issueText(i18n, is)}</li>{/each}
			</ul>
		</Alert>
	</div>
{/if}

<div class="mt-6 grid gap-6 lg:grid-cols-2">
	<section class="rounded-xl border border-slate-200 bg-white p-5">
		<dl class="grid grid-cols-[auto_1fr] gap-x-6 gap-y-3 text-sm">
			<dt class="text-slate-500">{i18n.t('leave.type')}</dt>
			<dd class="font-medium">{leaveTypeLabel(i18n, r.leave_type)}</dd>
			<dt class="text-slate-500">{i18n.t('review.from')}</dt>
			<dd>
				{formatDate(r.start_date)} ({r.start_date ? weekday(i18n, r.start_date) : ''}) · {r.start_time}
			</dd>
			<dt class="text-slate-500">{i18n.t('review.to')}</dt>
			<dd>
				{formatDate(r.end_date)} ({r.end_date ? weekday(i18n, r.end_date) : ''}) · {r.end_time}
			</dd>
			<dt class="text-slate-500">{i18n.t('leave.total')}</dt>
			<dd class="text-lg font-semibold text-brand-900">
				{calc ? formatDuration(i18n, calc.total_minutes, ev.minutes_per_day) : '—'}
			</dd>
			<dt class="text-slate-500">{i18n.t('leave.excluded')}</dt>
			<dd>
				{#if calc?.excluded.length}
					<ul class="space-y-0.5">
						{#each calc.excluded as x (x.date)}<li>
								{formatDate(x.date)} · {x.reason === 'rest_day'
									? i18n.t('leave.ex.rest_day')
									: `${x.name} (${kindLabel(i18n, x.kind ?? '')})`}
							</li>{/each}
					</ul>
				{:else}{i18n.t('review.none')}{/if}
			</dd>
			<dt class="text-slate-500">{i18n.t('review.reviewer')}</dt>
			<dd>
				{data.review.manager
					? `${data.review.manager.full_name} (${data.review.manager.email})`
					: i18n.t('leave.noManager')}
			</dd>
			<dt class="text-slate-500">{i18n.t('leave.reason')}</dt>
			<dd class="break-words whitespace-pre-line">
				{reasonLabel(i18n, r.reason_code, r.reason_label)}{r.reason_text
					? `\n${r.reason_text}`
					: ''}
			</dd>
		</dl>

		<div class="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:justify-between">
			<a
				href={editHref}
				class="rounded-lg border border-slate-300 px-4 py-2.5 text-center text-sm font-medium text-slate-700 hover:bg-slate-50"
				>{i18n.t('review.back')}</a
			>
			<form method="POST" action="?/submit" onsubmit={() => (submitting = true)}>
				<input type="hidden" name="version" value={r.version} />
				<input type="hidden" name="idempotency_key" value={data.idempotencyKey} />
				<button
					class="w-full rounded-lg bg-emerald-700 px-5 py-2.5 text-sm font-semibold text-white hover:bg-emerald-800 disabled:opacity-50 sm:w-auto"
					disabled={blocked || submitting}
				>
					{i18n.t('review.confirm')}
				</button>
			</form>
		</div>
	</section>

	<section class="rounded-xl border border-slate-200 bg-white p-5" aria-labelledby="mail-h">
		<h2 id="mail-h" class="font-semibold">{i18n.t('review.email')}</h2>
		<p class="mt-1 text-xs text-slate-500">{i18n.t('review.emailNote')}</p>
		{#if data.review.email_preview}
			{@const m = data.review.email_preview}
			<dl class="mt-3 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-sm">
				<dt class="text-slate-500">{i18n.t('review.emailTo')}</dt>
				<dd class="break-all">{m.to}</dd>
				<dt class="text-slate-500">{i18n.t('review.emailCc')}</dt>
				<dd class="break-all">{m.cc.length ? m.cc.join(', ') : '—'}</dd>
				<dt class="text-slate-500">{i18n.t('review.emailSubject')}</dt>
				<dd class="font-medium break-words">{m.subject}</dd>
			</dl>
			<pre
				class="mt-3 max-h-96 overflow-auto rounded-lg bg-slate-50 p-3 text-xs leading-relaxed whitespace-pre-wrap text-slate-800">{m.body}</pre>
		{:else}
			<p class="mt-3 text-sm text-slate-600">—</p>
		{/if}
	</section>
</div>
