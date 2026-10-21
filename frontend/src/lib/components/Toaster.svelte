<script lang="ts">
	import { dismiss, toasts } from '#lib/toast.svelte.ts';
	import { useI18n } from '#lib/i18n/index.ts';
	const i18n = useI18n();

	const tones = {
		success: 'bg-emerald-700',
		error: 'bg-red-700',
		info: 'bg-brand-800'
	};
</script>

<div
	class="pointer-events-none fixed inset-x-0 bottom-4 z-50 flex flex-col items-center gap-2 px-4 sm:items-end sm:px-6"
	aria-live="polite"
>
	{#each toasts as t (t.id)}
		<div
			class="pointer-events-auto flex w-full max-w-sm items-start gap-3 rounded-lg px-4 py-3 text-sm text-white shadow-lg {tones[
				t.tone
			]}"
			role={t.tone === 'error' ? 'alert' : 'status'}
		>
			<p class="flex-1">{t.message}</p>
			<button
				type="button"
				class="-m-1 rounded p-1 opacity-80 hover:opacity-100"
				aria-label={i18n.t('common.dismiss')}
				onclick={() => dismiss(t.id)}>✕</button
			>
		</div>
	{/each}
</div>
