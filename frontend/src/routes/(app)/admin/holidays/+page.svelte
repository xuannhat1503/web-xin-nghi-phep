<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import Alert from '#lib/components/Alert.svelte';
	import Badge from '#lib/components/Badge.svelte';
	import CalendarDayFields from '#lib/components/CalendarDayFields.svelte';
	import ConfirmButton from '#lib/components/ConfirmButton.svelte';
	import PageHeader from '#lib/components/PageHeader.svelte';
	import YearCalendar from '#lib/components/YearCalendar.svelte';
	import { useI18n, type MessageKey } from '#lib/i18n/index.ts';
	import {
		codedText,
		formatDate,
		formatDateTime,
		KINDS,
		kindLabel,
		weekday,
		yearOptions
	} from '#lib/format.ts';
	import { withToast, type ActionData } from '#lib/forms.ts';
	import type { PageProps } from './$types';

	let { data, form: rawForm }: PageProps = $props();
	const form = $derived(rawForm as ActionData);
	const i18n = useI18n();
	const enhanced = withToast(i18n);
	const v = $derived(data.view);
	const days = $derived(
		v.days.filter(
			(d) => (!data.kind || d.kind === data.kind) && (!data.status || d.status === data.status)
		)
	);
	const drafts = $derived(v.days.filter((d) => d.status === 'DRAFT').length);
	const affected = $derived(Object.values(v.affected_requests ?? {}).reduce((a, b) => a + b, 0));
	const years = $derived(
		[...new Set([...yearOptions(v.year), ...(v.years ?? []).map((y) => y.year), v.year])].sort()
	);

	function href(changes: Record<string, string | null>): string {
		const u = new URL(page.url.href);
		for (const [k, val] of Object.entries(changes)) {
			if (val === null || val === '') u.searchParams.delete(k);
			else u.searchParams.set(k, val);
		}
		u.searchParams.delete('updated');
		return u.pathname + u.search;
	}
	const kindTone: Record<string, 'red' | 'violet' | 'amber' | 'blue'> = {
		NATIONAL_HOLIDAY: 'red',
		COMPANY_HOLIDAY: 'violet',
		COMPENSATORY_HOLIDAY: 'amber',
		MAKEUP_WORKDAY: 'blue'
	};
	const statusText = (s: string) => i18n.t(`status.${s}` as MessageKey);
	const btn = 'rounded-lg px-3 py-2 text-sm font-medium';
</script>

<svelte:head><title>{i18n.t('hol.title')} {v.year} · G-ASIAPACIFIC</title></svelte:head>

