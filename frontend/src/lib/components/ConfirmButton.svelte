<script lang="ts">
	import type { Snippet } from 'svelte';
	import { useI18n } from '#lib/i18n/index.ts';

	// A submit button that asks for confirmation in a modal <dialog>. Without JavaScript the form
	// submits directly (the server still validates everything).
	let {
		label,
		title,
		message,
		confirmLabel,
		tone = 'primary',
		name,
		value,
		class: className = '',
		children
	}: {
		label: string;
		title: string;
		message?: string;
		confirmLabel?: string;
		tone?: 'primary' | 'danger';
		name?: string;
		value?: string;
		class?: string;
		children?: Snippet;
	} = $props();

	const i18n = useI18n();
	let dialog: HTMLDialogElement | undefined = $state();
	let trigger: HTMLButtonElement | undefined = $state();

	function open(e: MouseEvent) {
		e.preventDefault();
		dialog?.showModal();
	}

	function confirm() {
		dialog?.close();
		const form = trigger?.form;
		if (form && trigger) form.requestSubmit(trigger);
	}

	const confirmClass = $derived(
		tone === 'danger' ? 'bg-red-700 hover:bg-red-800' : 'bg-brand-700 hover:bg-brand-800'
	);
</script>

<button bind:this={trigger} type="submit" {name} {value} class={className} onclick={open}>
	{label}
</button>

<dialog
	bind:this={dialog}
	class="m-auto w-[min(28rem,calc(100vw-2rem))] rounded-xl p-0 shadow-2xl backdrop:bg-slate-900/50"
	aria-labelledby="dlg-title"
>
	<div class="p-6">
		<h2 id="dlg-title" class="text-lg font-semibold text-slate-900">{title}</h2>
		{#if message}<p class="mt-2 text-sm text-slate-600">{message}</p>{/if}
		{#if children}<div class="mt-3 text-sm text-slate-700">{@render children()}</div>{/if}
		<div class="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
			<button
				type="button"
				class="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
				onclick={() => dialog?.close()}>{i18n.t('common.cancel')}</button
			>
			<button
				type="button"
				class="rounded-lg px-4 py-2 text-sm font-medium text-white {confirmClass}"
				onclick={confirm}>{confirmLabel ?? i18n.t('common.confirm')}</button
			>
		</div>
	</div>
</dialog>
