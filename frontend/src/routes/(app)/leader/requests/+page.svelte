<script lang="ts">
	import { resolve } from '$app/paths';
	import { navigating } from '$app/state';
	import PageHeader from '#lib/components/PageHeader.svelte';
	import Pagination from '#lib/components/Pagination.svelte';
	import RequestFilters from '#lib/components/RequestFilters.svelte';
	import RequestList from '#lib/components/RequestList.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import type { ListItem } from '#lib/leave.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const i18n = useI18n();
	const href = (it: ListItem) => resolve('/(app)/leader/requests/[id]', { id: String(it.id) });
</script>

<svelte:head><title>{i18n.t('lr.title')} · G-ASIAPACIFIC</title></svelte:head>

<PageHeader title={i18n.t('lr.title')} />
<div class="mt-6"><RequestFilters showDraft={false} /></div>
<section
	class="mt-4 transition-opacity {navigating.to ? 'opacity-60' : ''}"
	aria-live="polite"
	aria-busy={!!navigating.to}
>
	<RequestList items={data.list.items ?? []} {href} showEmployee />
	<Pagination
		page={data.list.page}
		totalPages={data.list.total_pages}
		totalItems={data.list.total_items}
	/>
</section>
