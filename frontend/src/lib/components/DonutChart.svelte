<script lang="ts">
	// Accessible SVG donut chart with a legend.
	type Slice = { label: string; value: number; color: string };
	let { slices, title, empty }: { slices: Slice[]; title: string; empty: string } = $props();

	const total = $derived(slices.reduce((a, s) => a + s.value, 0));
	const R = 60;
	const C = 2 * Math.PI * R;
	const arcs = $derived.by(() => {
		let offset = 0;
		return slices
			.filter((s) => s.value > 0)
			.map((s) => {
				const len = (s.value / total) * C;
				const a = { ...s, len, offset };
				offset += len;
				return a;
			});
	});
</script>

<figure class="flex flex-col items-center gap-4 sm:flex-row">
	{#if total === 0}
		<p class="text-sm text-slate-500">{empty}</p>
	{:else}
		<svg viewBox="0 0 160 160" class="h-40 w-40 shrink-0" role="img" aria-label={title}>
			<g transform="rotate(-90 80 80)">
				{#each arcs as a (a.label)}
					<circle
						cx="80"
						cy="80"
						r={R}
						fill="none"
						stroke={a.color}
						stroke-width="22"
						stroke-dasharray="{a.len} {C - a.len}"
						stroke-dashoffset={-a.offset}
					>
						<title>{a.label}: {a.value}</title>
					</circle>
				{/each}
			</g>
			<text x="80" y="86" text-anchor="middle" font-size="22" font-weight="600" fill="#0f172a"
				>{total}</text
			>
		</svg>
		<ul class="space-y-1 text-sm">
			{#each slices as s (s.label)}
				<li class="flex items-center gap-2">
					<span class="h-3 w-3 rounded" style="background:{s.color}"></span>
					<span>{s.label}</span>
					<span class="ml-auto pl-4 font-medium tabular-nums"
						>{s.value}{total ? ` (${Math.round((s.value / total) * 100)}%)` : ''}</span
					>
				</li>
			{/each}
		</ul>
	{/if}
</figure>
