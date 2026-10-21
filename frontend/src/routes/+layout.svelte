<script lang="ts">
	import './layout.css';
	import { provideI18n } from '#lib/i18n/index.ts';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();
	const i18n = provideI18n(
		() => data.lang,
		() => data.catalog
	);
</script>

<svelte:head>
	<link rel="icon" href="/favicon.svg" type="image/svg+xml" />
	<title>G-ASIAPACIFIC · {i18n.t('app.name')}</title>
</svelte:head>

{@render children()}
