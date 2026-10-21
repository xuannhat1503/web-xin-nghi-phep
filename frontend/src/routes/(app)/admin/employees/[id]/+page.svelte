<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import Alert from '#lib/components/Alert.svelte';
	import Badge from '#lib/components/Badge.svelte';
	import ConfirmButton from '#lib/components/ConfirmButton.svelte';
	import { useI18n, hasKey } from '#lib/i18n/index.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { withToast, type ActionData } from '#lib/forms.ts';
	import type { PageProps } from './$types';

	let { data, form: rawForm }: PageProps = $props();
	const form = $derived(rawForm as ActionData);
	const i18n = useI18n();
	const enhanced = withToast(i18n);
	const e = $derived(data.employee);
	const isSelf = $derived(e.id === data.me);

	type H = NonNullable<typeof e.history>[number];
	function actionLabel(h: H): string {
		const key = `hist.${h.action}`;
		return hasKey(key) ? i18n.t(key) : h.action;
	}
	function describe(h: H): string {
		const a = h.after ?? {};
		const none = i18n.t('hist.none');
		if (h.kind === 'manager') return `${a.from ?? none} → ${a.to ?? none}`;
		if (h.action === 'ROLE_CHANGED') return `${h.before?.role ?? ''} → ${a.role ?? ''}`;
		return '';
	}
	const fieldErr = (k: string) => (form?.fields?.[k] ? i18n.server(form.fields[k]) : '');
	const btn = 'rounded-lg px-4 py-2 text-sm font-medium';
</script>

<svelte:head><title>{e.full_name} · {i18n.t('emp.title')} · G-ASIAPACIFIC</title></svelte:head>

<a href={resolve('/(app)/admin/employees')} class="text-sm text-brand-700 hover:underline"
	>{i18n.t('empd.back')}</a
>

<div class="mt-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
	<div class="min-w-0">
		<h1 class="truncate text-2xl font-semibold">{e.full_name}</h1>
		<p class="truncate text-sm text-slate-600">{e.email}</p>
	</div>
	<div class="flex gap-2">
		<Badge
			tone={e.role === 'ADMIN' ? 'blue' : 'slate'}
			label={i18n.t(e.role === 'ADMIN' ? 'role.ADMIN' : 'role.EMPLOYEE')}
		/>
		<Badge
			tone={e.active ? 'green' : 'amber'}
			label={i18n.t(e.active ? 'common.active' : 'common.inactive')}
		/>
	</div>
</div>

