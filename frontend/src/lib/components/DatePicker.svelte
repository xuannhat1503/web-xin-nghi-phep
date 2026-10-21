<script lang="ts">
	import { useI18n, type MessageKey } from '#lib/i18n/index.ts';
	import { formatDate, isoWeekday, kindLabel, parseDisplayDate } from '#lib/format.ts';
	import type { ConfirmedDay } from '#lib/leave.ts';

	// A dd/MM/yyyy text field (works without JavaScript) with a month popup that marks
	// confirmed holidays, company days off, makeup workdays and regular rest days.
	let {
		id,
		name,
		label,
		value = $bindable(''),
		days,
		workdays,
		invalid = false,
		describedby
	}: {
		id: string;
		name: string;
		label: string;
		value?: string; // dd/MM/yyyy as typed
		days: ConfirmedDay[];
		workdays: number[];
		invalid?: boolean;
		describedby?: string;
	} = $props();

	const i18n = useI18n();
	let open = $state(false);
	const byDate = $derived(new Map(days.map((d) => [d.date, d])));

	function today(): string {
		return new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Ho_Chi_Minh' }).format(new Date());
	}
	let view = $state({ y: 2026, m: 0 });
	function openPicker() {
		const iso = parseDisplayDate(value) ?? today();
		view = { y: Number(iso.slice(0, 4)), m: Number(iso.slice(5, 7)) - 1 };
		open = true;
	}
	function shift(n: number) {
		const d = new Date(Date.UTC(view.y, view.m + n, 1));
		view = { y: d.getUTCFullYear(), m: d.getUTCMonth() };
	}
	function pick(iso: string) {
		value = formatDate(iso);
		open = false;
	}
	const cells = $derived.by(() => {
		const first = new Date(Date.UTC(view.y, view.m, 1));
		const lead = (first.getUTCDay() + 6) % 7;
		const count = new Date(Date.UTC(view.y, view.m + 1, 0)).getUTCDate();
		const out: (string | null)[] = Array(lead).fill(null);
		for (let d = 1; d <= count; d++) {
			out.push(`${view.y}-${String(view.m + 1).padStart(2, '0')}-${String(d).padStart(2, '0')}`);
		}
		return out;
	});
	const selected = $derived(parseDisplayDate(value));

	function cellClass(iso: string): string {
		const d = byDate.get(iso);
		const rest = !workdays.includes(isoWeekday(iso));
		if (d?.kind === 'MAKEUP_WORKDAY') return 'bg-sky-100 text-sky-900 font-semibold';
		if (d) return 'bg-red-100 text-red-800 font-semibold';
		if (rest) return 'bg-slate-100 text-slate-400';
		return 'text-slate-800 hover:bg-brand-50';
	}
	function cellTitle(iso: string): string {
		const d = byDate.get(iso);
		if (d)
			return `${d.name} · ${kindLabel(i18n, d.kind)}${d.start_time ? ` (${d.start_time}–${d.end_time})` : ''}`;
		if (!workdays.includes(isoWeekday(iso))) return i18n.t('leave.legend.rest');
		return formatDate(iso);
	}
	function onKey(e: KeyboardEvent) {
		if (e.key === 'Escape') open = false;
	}
</script>

<div class="relative">
	<label for={id} class="block text-sm font-medium">{label}</label>
	<div class="mt-1 flex">
		<input
			{id}
			{name}
			type="text"
			inputmode="numeric"
			autocomplete="off"
			placeholder={i18n.t('leave.dateHint')}
			maxlength="10"
			bind:value
			aria-invalid={invalid}
			aria-describedby={describedby}
			class="min-w-0 flex-1 rounded-l-lg border-slate-300 text-sm tabular-nums"
		/>
		<button
			type="button"
			class="rounded-r-lg border border-l-0 border-slate-300 bg-white px-3 text-slate-600 hover:bg-slate-50"
			aria-label={i18n.t('leave.pickDate')}
			aria-expanded={open}
			aria-controls="{id}-picker"
			onclick={() => (open ? (open = false) : openPicker())}
		>
			<svg
				aria-hidden="true"
				viewBox="0 0 24 24"
				class="h-5 w-5"
				fill="none"
				stroke="currentColor"
				stroke-width="1.8"
				><rect x="3" y="5" width="18" height="16" rx="2" /><path d="M3 10h18M8 3v4M16 3v4" /></svg
			>
		</button>
	</div>

	{#if open}
		<div
			id="{id}-picker"
			role="dialog"
			aria-label={label}
			tabindex="-1"
			class="absolute z-40 mt-2 w-72 max-w-[calc(100vw-2rem)] rounded-xl border border-slate-200 bg-white p-3 shadow-xl"
			onkeydown={onKey}
		>
			<div class="flex items-center justify-between">
				<button
					type="button"
					class="rounded p-1 hover:bg-slate-100"
					aria-label={i18n.t('common.previous')}
					onclick={() => shift(-1)}>‹</button
				>
				<p class="text-sm font-semibold">{i18n.t(`month.${view.m + 1}` as MessageKey)} {view.y}</p>
				<button
					type="button"
					class="rounded p-1 hover:bg-slate-100"
					aria-label={i18n.t('common.next')}
					onclick={() => shift(1)}>›</button
				>
			</div>
			<div class="mt-2 grid grid-cols-7 gap-1 text-center text-xs">
				{#each [1, 2, 3, 4, 5, 6, 7] as w (w)}
					<div class="font-medium text-slate-400" aria-hidden="true">
						{i18n.t(`wd.${w}` as MessageKey)}
					</div>
				{/each}
				{#each cells as iso, i (i)}
					{#if iso === null}
						<div></div>
					{:else}
						<button
							type="button"
							class="aspect-square rounded text-xs {cellClass(iso)} {selected === iso
								? 'ring-2 ring-brand-600'
								: ''}"
							title={cellTitle(iso)}
							aria-label="{formatDate(iso)}: {cellTitle(iso)}"
							aria-pressed={selected === iso}
							onclick={() => pick(iso)}>{Number(iso.slice(8))}</button
						>
					{/if}
				{/each}
			</div>
			<div class="mt-3 flex flex-wrap gap-x-3 gap-y-1 text-[11px] text-slate-600">
				<span class="flex items-center gap-1"
					><span class="h-2.5 w-2.5 rounded bg-red-100"></span>{i18n.t(
						'leave.legend.holiday'
					)}</span
				>
				<span class="flex items-center gap-1"
					><span class="h-2.5 w-2.5 rounded bg-sky-100"></span>{i18n.t('leave.legend.makeup')}</span
				>
				<span class="flex items-center gap-1"
					><span class="h-2.5 w-2.5 rounded bg-slate-100"></span>{i18n.t('leave.legend.rest')}</span
				>
			</div>
			<button
				type="button"
				class="mt-2 w-full rounded-md py-1 text-xs text-slate-500 hover:bg-slate-50"
				onclick={() => (open = false)}>{i18n.t('leave.closePicker')}</button
			>
		</div>
	{/if}
</div>
