<script lang="ts">
	import { resolve } from '$app/paths';
	import Badge from '#lib/components/Badge.svelte';
	import PageHeader from '#lib/components/PageHeader.svelte';
	import { useI18n, type MessageKey } from '#lib/i18n/index.ts';
	import { formatDateTime, queueLabel } from '#lib/format.ts';
	import type { PageProps } from './$types';
	import type { SystemStatus } from './+page.server.ts';

	let { data }: PageProps = $props();
	const i18n = useI18n();
	const s = $derived(data.status);
	const email = $derived(data.email);
	const now = $derived(new Date(s.server_time).getTime());
	const card = 'rounded-xl border border-slate-200 bg-white p-5';

	// Background jobs and how often they run (seconds); a job is "late" after about 3 missed runs.
	const JOBS: Record<string, number> = {
		email_outbox: 30,
		leader_reminders: 300,
		session_cleanup: 600,
		idempotency_cleanup: 3600,
		email_log_retention: 21600
	};
	type Job = NonNullable<SystemStatus['workers']>[number];
	type JobState = 'ok' | 'failed' | 'late' | 'never';
	function jobState(w: Job): JobState {
		if (!w.last_run_at) return 'never';
		const ok = w.last_success_at ? new Date(w.last_success_at).getTime() : 0;
		if (w.last_error && ok < new Date(w.last_run_at).getTime()) return 'failed';
		const every = JOBS[w.name] ?? 3600;
		return now - ok > (every * 3 + 60) * 1000 ? 'late' : 'ok';
	}
	const stateTone = { ok: 'green', failed: 'red', late: 'amber', never: 'slate' } as const;
	const jobLabel = (n: string) =>
		n in JOBS ? i18n.t(`job.${n}` as MessageKey) : n.replaceAll('_', ' ');
	const jobHelp = (n: string) => (n in JOBS ? i18n.t(`job.${n}.help` as MessageKey) : '');

	function ago(iso: string | null): string {
		if (!iso) return i18n.t('sys.never');
		const min = Math.max(0, Math.round((now - new Date(iso).getTime()) / 60000));
		if (min < 1) return i18n.t('sys.justNow');
		if (min < 60) return i18n.t('sys.minAgo', { n: min });
		const h = Math.round(min / 60);
		return h < 48 ? i18n.t('sys.hourAgo', { n: h }) : formatDateTime(iso);
	}

	const failedMails = $derived(s.outbox?.FAILED ?? 0);
	const waitingMails = $derived((s.outbox?.PENDING ?? 0) + (s.outbox?.PROCESSING ?? 0));
	const problems = $derived(
		[
			s.database !== 'ok' && 'sys.p.db',
			s.migration && !s.migration.up_to_date && 'sys.p.migration',
			(s.workers ?? []).some((w) => ['failed', 'late'].includes(jobState(w))) && 'sys.p.jobs',
			failedMails > 0 && 'sys.p.mail'
		].filter(Boolean) as MessageKey[]
	);
</script>

<svelte:head><title>{i18n.t('sys.title')} · G-ASIAPACIFIC</title></svelte:head>

<PageHeader title={i18n.t('sys.title')} description={i18n.t('sys.desc')} />

<section
	class="mt-6 rounded-xl border p-5 {problems.length
		? 'border-amber-300 bg-amber-50'
		: 'border-emerald-300 bg-emerald-50'}"
	aria-live="polite"
