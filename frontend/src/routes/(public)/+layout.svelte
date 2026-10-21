<script lang="ts">
	import LangSwitch from '#lib/components/LangSwitch.svelte';
	import Logo from '#lib/components/Logo.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
	const i18n = useI18n();
</script>

<div
	class="relative flex min-h-dvh flex-col items-center justify-center bg-gradient-to-br from-brand-950 via-brand-900 to-brand-700 px-4 py-10"
>
	<div class="absolute top-4 right-4"><LangSwitch dark /></div>
	<div class="mb-8"><Logo class="h-12 w-auto" /></div>
	<main class="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl sm:p-8">
		{@render children()}
	</main>
	<p class="mt-6 text-center text-xs text-brand-100/80">{i18n.t('app.internal')}</p>
</div>
