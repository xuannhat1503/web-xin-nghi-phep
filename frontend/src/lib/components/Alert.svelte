<script lang="ts">
	import type { Snippet } from 'svelte';

	type Tone = 'info' | 'success' | 'warning' | 'error';
	let {
		tone = 'info',
		title,
		children
	}: { tone?: Tone; title?: string; children: Snippet } = $props();

	const styles: Record<Tone, string> = {
		info: 'border-brand-100 bg-brand-50 text-brand-900',
		success: 'border-emerald-200 bg-emerald-50 text-emerald-900',
		warning: 'border-amber-200 bg-amber-50 text-amber-900',
		error: 'border-red-200 bg-red-50 text-red-900'
	};
</script>

<div
	class="rounded-lg border px-4 py-3 text-sm {styles[tone]}"
	role={tone === 'error' || tone === 'warning' ? 'alert' : 'status'}
>
	{#if title}<p class="font-semibold">{title}</p>{/if}
	<div class={title ? 'mt-1' : ''}>{@render children()}</div>
</div>