{#if form?.error || form?.errorKey}
	<div class="mt-4">
		<Alert tone="error">{form.error ? i18n.server(form.error) : i18n.msg(form.errorKey)}</Alert>
	</div>
{:else if form?.success}
	<div class="mt-4"><Alert tone="success">{i18n.msg(form.success)}</Alert></div>
{/if}

<div class="mt-6 grid gap-6 lg:grid-cols-2">
	<section class="rounded-xl border border-slate-200 bg-white p-5" aria-labelledby="access-h">
		<h2 id="access-h" class="font-semibold">{i18n.t('empd.access')}</h2>
		<dl class="mt-3 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
			<dt class="text-slate-500">{i18n.t('empd.lastSignIn')}</dt>
			<dd>{formatDateTime(e.last_login_at) || i18n.t('common.never')}</dd>
			<dt class="text-slate-500">{i18n.t('empd.registered')}</dt>
			<dd>{formatDateTime(e.created_at)}</dd>
			<dt class="text-slate-500">{i18n.t('empd.directReports')}</dt>
			<dd>{i18n.t('empd.reports', { n: e.active_reports })}</dd>
		</dl>

		{#if isSelf}
			<p class="mt-4 text-sm text-slate-600">{i18n.t('empd.selfNote')}</p>
		{:else}
			<form
				method="POST"
				action="?/access"
				use:enhance={enhanced}
				class="mt-4 flex flex-wrap gap-2"
			>
				<input type="hidden" name="version" value={e.version} />
				{#if e.active}
					<ConfirmButton
						name="op"
						value="deactivate"
						label={i18n.t('empd.deactivate')}
						tone="danger"
						title={i18n.t('empd.deactivateTitle', { name: e.full_name })}
						message={i18n.t('empd.deactivateMsg')}
						confirmLabel={i18n.t('empd.deactivate')}
						class="{btn} border border-red-300 text-red-700 hover:bg-red-50"
					/>
				{:else}
					<button
						name="op"
						value="activate"
						class="{btn} bg-emerald-700 text-white hover:bg-emerald-800"
						>{i18n.t('empd.activate')}</button
					>
				{/if}
				{#if e.role === 'ADMIN'}
					<ConfirmButton
						name="op"
						value="make_employee"
						label={i18n.t('empd.removeAdmin')}
						title={i18n.t('empd.removeAdminTitle')}
						message={i18n.t('empd.removeAdminMsg', { name: e.full_name })}
						class="{btn} border border-slate-300 text-slate-700 hover:bg-slate-50"
					/>
				{:else}
					<ConfirmButton
						name="op"
						value="make_admin"
						label={i18n.t('empd.makeAdmin')}
						title={i18n.t('empd.makeAdminTitle')}
						message={i18n.t('empd.makeAdminMsg')}
						class="{btn} border border-slate-300 text-slate-700 hover:bg-slate-50"
					/>
				{/if}
			</form>
		{/if}
	</section>

	<section class="rounded-xl border border-slate-200 bg-white p-5" aria-labelledby="mgr-h">
		<h2 id="mgr-h" class="font-semibold">{i18n.t('empd.manager')}</h2>
		<p class="mt-1 text-sm text-slate-600">{i18n.t('empd.managerHelp')}</p>
		<p class="mt-3 text-sm">
			{i18n.t('empd.current')}
			{#if e.manager}
				<span class="font-medium">{e.manager.full_name}</span>
				{#if e.manager_active === false}<Badge tone="red" label={i18n.t('common.inactive')} />{/if}
			{:else}
				<span class="text-slate-500">{i18n.t('emp.notAssigned')}</span>
			{/if}
		</p>

		<form method="GET" class="mt-4 flex gap-2" role="search">
			<label for="mq" class="sr-only">{i18n.t('empd.searchCandidates')}</label>
			<input
				id="mq"
				name="mq"
				type="search"
				value={data.mq}
				maxlength="100"
				placeholder={i18n.t('empd.findPlaceholder')}
				class="min-w-0 flex-1 rounded-lg border-slate-300 text-sm"
			/>
			<button class="{btn} border border-slate-300 text-slate-700 hover:bg-slate-50"
				>{i18n.t('common.find')}</button
			>
		</form>

		<form method="POST" action="?/manager" use:enhance={enhanced} class="mt-3 space-y-3">
			<input type="hidden" name="version" value={e.version} />
			<div>
				<label for="manager_id" class="block text-sm font-medium"
					>{i18n.t('empd.managerSelect')}</label
				>
				<select
					id="manager_id"
					name="manager_id"
					class="mt-1 w-full rounded-lg border-slate-300 text-sm"
				>
					<option value="none">{i18n.t('empd.noManagerOption')}</option>
					{#each data.candidates as c (c.id)}
						<option value={c.id} selected={c.id === e.manager?.id}>{c.full_name} ({c.email})</option
						>
					{/each}
				</select>
				{#if data.candidates.length === 20}<p class="mt-1 text-xs text-slate-500">
						{i18n.t('empd.first20')}
					</p>{/if}
				{#if fieldErr('manager_id')}<p class="mt-1 text-sm text-red-700">
						{fieldErr('manager_id')}
					</p>{/if}
			</div>
			<div>
				<label for="reason" class="block text-sm font-medium"
					>{i18n.t('empd.reason')}
					<span class="font-normal text-slate-500">{i18n.t('common.optional')}</span></label
				>
				<input
					id="reason"
					name="reason"
					maxlength="500"
					class="mt-1 w-full rounded-lg border-slate-300 text-sm"
				/>
				{#if fieldErr('reason')}<p class="mt-1 text-sm text-red-700">{fieldErr('reason')}</p>{/if}
			</div>
			<button class="{btn} bg-brand-700 text-white hover:bg-brand-800"
				>{i18n.t('empd.saveManager')}</button
			>
		</form>
	</section>
</div>

<section class="mt-6 rounded-xl border border-slate-200 bg-white p-5" aria-labelledby="hist-h">
	<h2 id="hist-h" class="font-semibold">{i18n.t('empd.history')}</h2>
	{#if !e.history?.length}
		<p class="mt-2 text-sm text-slate-600">{i18n.t('empd.noHistory')}</p>
	{:else}
		<ol class="mt-3 space-y-3 border-l border-slate-200 pl-4">
			{#each e.history as h, i (i)}
				<li class="text-sm">
					<p>
						<span class="font-medium">{actionLabel(h)}</span>
						{#if describe(h)}<span class="text-slate-600"> · {describe(h)}</span>{/if}
					</p>
					<p class="text-xs text-slate-500">
						{formatDateTime(h.created_at)} · {i18n.t('common.by', {
							name: h.actor === 'System' ? i18n.t('common.system') : h.actor
						})}
					</p>
					{#if h.reason}<p class="mt-0.5 text-slate-600">“{h.reason}”</p>{/if}
				</li>
			{/each}
		</ol>
	{/if}
</section>
