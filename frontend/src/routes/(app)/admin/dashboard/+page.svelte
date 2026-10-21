<script lang="ts">
	import { resolve } from '$app/paths';
	import BarChart from '#lib/components/BarChart.svelte';
	import DonutChart from '#lib/components/DonutChart.svelte';
	import PageHeader from '#lib/components/PageHeader.svelte';
	import { useI18n, type MessageKey } from '#lib/i18n/index.ts';
	import { formatDuration, leaveTypeLabel, statusLabel, yearOptions } from '#lib/format.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const i18n = useI18n();
	const a = $derived(data.analytics);
	const k = $derived(a.kpis);
	const years = $derived(yearOptions(a.year));
	const months = Array.from({ length: 12 }, (_, i) => String(i + 1));

	const statusColors: Record<string, string> = {
		PENDING: '#d97706',
		NEEDS_CHANGES: '#7c3aed',
		APPROVED: '#059669',
		REJECTED: '#dc2626',
		CANCELLED: '#94a3b8'
	};
	// Leave types are Admin-managed, so colors come from a fixed palette in display order.
	const palette = [
		'#2f6db5',
		'#ea580c',
		'#64748b',
		'#059669',
		'#a855f7',
		'#d97706',
		'#0891b2',
		'#db2777'
	];
	const slices = $derived(
		Object.keys(statusColors).map((s) => ({
			label: statusLabel(i18n, s),
			value: k.by_status[s] ?? 0,
			color: statusColors[s]!
		}))
	);
	const typeSeries = $derived(
		i18n.leaveTypes
			.filter((t) => t.active || a.approved_by_month_type.some((r) => r.leave_type === t.code))
			.map(({ code: t }, idx) => ({
				label: leaveTypeLabel(i18n, t),
				color: palette[idx % palette.length]!,
				values: Array.from(
					{ length: 12 },
					(_, i) =>
						Math.round(
							((a.approved_by_month_type.find((r) => r.month === i + 1 && r.leave_type === t)
								?.minutes ?? 0) /
								60) *
								10
						) / 10
				)
			}))
	);
	const card = 'rounded-xl border border-slate-200 bg-white p-5';
	const kpi = 'mt-1 text-2xl font-semibold tabular-nums';
</script>

<svelte:head><title>{i18n.t('adash.title')} · G-ASIAPACIFIC</title></svelte:head>

