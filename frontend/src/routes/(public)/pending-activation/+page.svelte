<script lang="ts">
	import { resolve } from '$app/paths';
	import Alert from '#lib/components/Alert.svelte';
	import SignOutButton from '#lib/components/SignOutButton.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const i18n = useI18n();
</script>

<svelte:head><title>{i18n.t('pending.title')} · G-ASIAPACIFIC</title></svelte:head>

<h1 class="text-2xl font-semibold text-slate-900">{i18n.t('pending.title')}</h1>

<div class="mt-4">
	<Alert tone="warning" title={i18n.t('pending.almost')}>
		<span class="break-words">{i18n.t('pending.body', { email: data.user?.email ?? '' })}</span>
	</Alert>
</div>

<p class="mt-4 text-sm text-slate-600">{i18n.t('pending.contact')}</p>

<div class="mt-6 flex flex-col gap-3 sm:flex-row">
	<a
		href={resolve('/(public)/pending-activation')}
		data-sveltekit-reload
		class="inline-flex flex-1 items-center justify-center rounded-lg bg-brand-700 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-800"
	>
		{i18n.t('pending.checkAgain')}
	</a>
	<SignOutButton
		csrfToken={data.csrfToken}
		class="w-full rounded-lg border border-slate-300 px-4 py-2.5 text-sm font-medium text-slate-700 hover:bg-slate-50 sm:w-auto"
	/>
</div>
