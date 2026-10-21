<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import Badge from '#lib/components/Badge.svelte';
	import PageHeader from '#lib/components/PageHeader.svelte';
	import Pagination from '#lib/components/Pagination.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import { formatDateTime } from '#lib/format.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const i18n = useI18n();
	const list = $derived(data.list);
	let q = $derived(data.filters.q);

	// Debounced search; filters live in the URL and every change returns to page 1.
	let timer: ReturnType<typeof setTimeout> | undefined;
	function setParams(changes: Record<string, string | null>) {
		const u = new URL(page.url.href);
		for (const [k, v] of Object.entries(changes)) {
			if (v === null || v === '') u.searchParams.delete(k);
			else u.searchParams.set(k, v);
		}
		u.searchParams.delete('page');
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		goto(u.pathname + u.search, { replace: true, reset: false });
	}
	function onSearch() {
		clearTimeout(timer);
		timer = setTimeout(() => setParams({ q }), 300);
	}

	const chips = $derived([
		{ label: i18n.t('emp.filterAll'), status: 'all', noManager: false },
		{
			label: i18n.t('emp.filterPending', { n: list.summary.inactive }),
			status: 'inactive',
			noManager: false
		},
		{
			label: i18n.t('emp.filterNoManager', { n: list.summary.active_without_manager }),
			status: 'active',
			noManager: true
		}
	]);
	const activeChip = (c: { status: string; noManager: boolean }) =>
		c.status === data.filters.status && c.noManager === data.filters.noManager;
	function chipHref(c: { status: string; noManager: boolean }): string {
		const u = new URL(page.url.href);
		u.searchParams.delete('page');
		if (c.status === 'all') u.searchParams.delete('status');
		else u.searchParams.set('status', c.status);
		if (c.noManager) u.searchParams.set('no_manager', '1');
		else u.searchParams.delete('no_manager');
		return u.pathname + u.search;
	}
	const roleLabel = (r: string) => i18n.t(r === 'ADMIN' ? 'role.ADMIN' : 'role.EMPLOYEE');
	const detail = (id: number) => resolve('/(app)/admin/employees/[id]', { id: String(id) });
</script>

<svelte:head><title>{i18n.t('emp.title')} · G-ASIAPACIFIC</title></svelte:head>

<PageHeader title={i18n.t('emp.title')} description={i18n.t('emp.desc')} />

<form
	method="GET"
	class="mt-6 space-y-3"
	role="search"
	onsubmit={(e) => {
		e.preventDefault();
		setParams({ q });
	}}
>
	<label for="q" class="sr-only">{i18n.t('emp.searchLabel')}</label>
	<div class="flex gap-2">
		<input
			id="q"
			name="q"
			type="search"
			maxlength="100"
			placeholder={i18n.t('emp.searchPlaceholder')}
			class="min-w-0 flex-1 rounded-lg border-slate-300 text-sm"
			bind:value={q}
			oninput={onSearch}
		/>
		<button class="rounded-lg bg-brand-700 px-4 text-sm font-medium text-white hover:bg-brand-800"
			>{i18n.t('common.search')}</button
		>
	</div>
	<div class="flex flex-wrap gap-2" aria-label={i18n.t('emp.filters')}>
		<!-- eslint-disable svelte/no-navigation-without-resolve -->
		{#each chips as c (c.status + c.noManager)}
			<a
				href={chipHref(c)}
				aria-current={activeChip(c) ? 'true' : undefined}
				data-sveltekit-noscroll
				class="rounded-full border px-3 py-1 text-sm {activeChip(c)
					? 'border-brand-600 bg-brand-50 text-brand-800'
					: 'border-slate-300 bg-white text-slate-700 hover:bg-slate-50'}">{c.label}</a
			>
		{/each}
		<!-- eslint-enable svelte/no-navigation-without-resolve -->
	</div>
</form>

<section class="mt-4" aria-live="polite">
	{#if list.items.length === 0}
		<div
			class="rounded-lg border border-dashed border-slate-300 bg-white p-8 text-center text-sm text-slate-600"
		>
			{i18n.t('emp.empty')}
		</div>
	{:else}
		<ul class="space-y-3 md:hidden">
			{#each list.items as e (e.id)}
				<li>
					<a
						href={detail(e.id)}
						class="hover:border-brand-300 block rounded-lg border border-slate-200 bg-white p-4"
					>
						<div class="flex items-start justify-between gap-2">
							<div class="min-w-0">
								<p class="truncate font-medium">{e.full_name}</p>
								<p class="truncate text-sm text-slate-600">{e.email}</p>
							</div>
							<Badge
								tone={e.active ? 'green' : 'amber'}
								label={i18n.t(e.active ? 'common.active' : 'common.inactive')}
							/>
						</div>
						<p class="mt-2 text-sm text-slate-600">
							{i18n.t('emp.managerLabel', { name: e.manager?.full_name ?? '—' })}
							{#if e.role === 'ADMIN'}<span class="ml-2"
									><Badge tone="blue" label={roleLabel('ADMIN')} /></span
								>{/if}
						</p>
					</a>
				</li>
			{/each}
		</ul>

		<div class="hidden overflow-hidden rounded-lg border border-slate-200 bg-white md:block">
			<table class="w-full text-left text-sm">
				<thead class="bg-slate-50 text-xs tracking-wide text-slate-500 uppercase">
					<tr>
						<th class="px-4 py-3 font-medium">{i18n.t('emp.col.employee')}</th>
						<th class="px-4 py-3 font-medium">{i18n.t('emp.col.role')}</th>
						<th class="px-4 py-3 font-medium">{i18n.t('emp.col.status')}</th>
						<th class="px-4 py-3 font-medium">{i18n.t('emp.col.manager')}</th>
						<th class="px-4 py-3 font-medium">{i18n.t('emp.col.lastLogin')}</th>
						<th class="px-4 py-3"><span class="sr-only">{i18n.t('emp.col.actions')}</span></th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each list.items as e (e.id)}
						<tr class="hover:bg-slate-50">
							<td class="px-4 py-3"
								><p class="font-medium">{e.full_name}</p>
								<p class="text-slate-600">{e.email}</p></td
							>
							<td class="px-4 py-3"
								><Badge
									tone={e.role === 'ADMIN' ? 'blue' : 'slate'}
									label={roleLabel(e.role)}
								/></td
							>
							<td class="px-4 py-3"
								><Badge
									tone={e.active ? 'green' : 'amber'}
									label={i18n.t(e.active ? 'common.active' : 'common.inactive')}
								/></td
							>
							<td class="px-4 py-3">
								{#if e.manager}
									{e.manager.full_name}
									{#if e.manager_active === false}<Badge
											tone="red"
											label={i18n.t('common.inactive')}
										/>{/if}
								{:else}
									<span class="text-slate-400">{i18n.t('emp.notAssigned')}</span>
								{/if}
							</td>
							<td class="px-4 py-3 text-slate-600">{formatDateTime(e.last_login_at) || '—'}</td>
							<td class="px-4 py-3 text-right">
								<a class="font-medium text-brand-700 hover:underline" href={detail(e.id)}
									>{i18n.t('emp.manage')}<span class="sr-only"> {e.full_name}</span></a
								>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
	<Pagination page={list.page} totalPages={list.total_pages} totalItems={list.total_items} />
</section>
