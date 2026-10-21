<script lang="ts">
	import { resolve } from '$app/paths';
	import Badge from '#lib/components/Badge.svelte';
	import PageHeader from '#lib/components/PageHeader.svelte';
	import Pagination from '#lib/components/Pagination.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import { deliveryLabel, deliveryTone, emailTypeLabel, formatDateTime } from '#lib/format.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const i18n = useI18n();
</script>

<svelte:head><title>{i18n.t('mail.title')} · G-ASIAPACIFIC</title></svelte:head>

<PageHeader title={i18n.t('mail.title')} description={i18n.t('mail.desc')} />

<section class="mt-6">
	{#if !data.list.items?.length}
		<div
			class="rounded-lg border border-dashed border-slate-300 bg-white p-8 text-center text-sm text-slate-600"
		>
			{i18n.t('mail.empty')}
		</div>
	{:else}
		<ul
			class="divide-y divide-slate-100 overflow-hidden rounded-xl border border-slate-200 bg-white"
		>
			{#each data.list.items as m (m.id)}
				<li>
					<a
						href={resolve('/(app)/admin/email-log/[id]', { id: String(m.id) })}
						class="flex flex-col gap-1 p-4 hover:bg-slate-50 sm:flex-row sm:items-center sm:justify-between"
					>
						<div class="min-w-0">
							<p class="text-sm font-medium break-words">{m.subject}</p>
							<p class="text-xs text-slate-500">
								{emailTypeLabel(i18n, m.email_type)} · {i18n.t('mail.to')}: {m.to} · {formatDateTime(
									m.created_at
								)}
							</p>
						</div>
						<Badge tone={deliveryTone(m.delivery)} label={deliveryLabel(i18n, m.delivery)} />
					</a>
				</li>
			{/each}
		</ul>
	{/if}
	<Pagination
		page={data.list.page}
		totalPages={data.list.total_pages}
		totalItems={data.list.total_items}
	/>
</section>
