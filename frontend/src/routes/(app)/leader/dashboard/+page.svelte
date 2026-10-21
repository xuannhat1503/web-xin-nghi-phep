<script lang="ts">
	import { resolve } from '$app/paths';
	import PageHeader from '#lib/components/PageHeader.svelte';
	import RequestList from '#lib/components/RequestList.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import type { ListItem } from '#lib/leave.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const i18n = useI18n();
	const s = $derived(data.summary);
	const href = (it: ListItem) => resolve('/(app)/leader/requests/[id]', { id: String(it.id) });
	const card = 'block rounded-xl border border-slate-200 bg-white p-5 hover:border-brand-300';
</script>

<svelte:head><title>{i18n.t('ld.title')} · G-ASIAPACIFIC</title></svelte:head>

<PageHeader title={i18n.t('ld.title')} description={i18n.t('ld.desc')} />

<div class="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
	<a class={card} href={resolve('/(app)/leader/requests?status=PENDING&sort=oldest')}>
		<p class="text-sm text-slate-600">{i18n.t('ld.pending')}</p>
		<p class="mt-1 text-3xl font-semibold tabular-nums">{s.counts.PENDING ?? 0}</p>
	</a>
	<div class="rounded-xl border border-slate-200 bg-white p-5">
		<p class="text-sm text-slate-600">{i18n.t('ld.overdue')}</p>
		<p class="mt-1 text-3xl font-semibold tabular-nums {s.overdue ? 'text-red-700' : ''}">
			{s.overdue}
		</p>
	</div>
	<a class={card} href={resolve('/(app)/leader/requests?status=NEEDS_CHANGES')}>
		<p class="text-sm text-slate-600">{i18n.t('ld.needsChanges')}</p>
		<p class="mt-1 text-3xl font-semibold tabular-nums">{s.counts.NEEDS_CHANGES ?? 0}</p>
	</a>
	<a class={card} href={resolve('/(app)/leader/requests?status=APPROVED')}>
		<p class="text-sm text-slate-600">{i18n.t('ld.approved')}</p>
		<p class="mt-1 text-3xl font-semibold tabular-nums">{s.counts.APPROVED ?? 0}</p>
	</a>
</div>

<section class="mt-8">
	<div class="flex items-center justify-between">
		<h2 class="font-semibold">{i18n.t('ld.toReview')}</h2>
		<a
			class="text-sm text-brand-700 hover:underline"
			href={resolve('/(app)/leader/requests?status=PENDING&sort=oldest')}>{i18n.t('ld.viewAll')}</a
		>
	</div>
	<div class="mt-3"><RequestList items={s.pending ?? []} {href} showEmployee /></div>
</section>

<section class="mt-8">
	<h2 class="font-semibold">{i18n.t('ld.awaiting')}</h2>
	<div class="mt-3"><RequestList items={s.needs_changes ?? []} {href} showEmployee /></div>
</section>
