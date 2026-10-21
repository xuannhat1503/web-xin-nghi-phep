<script lang="ts">
	import Badge from '#lib/components/Badge.svelte';
	import PageHeader from '#lib/components/PageHeader.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import { formatDateTime } from '#lib/format.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const i18n = useI18n();
	const s = $derived(data.status);
	const card = 'rounded-xl border border-slate-200 bg-white p-5';
</script>

<svelte:head><title>{i18n.t('sys.title')} · G-ASIAPACIFIC</title></svelte:head>

<PageHeader title={i18n.t('sys.title')} description={i18n.t('sys.desc')} />

<div class="mt-6 grid gap-4 sm:grid-cols-3">
	<div class={card}>
		<p class="text-sm text-slate-600">{i18n.t('sys.db')}</p>
		<p class="mt-2">
			<Badge
				tone={s.database === 'ok' ? 'green' : 'red'}
				label={i18n.t(s.database === 'ok' ? 'sys.ok' : 'sys.fail')}
			/>
		</p>
	</div>
	<div class={card}>
		<p class="text-sm text-slate-600">{i18n.t('sys.migration')}</p>
		{#if s.migration}
			<p class="mt-1 text-lg font-semibold">{s.migration.version} / {s.migration.latest}</p>
			<Badge
				tone={s.migration.up_to_date ? 'green' : 'amber'}
				label={i18n.t(s.migration.up_to_date ? 'sys.upToDate' : 'sys.outdated')}
			/>
		{/if}
	</div>
	<div class={card}>
		<p class="text-sm text-slate-600">{i18n.t('sys.emailMode')}</p>
		<p class="mt-1 text-lg font-semibold">{s.email_mode ?? 'MOCK'}</p>
	</div>
</div>

<section class="{card} mt-6">
	<h2 class="font-semibold">{i18n.t('sys.workers')}</h2>
	<ul class="mt-3 divide-y divide-slate-100 text-sm">
		{#each s.workers ?? [] as w (w.name)}
			<li class="flex flex-col gap-1 py-2 sm:flex-row sm:items-center sm:justify-between">
				<span class="font-medium">{w.name}</span>
				<span class="text-slate-600">
					{i18n.t('sys.lastRun')}: {formatDateTime(w.last_run_at) || '—'} · {i18n.t('sys.lastOk')}: {formatDateTime(
						w.last_success_at
					) || '—'}
					{#if w.last_error}<Badge tone="red" label={w.last_error} />{/if}
				</span>
			</li>
		{/each}
	</ul>
</section>

<section class="{card} mt-6">
	<h2 class="font-semibold">{i18n.t('sys.outbox')}</h2>
	<div class="mt-3 flex flex-wrap gap-3 text-sm">
		{#each Object.entries(s.outbox ?? {}) as [k, n] (k)}
			<span class="rounded-lg bg-slate-50 px-3 py-2"
				>{k}: <strong class="tabular-nums">{n}</strong></span
			>
		{/each}
	</div>
</section>

<section class="{card} mt-6">
	<h2 class="font-semibold">{i18n.t('sys.errors')}</h2>
	{#if !s.recent_errors?.length}
		<p class="mt-2 text-sm text-slate-600">{i18n.t('sys.noErrors')}</p>
	{:else}
		<ul class="mt-3 space-y-1 text-sm">
			{#each s.recent_errors as e, i (i)}
				<li>
					<span class="text-slate-500 tabular-nums">{formatDateTime(e.created_at)}</span> ·
					<strong>{e.component}</strong>
					· {e.message}
				</li>
			{/each}
		</ul>
	{/if}
	<p class="mt-4 text-xs text-slate-500">/health/live · /health/ready</p>
</section>