<PageHeader title={i18n.t('adash.title')} description={i18n.t('an.note')}>
	{#snippet actions()}
		<form method="GET" class="flex gap-2">
			<label for="a-year" class="sr-only">{i18n.t('an.year')}</label>
			<select
				id="a-year"
				name="year"
				class="rounded-lg border-slate-300 text-sm"
				onchange={(e) => e.currentTarget.form?.requestSubmit()}
			>
				{#each years as y (y)}<option value={y} selected={a.year === y}>{y}</option>{/each}
			</select>
			<label for="a-month" class="sr-only">{i18n.t('an.month')}</label>
			<select
				id="a-month"
				name="month"
				class="rounded-lg border-slate-300 text-sm"
				onchange={(e) => e.currentTarget.form?.requestSubmit()}
			>
				<option value="">{i18n.t('an.allYear')}</option>
				{#each Array.from({ length: 12 }, (_, i) => i + 1) as m (m)}<option
						value={m}
						selected={a.month === m}>{i18n.t(`month.${m}` as MessageKey)}</option
					>{/each}
			</select>
			<button class="rounded-lg border border-slate-300 bg-white px-3 text-sm"
				>{i18n.t('filter.apply')}</button
			>
		</form>
	{/snippet}
</PageHeader>

<div class="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
	<div class={card}>
		<p class="text-sm text-slate-600">{i18n.t('an.total')}</p>
		<p class={kpi}>{k.total}</p>
	</div>
	<div class={card}>
		<p class="text-sm text-slate-600">{statusLabel(i18n, 'PENDING')}</p>
		<p class={kpi}>{k.by_status.PENDING ?? 0}</p>
	</div>
	<div class={card}>
		<p class="text-sm text-slate-600">{statusLabel(i18n, 'APPROVED')}</p>
		<p class={kpi}>{k.by_status.APPROVED ?? 0}</p>
	</div>
	<div class={card}>
		<p class="text-sm text-slate-600">{statusLabel(i18n, 'REJECTED')}</p>
		<p class={kpi}>{k.by_status.REJECTED ?? 0}</p>
	</div>
	<div class={card}>
		<p class="text-sm text-slate-600">{statusLabel(i18n, 'CANCELLED')}</p>
		<p class={kpi}>{k.by_status.CANCELLED ?? 0}</p>
	</div>
	<div class={card}>
		<p class="text-sm text-slate-600">{statusLabel(i18n, 'NEEDS_CHANGES')}</p>
		<p class={kpi}>{k.by_status.NEEDS_CHANGES ?? 0}</p>
	</div>
	<div class={card}>
		<p class="text-sm text-slate-600">{i18n.t('an.approvedTime')}</p>
		<p class="{kpi} text-lg">{formatDuration(i18n, k.approved_minutes, a.minutes_per_day)}</p>
	</div>
	<div class={card}>
		<p class="text-sm text-slate-600">{i18n.t('an.avgProcessing')}</p>
		<p class={kpi}>
			{k.avg_processing_hours === null
				? '—'
				: i18n.t('common.hours', { h: k.avg_processing_hours })}
		</p>
	</div>
	<div class={card}>
		<p class="text-sm text-slate-600">{i18n.t('an.overdue')}</p>
		<p class="{kpi} {k.overdue_pending ? 'text-red-700' : ''}">{k.overdue_pending}</p>
	</div>
	<a class="{card} hover:border-brand-300" href={resolve('/(app)/admin/employees?status=inactive')}
		><p class="text-sm text-slate-600">{i18n.t('adash.pending')}</p>
		<p class={kpi}>{k.inactive_employees}</p></a
	>
	<a
		class="{card} hover:border-brand-300"
		href={resolve('/(app)/admin/employees?status=active&no_manager=1')}
		><p class="text-sm text-slate-600">{i18n.t('adash.noManager')}</p>
		<p class={kpi}>{k.employees_without_manager}</p></a
	>
	<a class="{card} hover:border-brand-300" href={resolve('/(app)/admin/holidays')}>
		<p class="text-sm text-slate-600">{i18n.t('an.unconfirmed')}</p>
		<p class="{kpi} text-lg {k.unconfirmed_years.length ? 'text-amber-700' : 'text-emerald-700'}">
			{k.unconfirmed_years.length ? k.unconfirmed_years.join(', ') : i18n.t('an.allConfirmed')}
		</p>
	</a>
</div>

<div class="mt-6 grid gap-6 lg:grid-cols-2">
	<section class={card}>
		<h2 class="font-semibold">{i18n.t('an.monthly', { year: a.year })}</h2>
		<div class="mt-3">
			{#if a.monthly_counts.some((n) => n > 0)}
				<BarChart
					title={i18n.t('an.monthly', { year: a.year })}
					labels={months}
					series={[{ label: i18n.t('an.total'), color: '#24599a', values: a.monthly_counts }]}
				/>
			{:else}<p class="text-sm text-slate-500">{i18n.t('an.noData')}</p>{/if}
		</div>
	</section>
	<section class={card}>
		<h2 class="font-semibold">
			{i18n.t('an.statusShare')}{a.month ? ` · ${i18n.t(`month.${a.month}` as MessageKey)}` : ''}
			{a.year}
		</h2>
		<div class="mt-3">
			<DonutChart title={i18n.t('an.statusShare')} {slices} empty={i18n.t('an.noData')} />
		</div>
	</section>
</div>

<section class="{card} mt-6">
	<h2 class="font-semibold">{i18n.t('an.approvedByType')} · {a.year}</h2>
	<div class="mt-3">
		{#if a.approved_by_month_type.length}
			<BarChart
				title={i18n.t('an.approvedByType')}
				labels={months}
				series={typeSeries}
				format={(n) => String(n)}
			/>
		{:else}<p class="text-sm text-slate-500">{i18n.t('an.noData')}</p>{/if}
	</div>
</section>
