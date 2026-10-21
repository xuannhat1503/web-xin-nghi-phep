<script lang="ts">
	type Tone = 'green' | 'amber' | 'red' | 'blue' | 'slate' | 'violet';
	let { tone = 'slate', label }: { tone?: Tone; label: string } = $props();

	const tones: Record<Tone, string> = {
		green: 'bg-emerald-50 text-emerald-800 ring-emerald-600/20',
		amber: 'bg-amber-50 text-amber-800 ring-amber-600/20',
		red: 'bg-red-50 text-red-800 ring-red-600/20',
		blue: 'bg-brand-50 text-brand-800 ring-brand-600/20',
		slate: 'bg-slate-100 text-slate-700 ring-slate-500/20',
		violet: 'bg-violet-50 text-violet-800 ring-violet-600/20'
	};
</script>

<span
	class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap ring-1 ring-inset {tones[
		tone
	]}">{label}</span
>
