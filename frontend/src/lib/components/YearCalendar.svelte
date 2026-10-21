<script lang="ts">
	import type { CalendarDay } from '#lib/admin-types.ts';
	import { useI18n, type MessageKey } from '#lib/i18n/index.ts';
	import { isoWeekday, KINDS, kindLabel } from '#lib/format.ts';

	// 12 month grids for one year. Rest days come from the policy's working days (ISO 1..7).
	let { year, days, workdays }: { year: number; days: CalendarDay[]; workdays: number[] } =
		$props();
	const i18n = useI18n();
	const byDate = $derived(new Map(days.map((d) => [d.date, d])));

	const kindClass: Record<string, string> = {
		NATIONAL_HOLIDAY: 'bg-red-600 text-white',
		COMPANY_HOLIDAY: 'bg-violet-600 text-white',
		COMPENSATORY_HOLIDAY: 'bg-orange-500 text-white',
		MAKEUP_WORKDAY: 'bg-sky-600 text-white'
	};

	function cells(month: number): (string | null)[] {
		const first = new Date(Date.UTC(year, month, 1));
		const lead = (first.getUTCDay() + 6) % 7; // Monday-first
		const count = new Date(Date.UTC(year, month + 1, 0)).getUTCDate();
		const out: (string | null)[] = Array(lead).fill(null);
		for (let d = 1; d <= count; d++) {
			out.push(`${year}-${String(month + 1).padStart(2, '0')}-${String(d).padStart(2, '0')}`);
		}
		return out;
	}
</script>

<div class="flex flex-wrap gap-3 text-xs" aria-label={i18n.t('hol.legend')}>
	{#each KINDS as k (k)}
		<span class="flex items-center gap-1.5"
			><span class="h-3 w-3 rounded {kindClass[k]}"></span>{kindLabel(i18n, k)}</span
		>
	{/each}
	<span class="flex items-center gap-1.5"
		><span class="h-3 w-3 rounded border border-dashed border-slate-500"></span>{i18n.t(
			'hol.draftLegend'
		)}</span
	>
	<span class="flex items-center gap-1.5"
		><span class="h-3 w-3 rounded bg-slate-200"></span>{i18n.t('hol.dayOff')}</span
	>
</div>

<div class="mt-4 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
	{#each Array.from({ length: 12 }, (_, i) => i) as m (m)}
		{@const name = i18n.t(`month.${m + 1}` as MessageKey)}
		<section class="rounded-lg border border-slate-200 bg-white p-3" aria-label="{name} {year}">
			<h3 class="text-sm font-semibold">{name}</h3>
			<div class="mt-2 grid grid-cols-7 gap-1 text-center text-xs" role="grid">
				{#each [1, 2, 3, 4, 5, 6, 7] as w (w)}
					<div class="font-medium text-slate-400" aria-hidden="true">
						{i18n.t(`wd.${w}` as MessageKey)}
					</div>
				{/each}
				{#each cells(m) as iso, i (i)}
					{#if iso === null}
						<div></div>
					{:else}
						{@const d = byDate.get(iso)}
						{@const off = !workdays.includes(isoWeekday(iso))}
						<div
							class="flex aspect-square items-center justify-center rounded text-[11px] {d
								? kindClass[d.kind] +
									(d.status === 'DRAFT'
										? ' opacity-60 outline-2 outline-offset-[-2px] outline-white outline-dashed'
										: '')
								: off
									? 'bg-slate-200 text-slate-500'
									: 'text-slate-700'}"
							title={d ? `${d.name} · ${kindLabel(i18n, d.kind)}` : undefined}
							role="gridcell"
							aria-label={d ? `${iso}: ${d.name}, ${kindLabel(i18n, d.kind)}, ${d.status}` : iso}
						>
							{Number(iso.slice(8))}
						</div>
					{/if}
				{/each}
			</div>
		</section>
	{/each}
</div>
