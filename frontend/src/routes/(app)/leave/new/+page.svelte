<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import Alert from '#lib/components/Alert.svelte';
	import ConfirmButton from '#lib/components/ConfirmButton.svelte';
	import PageHeader from '#lib/components/PageHeader.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import { formatDate, formatDateTime, formatMinutes, leaveTypeLabel } from '#lib/format.ts';
	import { withToast, type ActionData } from '#lib/forms.ts';
	import type { PageProps } from './$types';

	let { data, form: rawForm }: PageProps = $props();
	const form = $derived(rawForm as ActionData);
	const i18n = useI18n();
	const enhanced = withToast(i18n);
</script>

<svelte:head><title>{i18n.t('leave.newTitle')} · G-ASIAPACIFIC</title></svelte:head>

<PageHeader title={i18n.t('leave.newTitle')}>
	{#snippet actions()}
		<form method="POST" action="?/create">
			<button
				class="rounded-lg bg-brand-700 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-800"
				>{i18n.t('leave.startNew')}</button
			>
		</form>
	{/snippet}
</PageHeader>

{#if form?.error}<div class="mt-4"><Alert tone="error">{i18n.server(form.error)}</Alert></div>{/if}

<section class="mt-6 rounded-xl border border-slate-200 bg-white p-5" aria-labelledby="drafts-h">
	<h2 id="drafts-h" class="font-semibold">{i18n.t('leave.drafts')}</h2>
	{#if data.drafts.length === 0}
		<p class="mt-2 text-sm text-slate-600">{i18n.t('leave.noDrafts')}</p>
	{:else}
		<ul class="mt-3 divide-y divide-slate-100">
			{#each data.drafts as d (d.id)}
				<li class="flex flex-col gap-3 py-3 sm:flex-row sm:items-center sm:justify-between">
					<div class="min-w-0">
						<p class="font-medium">
							{d.leave_type ? leaveTypeLabel(i18n, d.leave_type) : i18n.t('leave.untitled')}
						</p>
						<p class="text-sm text-slate-600">
							{#if d.start_date}{formatDate(d.start_date)}{d.end_date && d.end_date !== d.start_date
									? ` – ${formatDate(d.end_date)}`
									: ''}{/if}
							{#if d.total_minutes}· {formatMinutes(i18n, d.total_minutes)}{/if}
						</p>
						<p class="text-xs text-slate-500">
							{i18n.t('leave.updatedAt', { at: formatDateTime(d.updated_at) })}
						</p>
					</div>
					<div class="flex gap-2">
						<a
							href={resolve('/(app)/leave/[id]/edit', { id: String(d.id) })}
							class="rounded-lg bg-brand-700 px-3 py-2 text-sm font-medium text-white hover:bg-brand-800"
							>{i18n.t('leave.resume')}</a
						>
						<form method="POST" action="?/delete" use:enhance={enhanced}>
							<input type="hidden" name="draft_id" value={d.id} />
							<input type="hidden" name="version" value={d.version} />
							<ConfirmButton
								label={i18n.t('leave.deleteDraft')}
								tone="danger"
								title={i18n.t('leave.deleteDraftTitle')}
								message={i18n.t('leave.deleteDraftMsg')}
								confirmLabel={i18n.t('common.delete')}
								class="rounded-lg border border-red-300 px-3 py-2 text-sm font-medium text-red-700 hover:bg-red-50"
							/>
						</form>
					</div>
				</li>
			{/each}
		</ul>
	{/if}
</section>
