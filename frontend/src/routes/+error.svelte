<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { useI18n } from '#lib/i18n/index.ts';

	const i18n = useI18n();
	const status = $derived(page.status);
	const title = $derived(
		status === 403
			? i18n.t('error.403')
			: status === 404
				? i18n.t('error.404')
				: status === 503
					? i18n.t('error.503')
					: i18n.t('error.other')
	);
	const message = $derived(
		status === 404
			? i18n.t('error.404msg')
			: status === 403
				? i18n.t('error.403msg')
				: i18n.server(page.error?.message) || i18n.t('common.genericError')
	);
</script>

<svelte:head><title>{title} · G-ASIAPACIFIC</title></svelte:head>

<div class="flex min-h-dvh items-center justify-center bg-slate-50 px-4">
	<main class="w-full max-w-md text-center">
		<p class="text-sm font-semibold text-brand-600">{status}</p>
		<h1 class="mt-2 text-2xl font-semibold text-slate-900">{title}</h1>
		<p class="mt-2 text-sm text-slate-600">{message}</p>
		<a
			href={resolve('/(app)/dashboard')}
			data-sveltekit-reload
			class="mt-6 inline-flex rounded-lg bg-brand-700 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-800"
		>
			{i18n.t('common.goHome')}
		</a>
	</main>
</div>
