<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import Alert from '#lib/components/Alert.svelte';
	import CalendarDayFields from '#lib/components/CalendarDayFields.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import { toast } from '#lib/toast.svelte.ts';
	import type { PageProps } from './$types';

	// The update action lives on the holidays page; it redirects back there on success.
	let { data }: PageProps = $props();
	const i18n = useI18n();
	let error = $state<string | null>(null);
	let fields = $state<Record<string, string>>({});
	const back = $derived(resolve(`/(app)/admin/holidays?year=${data.year}`));
</script>

<svelte:head><title>{i18n.t('hol.editTitle')} · {data.day.name} · G-ASIAPACIFIC</title></svelte:head
>

<a href={back} class="text-sm text-brand-700 hover:underline"
	>{i18n.t('hol.back', { year: data.year })}</a
>
<h1 class="mt-3 text-2xl font-semibold">{i18n.t('hol.editTitle')}</h1>

<div class="mt-4 max-w-2xl">
	<Alert tone="info">{i18n.t('hol.editInfo', { year: data.year })}</Alert>
	{#if error}<div class="mt-3"><Alert tone="error">{i18n.server(error)}</Alert></div>{/if}

	<form
		method="POST"
		action="/admin/holidays?/update&year={data.year}"
		class="mt-6 space-y-4 rounded-xl border border-slate-200 bg-white p-5"
		use:enhance={() =>
			async ({ result, update }) => {
				if (result.type === 'failure') {
					error = (result.data?.error as string) ?? null;
					fields = (result.data?.fields as Record<string, string>) ?? {};
					if (error) toast(i18n.server(error), 'error', 6000);
				}
				await update({ reset: false });
			}}
	>
		<input type="hidden" name="day_id" value={data.day.id} />
		<input type="hidden" name="version" value={data.day.version} />
		<CalendarDayFields day={data.day} {fields} />
		<div class="flex gap-2">
			<button
				class="rounded-lg bg-brand-700 px-4 py-2 text-sm font-medium text-white hover:bg-brand-800"
				>{i18n.t('hol.saveDraft')}</button
			>
			<a
				href={back}
				class="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
				>{i18n.t('common.cancel')}</a
			>
		</div>
	</form>
</div>
