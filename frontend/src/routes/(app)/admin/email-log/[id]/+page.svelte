<script lang="ts">
	import { resolve } from '$app/paths';
	import Badge from '#lib/components/Badge.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import { deliveryLabel, deliveryTone, emailTypeLabel, formatDateTime } from '#lib/format.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const i18n = useI18n();
	const m = $derived(data.mail);
</script>

<svelte:head><title>{m.email.subject} · {i18n.t('mail.title')}</title></svelte:head>

<a href={resolve('/(app)/admin/email-log')} class="text-sm text-brand-700 hover:underline"
	>← {i18n.t('mail.title')}</a
>
<section class="mt-4 rounded-xl border border-slate-200 bg-white p-5">
	<dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
		<dt class="text-slate-500">{i18n.t('mail.type')}</dt>
		<dd>{emailTypeLabel(i18n, m.email.email_type)}</dd>
		<dt class="text-slate-500">{i18n.t('mail.to')}</dt>
		<dd class="break-all">{m.email.to}</dd>
		<dt class="text-slate-500">{i18n.t('mail.cc')}</dt>
		<dd class="break-all">{m.cc.length ? m.cc.join(', ') : '—'}</dd>
		<dt class="text-slate-500">{i18n.t('mail.subject')}</dt>
		<dd class="font-medium break-words">{m.email.subject}</dd>
		<dt class="text-slate-500">{i18n.t('mail.createdAt')}</dt>
		<dd>{formatDateTime(m.email.created_at)}</dd>
		<dt class="text-slate-500">{i18n.t('mail.delivery')}</dt>
		<dd>
			<Badge tone={deliveryTone(m.email.delivery)} label={deliveryLabel(i18n, m.email.delivery)} />
			{#if m.email.delivered_at}
				<span class="ml-1 text-slate-600">{formatDateTime(m.email.delivered_at)}</span>
			{/if}
		</dd>
		{#if m.delivered_to}
			<dt class="text-slate-500">{i18n.t('mail.deliveredTo')}</dt>
			<dd class="break-all">{m.delivered_to}</dd>
		{/if}
		{#if m.error}
			<dt class="text-slate-500">{i18n.t('mail.error')}</dt>
			<dd class="text-red-700">{i18n.server(m.error)}</dd>
		{/if}
	</dl>
	<pre
		class="mt-4 overflow-auto rounded-lg bg-slate-50 p-4 text-xs leading-relaxed whitespace-pre-wrap text-slate-800">{m.body}</pre>
</section>
