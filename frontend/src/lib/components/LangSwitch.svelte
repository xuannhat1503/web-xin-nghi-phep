<script lang="ts">
	import { page } from '$app/state';
	import { LANGS, useI18n } from '#lib/i18n/index.ts';

	let { dark = false }: { dark?: boolean } = $props();
	const i18n = useI18n();
	const back = $derived(page.url.pathname + page.url.search);
</script>

<form
	method="POST"
	action="/lang"
	data-sveltekit-reload
	class="flex items-center"
	aria-label={i18n.t('lang.label')}
>
	<input type="hidden" name="redirect_to" value={back} />
	<div class="flex rounded-md p-0.5 text-xs font-semibold {dark ? 'bg-white/10' : 'bg-slate-100'}">
		{#each LANGS as l (l)}
			<button
				name="lang"
				value={l}
				aria-pressed={i18n.lang === l}
				title={i18n.t(l === 'vi' ? 'lang.vi' : 'lang.en')}
				class="rounded px-2 py-1 uppercase {i18n.lang === l
					? dark
						? 'bg-white text-brand-900'
						: 'bg-white text-brand-800 shadow-sm'
					: dark
						? 'text-white/80 hover:text-white'
						: 'text-slate-500 hover:text-slate-800'}">{l}</button
			>
		{/each}
	</div>
</form>
