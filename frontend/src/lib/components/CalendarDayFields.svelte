<script lang="ts">
	import type { CalendarDay } from '#lib/admin-types.ts';
	import { useI18n } from '#lib/i18n/index.ts';
	import { KINDS, kindLabel } from '#lib/format.ts';

	// Shared create/edit fields for a calendar day. Errors come from the API's field map.
	let {
		day,
		defaultDate = '',
		fields = {}
	}: { day?: CalendarDay; defaultDate?: string; fields?: Record<string, string> } = $props();

	const i18n = useI18n();
	let kind = $derived<string>(day?.kind ?? 'NATIONAL_HOLIDAY');
	const input = 'mt-1 w-full rounded-lg border-slate-300 text-sm';
	const err = (k: string) => (fields[k] ? i18n.server(fields[k]) : '');
</script>

<div class="grid gap-4 sm:grid-cols-2">
	<div>
		<label for="date" class="block text-sm font-medium">{i18n.t('day.date')}</label>
		<input
			id="date"
			name="date"
			type="date"
			required
			class={input}
			value={day?.date ?? defaultDate}
			aria-invalid={!!fields.date}
		/>
		{#if err('date')}<p class="mt-1 text-sm text-red-700">{err('date')}</p>{/if}
	</div>
	<div>
		<label for="kind" class="block text-sm font-medium">{i18n.t('day.type')}</label>
		<select id="kind" name="kind" class={input} bind:value={kind} aria-invalid={!!fields.kind}>
			{#each KINDS as k (k)}<option value={k}>{kindLabel(i18n, k)}</option>{/each}
		</select>
		{#if err('kind')}<p class="mt-1 text-sm text-red-700">{err('kind')}</p>{/if}
	</div>
	<div class="sm:col-span-2">
		<label for="name" class="block text-sm font-medium">{i18n.t('day.name')}</label>
		<input
			id="name"
			name="name"
			required
			minlength="2"
			maxlength="200"
			class={input}
			value={day?.name ?? ''}
			aria-invalid={!!fields.name}
		/>
		{#if err('name')}<p class="mt-1 text-sm text-red-700">{err('name')}</p>{/if}
	</div>
	{#if kind === 'COMPANY_HOLIDAY'}
		<fieldset class="sm:col-span-2">
			<legend class="text-sm font-medium">
				{i18n.t('day.partOfDay')}
				<span class="font-normal text-slate-500">{i18n.t('day.partHint')}</span>
			</legend>
			<div class="mt-1 grid grid-cols-2 gap-4">
				<div>
					<label for="start_time" class="block text-xs text-slate-600">{i18n.t('day.from')}</label>
					<input
						id="start_time"
						name="start_time"
						type="text"
						inputmode="numeric"
						pattern="([01][0-9]|2[0-3]):[0-5][0-9]"
						placeholder="HH:mm"
						maxlength="5"
						autocomplete="off"
						class="{input} tabular-nums"
						value={day?.start_time ?? ''}
					/>
				</div>
				<div>
					<label for="end_time" class="block text-xs text-slate-600">{i18n.t('day.to')}</label>
					<input
						id="end_time"
						name="end_time"
						type="text"
						inputmode="numeric"
						pattern="([01][0-9]|2[0-4]):[0-5][0-9]"
						placeholder="HH:mm"
						maxlength="5"
						autocomplete="off"
						class="{input} tabular-nums"
						value={day?.end_time ?? ''}
					/>
				</div>
			</div>
			{#if err('start_time') || err('end_time')}<p class="mt-1 text-sm text-red-700">
					{err('start_time') || err('end_time')}
				</p>{/if}
		</fieldset>
	{/if}
	<div>
		<label for="source" class="block text-sm font-medium"
			>{i18n.t('day.source')}
			<span class="font-normal text-slate-500">{i18n.t('common.optional')}</span></label
		>
		<input
			id="source"
			name="source"
			maxlength="255"
			class={input}
			value={day?.source ?? ''}
			placeholder={i18n.t('day.sourcePh')}
		/>
	</div>
	<div>
		<label for="note" class="block text-sm font-medium"
			>{i18n.t('day.note')}
			<span class="font-normal text-slate-500">{i18n.t('common.optional')}</span></label
		>
		<input id="note" name="note" maxlength="1000" class={input} value={day?.note ?? ''} />
	</div>
</div>
