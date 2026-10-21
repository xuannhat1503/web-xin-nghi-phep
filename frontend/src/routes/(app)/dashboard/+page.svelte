<script lang="ts">
	import { resolve } from '$app/paths';
	import Badge from '#lib/components/Badge.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import StatusBadge from '#lib/components/StatusBadge.svelte';
	import { formatDate, kindLabel, statusLabel, weekday } from '#lib/format.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const i18n = useI18n();
	const user = $derived(data.user!);
</script>

<svelte:head><title>{i18n.t('nav.dashboard')} · G-ASIAPACIFIC</title></svelte:head>

<div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
	<div class="min-w-0">
		<h1 class="text-2xl font-semibold">{i18n.t('dash.welcome', { name: user.full_name })}</h1>
		<p class="mt-1 text-sm text-slate-600">
			{i18n.t('dash.signedInAs', { email: user.email })}
			<span class="ml-2"
				><Badge
					tone={user.role === 'ADMIN' ? 'blue' : 'slate'}
					label={i18n.t(user.role === 'ADMIN' ? 'role.ADMIN' : 'role.EMPLOYEE')}
				/></span
			>
		</p>
	</div>
	<a
		href={resolve('/(app)/leave/new')}
		class="inline-flex items-center justify-center rounded-lg bg-brand-700 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-800"
	>
		{i18n.t('dash.newRequest')}
	</a>
</div>

<div class="mt-6 grid gap-6 lg:grid-cols-2">
	<section class="rounded-xl border border-slate-200 bg-white p-5" aria-labelledby="mine-h">
		<div class="flex items-center justify-between">
			<h2 id="mine-h" class="font-semibold">{i18n.t('dash.myRequests')}</h2>
			<a href={resolve('/(app)/leave/history')} class="text-sm text-brand-700 hover:underline"
				>{i18n.t('dash.viewHistory')}</a
			>
		</div>
		<div class="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
			{#each ['PENDING', 'NEEDS_CHANGES', 'APPROVED', 'REJECTED'] as st (st)}
				<a
					href={resolve(`/(app)/leave/history?status=${st}`)}
					class="rounded-lg bg-slate-50 p-3 hover:bg-slate-100"
				>
					<p class="text-xs text-slate-600">{statusLabel(i18n, st)}</p>
					<p class="text-xl font-semibold tabular-nums">{data.summary.counts[st] ?? 0}</p>
				</a>
			{/each}
		</div>
		<h3 class="mt-5 text-sm font-medium">{i18n.t('dash.recent')}</h3>
		{#if !data.summary.recent?.length}
			<p class="mt-2 text-sm text-slate-600">{i18n.t('dash.noRecent')}</p>
		{:else}
			<ul class="mt-2 divide-y divide-slate-100 text-sm">
				{#each data.summary.recent as r (r.id)}
					<li>
						<a
							href={resolve('/(app)/leave/[id]', { id: String(r.id) })}
							class="flex items-center justify-between gap-2 py-2 hover:text-brand-700"
						>
							<span class="min-w-0 truncate"
								>{r.request_code} · {formatDate(r.start_date)}{r.end_date &&
								r.end_date !== r.start_date
									? ` → ${formatDate(r.end_date)}`
									: ''}</span
							>
							<StatusBadge status={r.status} />
						</a>
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	<section class="rounded-xl border border-slate-200 bg-white p-5" aria-labelledby="hol-h">
		<h2 id="hol-h" class="font-semibold">
			{i18n.t('dash.upcoming', { year: data.calendar.year })}
		</h2>
		{#if !data.calendar.year_confirmed}
			<p class="mt-2 text-sm text-amber-800">
				{i18n.t('dash.notConfirmed', { year: data.calendar.year })}
			</p>
		{/if}
		{#if data.upcoming.length === 0}
			<p class="mt-2 text-sm text-slate-600">{i18n.t('dash.noUpcoming')}</p>
		{:else}
			<ul class="mt-3 divide-y divide-slate-100 text-sm">
				{#each data.upcoming as d (d.date)}
					<li class="flex items-center justify-between gap-3 py-2">
						<span class="min-w-0">
							<span class="font-medium tabular-nums">{formatDate(d.date)}</span>
							<span class="text-slate-500"
								>({weekday(i18n, d.date)}{d.start_time
									? `, ${d.start_time}–${d.end_time}`
									: ''})</span
							>
							<span class="block truncate text-slate-700">{d.name}</span>
						</span>
						<Badge
							tone={d.kind === 'COMPANY_HOLIDAY'
								? 'violet'
								: d.kind === 'COMPENSATORY_HOLIDAY'
									? 'amber'
									: 'red'}
							label={kindLabel(i18n, d.kind)}
						/>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</div>
