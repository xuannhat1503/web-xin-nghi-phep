<script lang="ts">
	import { resolve } from '$app/paths';
	import PageHeader from '#lib/components/PageHeader.svelte';
	import Pagination from '#lib/components/Pagination.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import { formatDateTime } from '#lib/format.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const i18n = useI18n();
</script>

<svelte:head><title>{i18n.t('mock.title')} · G-ASIAPACIFIC</title></svelte:head>

<PageHeader title={i18n.t('mock.title')} description={i18n.t('mock.desc')} />

<section class="mt-6">
	{#if !data.list.items?.length}
		<div
			class="rounded-lg border border-dashed border-slate-300 bg-white p-8 text-center text-sm text-slate-600"
		>
			{i18n.t('mock.empty')}
		</div>
	{:else}
		<ul
			class="divide-y divide-slate-100 overflow-hidden rounded-xl border border-slate-200 bg-white"
		>
			{#each data.list.items as m (m.id)}
				<li>
					<a
						href={resolve('/(app)/admin/mock-email/[id]', { id: String(m.id) })}
						class="block p-4 hover:bg-slate-50"
					>
						<p class="text-sm font-medium break-words">{m.subject}</p>
						<p class="text-xs text-slate-500">
							{m.email_type} · {i18n.t('mock.to')}: {m.to} · {formatDateTime(m.created_at)}
						</p>
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
