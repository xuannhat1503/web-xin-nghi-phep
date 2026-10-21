<script lang="ts">
	import { page as pageState } from '$app/state';
	import { useI18n } from '#lib/i18n/index.ts';

	// Server-side pagination: plain links that keep every other query parameter.
	let { page, totalPages, totalItems }: { page: number; totalPages: number; totalItems: number } =
		$props();
	const i18n = useI18n();

	function href(p: number): string {
		const u = new URL(pageState.url.href);
		if (p <= 1) u.searchParams.delete('page');
		else u.searchParams.set('page', String(p));
		return u.pathname + u.search;
	}
	const link = 'rounded-md border border-slate-300 bg-white px-3 py-1.5 hover:bg-slate-50';
	const off = 'rounded-md border border-slate-200 px-3 py-1.5 text-slate-400';
</script>

{#if totalPages > 1}
	<nav
		class="mt-4 flex flex-wrap items-center justify-between gap-3 text-sm"
		aria-label={i18n.t('nav.pagination')}
	>
		<p class="text-slate-600">
			{i18n.t('common.pageOf', { page, pages: totalPages, total: totalItems })}
		</p>
		<div class="flex gap-2">
			<!-- eslint-disable svelte/no-navigation-without-resolve -->
			{#if page > 1}
				<a class={link} href={href(page - 1)}>{i18n.t('common.previous')}</a>
			{:else}
				<span class={off} aria-disabled="true">{i18n.t('common.previous')}</span>
			{/if}
			{#if page < totalPages}
				<a class={link} href={href(page + 1)}>{i18n.t('common.next')}</a>
			{:else}
				<span class={off} aria-disabled="true">{i18n.t('common.next')}</span>
			{/if}
			<!-- eslint-enable svelte/no-navigation-without-resolve -->
		</div>
	</nav>
{/if}
