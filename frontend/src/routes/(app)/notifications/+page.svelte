<script lang="ts">
	import { onMount } from 'svelte';
	import PageHeader from '#lib/components/PageHeader.svelte';
	import Pagination from '#lib/components/Pagination.svelte';
	import { useI18n, hasKey } from '#lib/i18n/index.ts';
	import { formatDateTime } from '#lib/format.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const i18n = useI18n();
	function text(type: string, raw: string): string {
		let params: Record<string, string>;
		try {
			params = JSON.parse(raw) ?? {};
		} catch {
			params = {};
		}
		const key = `notif.${type}`;
		return hasKey(key) ? i18n.t(key, params) : type;
	}
	// Seeing the list counts as reading it. Items keep their "new" highlight for this visit.
	onMount(() => {
		if (!data.unread) return;
		void fetch('/api/v1/notifications/read-all', {
			method: 'POST',
			headers: { 'x-csrf-token': data.csrfToken ?? '' }
		}).catch(() => {});
	});
</script>

<svelte:head><title>{i18n.t('notif.title')} · G-ASIAPACIFIC</title></svelte:head>

<PageHeader title={i18n.t('notif.title')} />

<section class="mt-4">
	{#if !data.list.items?.length}
		<div
			class="rounded-lg border border-dashed border-slate-300 bg-white p-8 text-center text-sm text-slate-600"
		>
			{i18n.t('notif.empty')}
		</div>
	{:else}
		<ul
			class="divide-y divide-slate-100 overflow-hidden rounded-xl border border-slate-200 bg-white"
		>
			{#each data.list.items as n (n.id)}
				<li
					class="flex flex-col gap-2 p-4 sm:flex-row sm:items-center {n.read_at
						? ''
						: 'bg-brand-50/60'}"
				>
					<div class="min-w-0 flex-1">
						<p class="text-sm {n.read_at ? 'text-slate-700' : 'font-medium text-slate-900'}">
							{#if !n.read_at}<span
									class="mr-1.5 inline-block h-2 w-2 rounded-full bg-brand-600"
									aria-hidden="true"
								></span>{/if}{text(n.type, n.params)}
						</p>
						<p class="text-xs text-slate-500">{formatDateTime(n.created_at)}</p>
					</div>
					<div class="flex gap-2">
						<!-- Links are app-relative paths produced by the API; access is checked again on the target page. -->
						<!-- eslint-disable svelte/no-navigation-without-resolve -->
						<a
							href={n.link}
							class="rounded-lg bg-brand-700 px-3 py-1.5 text-sm text-white hover:bg-brand-800"
							>{i18n.t('notif.open')}</a
						>
						<!-- eslint-enable svelte/no-navigation-without-resolve -->
					</div>
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
