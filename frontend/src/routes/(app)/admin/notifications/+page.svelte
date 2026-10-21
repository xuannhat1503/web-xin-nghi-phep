<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import Badge from '#lib/components/Badge.svelte';
	import PageHeader from '#lib/components/PageHeader.svelte';
	import Pagination from '#lib/components/Pagination.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import { emailTypeLabel, formatDateTime, queueLabel, queueTone } from '#lib/format.ts';
	import { withToast } from '#lib/forms.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const i18n = useI18n();
	const enhanced = withToast(i18n);
	const filters = ['', 'PENDING', 'PROCESSING', 'SENT', 'FAILED'];
</script>

<svelte:head><title>{i18n.t('outbox.title')} · G-ASIAPACIFIC</title></svelte:head>

<PageHeader title={i18n.t('outbox.title')} description={i18n.t('outbox.desc')} />

<nav class="mt-5 flex flex-wrap gap-2" aria-label={i18n.t('outbox.status')}>
	{#each filters as f (f)}
		<a
			href={resolve(f ? `/(app)/admin/notifications?status=${f}` : '/(app)/admin/notifications')}
			aria-current={data.status === f ? 'true' : undefined}
			class="rounded-full border px-3 py-1 text-sm {data.status === f
				? 'border-brand-600 bg-brand-50 text-brand-800'
				: 'border-slate-300 bg-white'}"
		>
			{f ? queueLabel(i18n, f) : i18n.t('outbox.all')}{f ? ` (${data.counts[f] ?? 0})` : ''}
		</a>
	{/each}
</nav>

<section class="mt-4">
	{#if !data.page.items?.length}
		<div
			class="rounded-lg border border-dashed border-slate-300 bg-white p-8 text-center text-sm text-slate-600"
		>
			{i18n.t('outbox.empty')}
		</div>
	{:else}
		<div class="overflow-x-auto rounded-lg border border-slate-200 bg-white">
			<table class="w-full min-w-[40rem] text-left text-sm">
				<thead class="bg-slate-50 text-xs tracking-wide text-slate-500 uppercase">
					<tr>
						<th class="px-4 py-3 font-medium">{i18n.t('outbox.event')}</th>
						<th class="px-4 py-3 font-medium">{i18n.t('list.code')}</th>
						<th class="px-4 py-3 font-medium">{i18n.t('outbox.status')}</th>
						<th class="px-4 py-3 font-medium">{i18n.t('outbox.attempts')}</th>
						<th class="px-4 py-3 font-medium">{i18n.t('outbox.next')}</th>
						<th class="px-4 py-3 font-medium">{i18n.t('outbox.error')}</th>
						<th class="px-4 py-3"></th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each data.page.items as o (o.id)}
						<tr>
							<td class="px-4 py-3"
								>{emailTypeLabel(i18n, o.event_type)}
								<p class="text-xs text-slate-500">{formatDateTime(o.created_at)}</p></td
							>
							<td class="px-4 py-3">{o.request_code ?? '—'}</td>
							<td class="px-4 py-3"
								><Badge tone={queueTone(o.status)} label={queueLabel(i18n, o.status)} /></td
							>
							<td class="px-4 py-3 tabular-nums">{o.attempts}</td>
							<td class="px-4 py-3"
								>{o.status === 'SENT'
									? formatDateTime(o.processed_at)
									: formatDateTime(o.next_retry_at)}</td
							>
							<td class="px-4 py-3 text-slate-600"
								>{o.last_error ? i18n.server(o.last_error) : ''}</td
							>
							<td class="px-4 py-3 text-right">
								{#if o.status === 'FAILED'}
									<form method="POST" action="?/retry" use:enhance={enhanced}>
										<input type="hidden" name="outbox_id" value={o.id} />
										<button
											class="rounded-lg border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-50"
											>{i18n.t('outbox.retry')}</button
										>
									</form>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
	<Pagination
		page={data.page.page}
		totalPages={data.page.total_pages}
		totalItems={data.page.total_items}
	/>
</section>
