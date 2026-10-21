<script lang="ts">
	import { resolve } from '$app/paths';
	import { useI18n } from '#lib/i18n/index.ts';
	import { formatDateTime } from '#lib/format.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const i18n = useI18n();
	const m = $derived(data.mail);
</script>

<svelte:head><title>{m.email.subject} · {i18n.t('mock.title')}</title></svelte:head>

<a href={resolve('/(app)/admin/mock-email')} class="text-sm text-brand-700 hover:underline"
	>{i18n.t('mock.back')}</a
>
<section class="mt-4 rounded-xl border border-slate-200 bg-white p-5">
	<dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
		<dt class="text-slate-500">{i18n.t('mock.type')}</dt>
		<dd>{m.email.email_type}</dd>
		<dt class="text-slate-500">{i18n.t('mock.to')}</dt>
		<dd class="break-all">{m.email.to}</dd>
		<dt class="text-slate-500">{i18n.t('mock.cc')}</dt>
		<dd class="break-all">{m.cc.length ? m.cc.join(', ') : '—'}</dd>
		<dt class="text-slate-500">{i18n.t('mock.subject')}</dt>
		<dd class="font-medium break-words">{m.email.subject}</dd>
		<dt class="text-slate-500">{i18n.t('mock.at')}</dt>
		<dd>{formatDateTime(m.email.created_at)}</dd>
		<dt class="text-slate-500">{i18n.t('mock.transport')}</dt>
		<dd>{m.transport}</dd>
	</dl>
	<pre
		class="mt-4 overflow-auto rounded-lg bg-slate-50 p-4 text-xs leading-relaxed whitespace-pre-wrap text-slate-800">{m.body}</pre>
</section>