<PageHeader title={i18n.t('hol.title')} description={i18n.t('hol.desc')}>
	{#snippet actions()}
		<form method="POST" action="?/seed&year={v.year}" use:enhance={enhanced}>
			<button
				class="{btn} border border-slate-300 bg-white text-slate-700 hover:bg-slate-50"
				disabled={v.status === 'CONFIRMED'}
			>
				{i18n.t('hol.suggest')}
			</button>
		</form>
		{#if v.exists && v.status !== 'CONFIRMED'}
			<form method="POST" action="?/confirmYear&year={v.year}" use:enhance={enhanced}>
				<input type="hidden" name="version" value={v.version} />
				<ConfirmButton
					label={i18n.t('hol.confirmYear', { year: v.year })}
					title={i18n.t('hol.confirmYearTitle', { year: v.year })}
					message={i18n.t('hol.confirmYearMsg', { n: drafts })}
					confirmLabel={i18n.t('hol.confirmCalendar')}
					class="{btn} bg-emerald-700 text-white hover:bg-emerald-800"
				/>
			</form>
		{/if}
	{/snippet}
</PageHeader>

<!-- eslint-disable svelte/no-navigation-without-resolve -->
<nav class="mt-5 flex flex-wrap items-center gap-2" aria-label={i18n.t('hol.yearNav')}>
	{#each years as y (y)}
		<a
			href={href({ year: String(y) })}
			aria-current={y === v.year ? 'page' : undefined}
			class="rounded-full border px-3 py-1 text-sm {y === v.year
				? 'border-brand-600 bg-brand-50 font-medium text-brand-800'
				: 'border-slate-300 bg-white hover:bg-slate-50'}">{y}</a
		>
	{/each}
	<form method="GET" class="flex items-center gap-1">
		<label for="year-jump" class="sr-only">{i18n.t('hol.goToYear')}</label>
		<input
			id="year-jump"
			name="year"
			type="number"
			min="2000"
			max="2200"
			placeholder={i18n.t('hol.year')}
			class="w-24 rounded-lg border-slate-300 py-1 text-sm"
		/>
		<button class="rounded-lg border border-slate-300 bg-white px-2 py-1 text-sm"
			>{i18n.t('hol.go')}</button
		>
	</form>
</nav>

<div class="mt-4 flex flex-wrap items-center gap-3 rounded-xl border border-slate-200 bg-white p-4">
	<p class="text-sm">
		{i18n.t('hol.status')}
		<Badge
			tone={v.status === 'CONFIRMED' ? 'green' : 'amber'}
			label={statusText(v.exists ? v.status : 'NOT_SET_UP')}
		/>
	</p>
	{#if v.confirmed_at}
		<p class="text-sm text-slate-600">
			{i18n.t('hol.confirmedBy', { at: formatDateTime(v.confirmed_at), by: v.confirmed_by ?? '' })}
		</p>
	{/if}
	{#if affected > 0}<p class="text-sm text-slate-600">
			{i18n.t('hol.affected', { n: affected, year: v.year })}
		</p>{/if}
</div>

{#if page.url.searchParams.has('updated')}
	<div class="mt-4"><Alert tone="success">{i18n.t('hol.updated')}</Alert></div>
{/if}
{#if form?.error || form?.errorKey}
	<div class="mt-4">
		<Alert tone="error">{form.error ? i18n.server(form.error) : i18n.msg(form.errorKey)}</Alert>
	</div>
{:else if form?.success}
	<div class="mt-4"><Alert tone="success">{i18n.msg(form.success)}</Alert></div>
{/if}

{#if v.warnings.length}
	<div class="mt-4">
		<Alert tone="warning" title={i18n.t('hol.check')}>
			<ul class="list-disc space-y-1 pl-5">
				{#each v.warnings as w (w.code + w.message)}<li>{codedText(i18n, 'warn', w)}</li>{/each}
			</ul>
		</Alert>
	</div>
{/if}

<div class="mt-6 flex flex-wrap items-center justify-between gap-3">
	<div
		class="flex rounded-lg border border-slate-300 bg-white p-0.5 text-sm"
		role="tablist"
		aria-label={i18n.t('hol.views')}
	>
		<a
			role="tab"
			aria-selected={data.mode === 'list'}
			href={href({ view: null })}
			class="rounded-md px-3 py-1 {data.mode === 'list' ? 'bg-brand-700 text-white' : ''}"
			>{i18n.t('hol.viewList')}</a
		>
		<a
			role="tab"
			aria-selected={data.mode === 'calendar'}
			href={href({ view: 'calendar' })}
			class="rounded-md px-3 py-1 {data.mode === 'calendar' ? 'bg-brand-700 text-white' : ''}"
			>{i18n.t('hol.viewCalendar')}</a
		>
	</div>
	<form method="GET" class="flex flex-wrap items-center gap-2 text-sm">
		<input type="hidden" name="year" value={v.year} />
		{#if data.mode === 'calendar'}<input type="hidden" name="view" value="calendar" />{/if}
		<label for="f-kind" class="sr-only">{i18n.t('hol.filterType')}</label>
		<select
			id="f-kind"
			name="kind"
			class="rounded-lg border-slate-300 py-1 text-sm"
			onchange={(e) => e.currentTarget.form?.requestSubmit()}
		>
			<option value="">{i18n.t('hol.allTypes')}</option>
			{#each KINDS as k (k)}<option value={k} selected={data.kind === k}
					>{kindLabel(i18n, k)}</option
				>{/each}
		</select>
		<label for="f-status" class="sr-only">{i18n.t('hol.filterStatus')}</label>
		<select
			id="f-status"
			name="status"
			class="rounded-lg border-slate-300 py-1 text-sm"
			onchange={(e) => e.currentTarget.form?.requestSubmit()}
		>
			<option value="">{i18n.t('hol.allStatuses')}</option>
			<option value="DRAFT" selected={data.status === 'DRAFT'}>{statusText('DRAFT')}</option>
			<option value="CONFIRMED" selected={data.status === 'CONFIRMED'}
				>{statusText('CONFIRMED')}</option
			>
		</select>
		<button class="rounded-lg border border-slate-300 bg-white px-2 py-1"
			>{i18n.t('common.apply')}</button
		>
	</form>
</div>
<!-- eslint-enable svelte/no-navigation-without-resolve -->

<section class="mt-4">
	{#if data.mode === 'calendar'}
		<YearCalendar year={v.year} {days} workdays={v.workdays} />
	{:else if days.length === 0}
		<div
			class="rounded-lg border border-dashed border-slate-300 bg-white p-8 text-center text-sm text-slate-600"
		>
			{v.days.length ? i18n.t('hol.emptyFilter') : i18n.t('hol.emptyYear', { year: v.year })}
		</div>
	{:else}
		<ul
			class="divide-y divide-slate-100 overflow-hidden rounded-xl border border-slate-200 bg-white"
		>
			{#each days as d (d.id)}
				<li class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center">
					<div class="w-32 shrink-0">
						<p class="font-medium tabular-nums">{formatDate(d.date)}</p>
						<p class="text-xs text-slate-500">
							{weekday(i18n, d.date)}{d.start_time ? ` · ${d.start_time}–${d.end_time}` : ''}
						</p>
					</div>
					<div class="min-w-0 flex-1">
						<p class="font-medium">{d.name}</p>
						<div class="mt-1 flex flex-wrap gap-1.5">
							<Badge tone={kindTone[d.kind]} label={kindLabel(i18n, d.kind)} />
							<Badge
								tone={d.status === 'CONFIRMED' ? 'green' : 'slate'}
								label={statusText(d.status)}
							/>
							{#if d.seeded && !d.admin_modified}<Badge
									tone="slate"
									label={i18n.t('hol.suggested')}
								/>{/if}
						</div>
						{#if d.note}<p class="mt-1 text-xs text-slate-500">{d.note}</p>{/if}
					</div>
					<div class="flex flex-wrap gap-2">
						<a
							class="{btn} border border-slate-300 text-slate-700 hover:bg-slate-50"
							href={resolve(`/(app)/admin/holidays/days/[id]?year=${v.year}`, { id: String(d.id) })}
							>{i18n.t('common.edit')}<span class="sr-only"> {d.name}</span></a
						>
						<form method="POST" action="?/setStatus&year={v.year}" use:enhance={enhanced}>
							<input type="hidden" name="day_id" value={d.id} />
							<input type="hidden" name="version" value={d.version} />
							{#if d.status === 'DRAFT'}
								<button
									name="op"
									value="confirm"
									class="{btn} bg-emerald-700 text-white hover:bg-emerald-800"
									>{i18n.t('hol.confirm')}<span class="sr-only"> {d.name}</span></button
								>
							{:else}
								<ConfirmButton
									name="op"
									value="unconfirm"
									label={i18n.t('hol.revert')}
									title={i18n.t('hol.revertTitle', { name: d.name })}
									message={i18n.t('hol.revertMsg', { year: v.year })}
									class="{btn} border border-slate-300 text-slate-700 hover:bg-slate-50"
								/>
							{/if}
						</form>
						{#if d.status === 'DRAFT'}
							<form method="POST" action="?/delete&year={v.year}" use:enhance={enhanced}>
								<input type="hidden" name="day_id" value={d.id} />
								<input type="hidden" name="version" value={d.version} />
								<ConfirmButton
									label={i18n.t('common.delete')}
									tone="danger"
									title={i18n.t('hol.deleteTitle', { name: d.name })}
									message={i18n.t('hol.deleteMsg')}
									confirmLabel={i18n.t('common.delete')}
									class="{btn} border border-red-300 text-red-700 hover:bg-red-50"
								/>
							</form>
						{/if}
					</div>
				</li>
			{/each}
		</ul>
	{/if}
</section>

<details class="mt-6 rounded-xl border border-slate-200 bg-white p-5" open={!!form?.values}>
	<summary class="cursor-pointer font-semibold">{i18n.t('hol.add')}</summary>
	<form method="POST" action="?/create&year={v.year}" use:enhance={enhanced} class="mt-4 space-y-4">
		<CalendarDayFields defaultDate="{v.year}-01-01" fields={form?.fields ?? {}} />
		<button class="{btn} bg-brand-700 text-white hover:bg-brand-800"
			>{i18n.t('hol.addDraft')}</button
		>
	</form>
</details>

{#if v.events.length}
	<details class="mt-6 rounded-xl border border-slate-200 bg-white p-5">
		<summary class="cursor-pointer font-semibold"
			>{i18n.t('hol.changelog', { n: v.events.length })}</summary
		>
		<ol class="mt-3 space-y-2 text-sm">
			{#each v.events as ev, i (i)}
				<li class="flex flex-wrap gap-x-2">
					<span class="text-slate-500 tabular-nums">{formatDateTime(ev.created_at)}</span>
					<span class="font-medium">{ev.action.replaceAll('_', ' ').toLowerCase()}</span>
					<span class="text-slate-600"
						>{i18n.t('common.by', {
							name: ev.actor === 'System' ? i18n.t('common.system') : ev.actor
						})}</span
					>
				</li>
			{/each}
		</ol>
	</details>
{/if}
