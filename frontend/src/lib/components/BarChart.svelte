<script lang="ts">
	// Lightweight accessible SVG bar chart (single or stacked series). No chart library needed.
	type Series = { label: string; color: string; values: number[] };
	let {
		labels,
		series,
		title,
		format = (n: number) => String(n)
	}: {
		labels: string[];
		series: Series[];
		title: string;
		format?: (n: number) => string;
	} = $props();

	const totals = $derived(labels.map((_, i) => series.reduce((a, s) => a + (s.values[i] ?? 0), 0)));
	const max = $derived(Math.max(1, ...totals));
	const W = 600;
	const H = 220;
	const pad = { l: 36, r: 8, t: 10, b: 24 };
	const bw = $derived((W - pad.l - pad.r) / labels.length);
	const y = (v: number) => pad.t + (H - pad.t - pad.b) * (1 - v / max);
	const ticks = $derived([0, max / 2, max]);
</script>

<figure>
	<svg viewBox="0 0 {W} {H}" class="h-auto w-full" role="img" aria-label={title}>
		{#each ticks as t, i (i)}
			<line x1={pad.l} x2={W - pad.r} y1={y(t)} y2={y(t)} stroke="#e2e8f0" />
			<text x={pad.l - 4} y={y(t) + 4} text-anchor="end" font-size="10" fill="#64748b"
				>{format(Math.round(t))}</text
			>
		{/each}
		{#each labels as l, i (i)}
			{@const x = pad.l + i * bw + bw * 0.15}
			{#each series as s, si (si)}
				{@const below = series.slice(0, si).reduce((a, p) => a + (p.values[i] ?? 0), 0)}
				{@const v = s.values[i] ?? 0}
				{#if v > 0}
					<rect
						{x}
						y={y(below + v)}
						width={bw * 0.7}
						height={y(below) - y(below + v)}
						fill={s.color}
						rx="2"
					>
						<title>{l} · {s.label}: {format(v)}</title>
					</rect>
				{/if}
			{/each}
			<text x={x + bw * 0.35} y={H - 8} text-anchor="middle" font-size="10" fill="#64748b">{l}</text
			>
		{/each}
	</svg>
	{#if series.length > 1}
		<figcaption class="mt-2 flex flex-wrap gap-3 text-xs text-slate-600">
			{#each series as s (s.label)}<span class="flex items-center gap-1.5"
					><span class="h-2.5 w-2.5 rounded" style="background:{s.color}"></span>{s.label}</span
				>{/each}
		</figcaption>
	{/if}
	<!-- Data table for screen readers -->
	<table class="sr-only">
		<caption>{title}</caption>
		<tbody>
			{#each labels as l, i (i)}<tr
					><th>{l}</th>{#each series as s (s.label)}<td>{s.label}: {format(s.values[i] ?? 0)}</td
						>{/each}</tr
				>{/each}
		</tbody>
	</table>
</figure>