>
	<p class="font-semibold {problems.length ? 'text-amber-900' : 'text-emerald-900'}">
		{problems.length ? i18n.t('sys.needsAttention') : i18n.t('sys.allGood')}
	</p>
	{#if problems.length}
		<ul class="mt-2 list-disc space-y-1 pl-5 text-sm text-amber-900">
			{#each problems as p (p)}<li>{i18n.t(p)}</li>{/each}
		</ul>
	{/if}
	<p class="mt-2 text-xs text-slate-600">
		{i18n.t('sys.checkedAt', { t: formatDateTime(s.server_time) })}
	</p>
</section>

<div class="mt-6 grid gap-4 md:grid-cols-3">
	<div class={card}>
		<p class="text-sm font-medium">{i18n.t('sys.db')}</p>
		<p class="mt-2">
			<Badge
				tone={s.database === 'ok' ? 'green' : 'red'}
				label={i18n.t(s.database === 'ok' ? 'sys.ok' : 'sys.fail')}
			/>
		</p>
		<p class="mt-2 text-xs text-slate-600">
			{i18n.t(s.database === 'ok' ? 'sys.dbOkHelp' : 'sys.dbFailHelp')}
		</p>
	</div>
	<div class={card}>
		<p class="text-sm font-medium">{i18n.t('sys.migration')}</p>
		{#if s.migration}
			<p class="mt-2">
				<Badge
					tone={s.migration.up_to_date ? 'green' : 'amber'}
					label={i18n.t(s.migration.up_to_date ? 'sys.upToDate' : 'sys.outdated')}
				/>
			</p>
			<p class="mt-2 text-xs text-slate-600">
				{i18n.t('sys.migrationHelp', { v: s.migration.version, latest: s.migration.latest })}
			</p>
		{:else}
			<p class="mt-2 text-sm text-slate-500">—</p>
		{/if}
	</div>
	<div class={card}>
		<p class="text-sm font-medium">{i18n.t('sys.emailMode')}</p>
		<p class="mt-2">
			<Badge
				tone={s.email_mode === 'LIVE' ? 'green' : 'slate'}
				label={i18n.t(s.email_mode === 'LIVE' ? 'sys.mailOn' : 'sys.mailOff')}
			/>
		</p>
		<p class="mt-2 text-xs text-slate-600">
			{#if email}
				{email.smtp_configured
					? i18n.t('set.smtpOk', { from: email.smtp_from })
					: i18n.t('set.smtpMissing')}
			{/if}
			<a href={resolve('/(app)/admin/settings?tab=email')} class="text-brand-700 hover:underline"
				>{i18n.t('sys.changeMail')}</a
			>
		</p>
	</div>
</div>

<section class="{card} mt-6" aria-labelledby="queue-h">
	<div class="flex flex-wrap items-baseline justify-between gap-2">
		<h2 id="queue-h" class="font-semibold">{i18n.t('sys.outbox')}</h2>
		<a href={resolve('/(app)/admin/notifications')} class="text-sm text-brand-700 hover:underline"
			>{i18n.t('sys.openQueue')}</a
		>
	</div>
	<p class="mt-1 text-sm text-slate-600">{i18n.t('sys.outboxHelp')}</p>
	<div class="mt-3 grid gap-3 sm:grid-cols-3">
		<div class="rounded-lg bg-slate-50 p-3">
			<p class="text-xs text-slate-600">{i18n.t('sys.mailWaiting')}</p>
			<p class="text-xl font-semibold tabular-nums">{waitingMails}</p>
		</div>
		<div class="rounded-lg bg-slate-50 p-3">
			<p class="text-xs text-slate-600">{queueLabel(i18n, 'SENT')}</p>
			<p class="text-xl font-semibold tabular-nums">{s.outbox?.SENT ?? 0}</p>
		</div>
		<div class="rounded-lg p-3 {failedMails ? 'bg-red-50' : 'bg-slate-50'}">
			<p class="text-xs text-slate-600">{queueLabel(i18n, 'FAILED')}</p>
			<p class="text-xl font-semibold tabular-nums {failedMails ? 'text-red-700' : ''}">
				{failedMails}
			</p>
		</div>
	</div>
</section>

<section class="{card} mt-6" aria-labelledby="jobs-h">
	<h2 id="jobs-h" class="font-semibold">{i18n.t('sys.workers')}</h2>
	<p class="mt-1 text-sm text-slate-600">{i18n.t('sys.workersHelp')}</p>
	<ul class="mt-3 divide-y divide-slate-100 text-sm">
		{#each s.workers ?? [] as w (w.name)}
			{@const st = jobState(w)}
			<li class="flex flex-col gap-1 py-3 sm:flex-row sm:items-center sm:justify-between">
				<div class="min-w-0">
					<p class="font-medium">{jobLabel(w.name)}</p>
					{#if jobHelp(w.name)}<p class="text-xs text-slate-500">{jobHelp(w.name)}</p>{/if}
				</div>
				<div class="flex flex-wrap items-center gap-2 sm:justify-end">
					<span class="text-xs text-slate-600"
						>{i18n.t('sys.lastOk')}: {ago(w.last_success_at)}</span
					>
					<Badge tone={stateTone[st]} label={i18n.t(`sys.job.${st}` as MessageKey)} />
				</div>
			</li>
		{/each}
	</ul>
</section>

<section class="{card} mt-6" aria-labelledby="err-h">
	<h2 id="err-h" class="font-semibold">{i18n.t('sys.errors')}</h2>
	{#if !s.recent_errors?.length}
		<p class="mt-2 text-sm text-slate-600">{i18n.t('sys.noErrors')}</p>
	{:else}
		<p class="mt-1 text-sm text-slate-600">{i18n.t('sys.errorsHelp')}</p>
		<ul class="mt-3 divide-y divide-slate-100 text-sm">
			{#each s.recent_errors as e, i (i)}
				<li class="py-2">
					<p><span class="font-medium">{jobLabel(e.component)}</span> · {i18n.server(e.message)}</p>
					<p class="text-xs text-slate-500 tabular-nums">
						{formatDateTime(e.created_at)} ({ago(e.created_at)})
					</p>
				</li>
			{/each}
		</ul>
	{/if}
</section>
