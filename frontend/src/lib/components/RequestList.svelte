<script lang="ts">
	import StatusBadge from '#lib/components/StatusBadge.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import { formatDate, formatDateTime, formatMinutes, leaveTypeLabel } from '#lib/format.ts';
	import type { ListItem } from '#lib/leave.ts';

	// Responsive list: cards on small screens, a table from md up.
	let {
		items,
		href,
		showEmployee = false
	}: { items: ListItem[]; href: (it: ListItem) => string; showEmployee?: boolean } = $props();
	const i18n = useI18n();
	const period = (it: ListItem) =>
		it.start_date
			? `${formatDate(it.start_date)} ${it.start_time ?? ''}` +
				(it.end_date ? ` → ${formatDate(it.end_date)} ${it.end_time ?? ''}` : '')
			: '—';
</script>

{#if items.length === 0}
	<div
		class="rounded-lg border border-dashed border-slate-300 bg-white p-8 text-center text-sm text-slate-600"
	>
		{i18n.t('list.empty')}
	</div>
{:else}
	<!-- eslint-disable svelte/no-navigation-without-resolve -->
	<ul class="space-y-3 md:hidden">
		{#each items as it (it.id)}
			<li>
				<a
					href={href(it)}
					class="hover:border-brand-300 block rounded-lg border border-slate-200 bg-white p-4"
				>
					<div class="flex items-start justify-between gap-2">
						<div class="min-w-0">
							<p class="font-medium">
								{it.request_code ?? i18n.t('list.draft')} · {leaveTypeLabel(i18n, it.leave_type) ||
									'—'}
							</p>
							{#if showEmployee}<p class="truncate text-sm text-slate-700">
									{it.employee_name}
								</p>{/if}
						</div>
						<StatusBadge status={it.status} />
					</div>
					<p class="mt-1 text-sm text-slate-600">{period(it)}</p>
					<p class="text-sm text-slate-600">
						{it.total_minutes ? formatMinutes(i18n, it.total_minutes) : ''}
					</p>
				</a>
			</li>
		{/each}
	</ul>
	<div class="hidden overflow-hidden rounded-lg border border-slate-200 bg-white md:block">
		<table class="w-full text-left text-sm">
			<thead class="bg-slate-50 text-xs tracking-wide text-slate-500 uppercase">
				<tr>
					<th class="px-4 py-3 font-medium">{i18n.t('list.code')}</th>
					{#if showEmployee}<th class="px-4 py-3 font-medium">{i18n.t('list.employee')}</th>{/if}
					<th class="px-4 py-3 font-medium">{i18n.t('filter.type')}</th>
					<th class="px-4 py-3 font-medium">{i18n.t('list.period')}</th>
					<th class="px-4 py-3 font-medium">{i18n.t('list.duration')}</th>
					<th class="px-4 py-3 font-medium">{i18n.t('list.submitted')}</th>
					<th class="px-4 py-3 font-medium">{i18n.t('list.status')}</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-slate-100">
				{#each items as it (it.id)}
					<tr class="hover:bg-slate-50">
						<td class="px-4 py-3"
							><a class="font-medium text-brand-700 hover:underline" href={href(it)}
								>{it.request_code ?? i18n.t('list.draft')}</a
							></td
						>
						{#if showEmployee}<td class="px-4 py-3">{it.employee_name}</td>{/if}
						<td class="px-4 py-3">{leaveTypeLabel(i18n, it.leave_type) || '—'}</td>
						<td class="px-4 py-3 tabular-nums">{period(it)}</td>
						<td class="px-4 py-3 tabular-nums"
							>{it.total_minutes ? formatMinutes(i18n, it.total_minutes) : '—'}</td
						>
						<td class="px-4 py-3 text-slate-600">{formatDateTime(it.submitted_at) || '—'}</td>
						<td class="px-4 py-3"><StatusBadge status={it.status} /></td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	<!-- eslint-enable svelte/no-navigation-without-resolve -->
{/if}
