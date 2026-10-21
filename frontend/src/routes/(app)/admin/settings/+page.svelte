<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import Alert from '#lib/components/Alert.svelte';
	import ConfirmButton from '#lib/components/ConfirmButton.svelte';
	import PageHeader from '#lib/components/PageHeader.svelte';
	import { useI18n, type MessageKey } from '#lib/i18n/index.ts';
	import { formatDateTime, formatMinutes, ISO_WEEKDAYS, leaveTypeLabel } from '#lib/format.ts';
	import { withToast, type ActionData } from '#lib/forms.ts';
	import type { PageProps } from './$types';

	let { data, form: rawForm }: PageProps = $props();
	const form = $derived(rawForm as ActionData);
	const i18n = useI18n();
	const enhanced = withToast(i18n);
	const s = $derived(data.settings);
	const p = $derived(s.policy);
	const fe = $derived((form?.fields ?? {}) as Record<string, string>);
	const open = $derived(
		(s.open_requests.PENDING ?? 0) +
			(s.open_requests.NEEDS_CHANGES ?? 0) +
			(s.open_requests.DRAFT ?? 0)
	);

	// Tabs live in the URL (?tab=…) so a reload, a form submit or a shared link keeps the same tab.
	const TABS = ['time', 'rules', 'reminders', 'types', 'reasons', 'email', 'history'] as const;
	type Tab = (typeof TABS)[number];
	const POLICY_TABS: Tab[] = ['time', 'rules', 'reminders'];
	const tab = $derived.by((): Tab => {
		const t = page.url.searchParams.get('tab');
		return (TABS as readonly string[]).includes(t ?? '') ? (t as Tab) : 'time';
	});
	const inPolicy = $derived(POLICY_TABS.includes(tab));
	// Which policy tab a server-side field error belongs to (shown as a dot on the tab).
	const fieldTab = (k: string): Tab =>
		k.startsWith('reminder_')
			? 'reminders'
			: k.startsWith('leave_') || k === 'min_leave_minutes'
				? 'rules'
				: 'time';
	const tabHasError = (t: Tab) => Object.keys(fe).some((k) => fieldTab(k) === t);
	const act = (name: string) => `?tab=${tab}&/${name}`;

	const activeTypes = $derived(s.leave_types.filter((t) => t.active));

	// Selected email mode (drives the confirmation text; the server re-checks every rule).
	let mailMode = $derived(s.email.mode);
	const accessText = $derived(
		[s.email.allowed_domain && '@' + s.email.allowed_domain, ...s.email.allowed_emails]
			.filter(Boolean)
			.join(', ')
	);

	// Live preview of the working-day length (the server recomputes and validates).
	let ws = $derived(p.work_start);
	let we = $derived(p.work_end);
	let ls = $derived(p.lunch_start);
	let le = $derived(p.lunch_end);
	const toMin = (t: string) => {
		const [h, m] = t.split(':').map(Number);
		return (h ?? 0) * 60 + (m ?? 0);
	};
	const dayMinutes = $derived(toMin(we) - toMin(ws) - (toMin(le) - toMin(ls)));
	const fmt = (m: number) => (m > 0 ? formatMinutes(i18n, m) : '—');

	const input = 'mt-1 w-full rounded-lg border-slate-300 text-sm';
	const time = 'mt-1 w-full rounded-lg border-slate-300 text-sm tabular-nums';
	const card = 'rounded-xl border border-slate-200 bg-white p-5';
	const small = 'mt-0.5 w-full rounded-lg border-slate-300 text-sm';
	const primary =
		'rounded-lg bg-brand-700 px-4 py-2 text-sm font-medium text-white hover:bg-brand-800';
	const secondary =
		'rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50';
	const danger =
		'rounded-lg border border-red-200 px-3 py-2 text-sm font-medium text-red-700 hover:bg-red-50';
	const err = (k: string) => (fe[k] ? i18n.server(fe[k]) : '');
	const pattern = '([01][0-9]|2[0-3]):[0-5][0-9]';
</script>

<svelte:head><title>{i18n.t('set.title')} · G-ASIAPACIFIC</title></svelte:head>

<PageHeader title={i18n.t('set.title')} description={i18n.t('set.desc')} />

<nav aria-label={i18n.t('set.tabs')} class="mt-6 overflow-x-auto border-b border-slate-200">
	<ul class="flex min-w-max gap-1">
		{#each TABS as t (t)}
			<li>
				<a
					href={resolve(`/(app)/admin/settings?tab=${t}`)}
					data-sveltekit-noscroll
					data-sveltekit-replacestate
					aria-current={tab === t ? 'page' : undefined}
					class="relative block border-b-2 px-4 py-2.5 text-sm font-medium whitespace-nowrap {tab ===
					t
						? 'border-brand-700 text-brand-800'
						: 'border-transparent text-slate-600 hover:border-slate-300 hover:text-slate-900'}"
				>
					{i18n.t(`set.tab.${t}` as MessageKey)}
					{#if tabHasError(t)}<span
							class="absolute top-2 right-1 size-2 rounded-full bg-red-600"
							aria-label={i18n.t('set.tabError')}
						></span>{/if}
				</a>
			</li>
		{/each}
	</ul>
</nav>

{#if form?.error || form?.errorKey}
	<div class="mt-4">
		<Alert tone="error">{form?.error ? i18n.server(form.error) : i18n.msg(form?.errorKey)}</Alert>
	</div>
{:else if form?.success}
	<div class="mt-4"><Alert tone="success">{i18n.msg(form.success)}</Alert></div>
{/if}

<!-- One policy form across three tabs: hidden sections are still submitted, so saving from any of
     them creates one complete policy version. -->
<form
	method="POST"
	action={act('policy')}
	use:enhance={enhanced}
	class="mt-6 space-y-6"
	hidden={!inPolicy}
>
	<input type="hidden" name="version" value={s.settings_version} />

	<section class={card} aria-labelledby="wt-h" hidden={tab !== 'time'}>
		<h2 id="wt-h" class="font-semibold">{i18n.t('set.workingTime')}</h2>
		<p class="mt-1 text-sm text-slate-600">
			{i18n.t('set.currentPolicy', { v: p.version_no, d: formatMinutes(i18n, p.minutes_per_day) })}
		</p>
		<div class="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<div>
				<label for="work_start" class="block text-sm font-medium">{i18n.t('set.workStart')}</label
				><input
					id="work_start"
					name="work_start"
					type="text"
					inputmode="numeric"
					{pattern}
					placeholder="HH:mm"
					maxlength="5"
					required
					class={time}
					bind:value={ws}
				/>{#if err('work_start')}<p class="mt-1 text-sm text-red-700">{err('work_start')}</p>{/if}
			</div>
			<div>
				<label for="work_end" class="block text-sm font-medium">{i18n.t('set.workEnd')}</label
				><input
					id="work_end"
					name="work_end"
					type="text"
					inputmode="numeric"
					{pattern}
					placeholder="HH:mm"
					maxlength="5"
					required
					class={time}
					bind:value={we}
				/>{#if err('work_end')}<p class="mt-1 text-sm text-red-700">{err('work_end')}</p>{/if}
			</div>
			<div>
				<label for="lunch_start" class="block text-sm font-medium">{i18n.t('set.lunchStart')}</label
				><input
					id="lunch_start"
					name="lunch_start"
					type="text"
					inputmode="numeric"
					{pattern}
					placeholder="HH:mm"
					maxlength="5"
					required
					class={time}
					bind:value={ls}
				/>{#if err('lunch_start')}<p class="mt-1 text-sm text-red-700">{err('lunch_start')}</p>{/if}
			</div>
			<div>
				<label for="lunch_end" class="block text-sm font-medium">{i18n.t('set.lunchEnd')}</label
				><input
					id="lunch_end"
					name="lunch_end"
					type="text"
					inputmode="numeric"
					{pattern}
					placeholder="HH:mm"
					maxlength="5"
					required
					class={time}
					bind:value={le}
				/>{#if err('lunch_end')}<p class="mt-1 text-sm text-red-700">{err('lunch_end')}</p>{/if}
			</div>
		</div>
		<p class="mt-2 text-sm text-slate-600" aria-live="polite">
			{i18n.t('set.workingDay', {
				d: fmt(dayMinutes),
				m: fmt(toMin(ls) - toMin(ws)),
				a: fmt(toMin(we) - toMin(le))
			})}
		</p>

		<fieldset class="mt-4">
			<legend class="text-sm font-medium">{i18n.t('set.workdays')}</legend>
			<div class="mt-2 flex flex-wrap gap-2">
				{#each ISO_WEEKDAYS as d (d)}
					<label
						class="flex items-center gap-1.5 rounded-lg border border-slate-300 px-3 py-1.5 text-sm has-[:checked]:border-brand-600 has-[:checked]:bg-brand-50"
					>
						<input
							type="checkbox"
							name="workdays"
							value={d}
							checked={p.workdays.includes(d)}
							class="rounded border-slate-300"
						/>{i18n.t(`wd.${d}` as MessageKey)}
					</label>
				{/each}
			</div>
			{#if err('workdays')}<p class="mt-1 text-sm text-red-700">{err('workdays')}</p>{/if}
		</fieldset>

		<div class="mt-4 max-w-sm">
			<label for="timezone" class="block text-sm font-medium">{i18n.t('set.timezone')}</label>
			<input id="timezone" name="timezone" required class={input} value={p.timezone} />
			{#if err('timezone')}<p class="mt-1 text-sm text-red-700">{err('timezone')}</p>{/if}
		</div>
	</section>

	<section class={card} aria-labelledby="lr-h" hidden={tab !== 'rules'}>
		<h2 id="lr-h" class="font-semibold">{i18n.t('set.leaveRules')}</h2>
		<div class="mt-4 grid gap-4 sm:grid-cols-2">
			<div>
				<label for="leave_unit_minutes" class="block text-sm font-medium"
					>{i18n.t('set.unit')}</label
				>
				<select id="leave_unit_minutes" name="leave_unit_minutes" class={input}>
					{#each [15, 30, 60] as u (u)}<option value={u} selected={p.leave_unit_minutes === u}
							>{i18n.t('set.unitOpt', { n: u })}</option
						>{/each}
				</select>
				{#if err('leave_unit_minutes')}<p class="mt-1 text-sm text-red-700">
						{err('leave_unit_minutes')}
					</p>{/if}
			</div>
			<div>
				<label for="min_leave_minutes" class="block text-sm font-medium"
					>{i18n.t('set.minLeave')}</label
				>
				<input
					id="min_leave_minutes"
					name="min_leave_minutes"
					type="number"
					min="15"
					max="1440"
					step="15"
					required
					class={input}
					value={p.min_leave_minutes}
				/>
				{#if err('min_leave_minutes')}<p class="mt-1 text-sm text-red-700">
						{err('min_leave_minutes')}
					</p>{/if}
			</div>
		</div>

		<p class="mt-5 text-sm text-slate-600">{i18n.t('set.rulesHelp')}</p>
		{#if err('leave_types')}<p class="mt-1 text-sm text-red-700">{err('leave_types')}</p>{/if}
		<div class="mt-3 overflow-x-auto">
			<table class="w-full min-w-[36rem] text-left text-sm">
				<caption class="sr-only">{i18n.t('set.rulesCaption')}</caption>
				<thead class="text-xs tracking-wide text-slate-500 uppercase">
					<tr>
						<th class="py-2 pr-3 font-medium">{i18n.t('set.col.type')}</th>
						<th class="px-3 py-2 font-medium">{i18n.t('set.col.notice')}</th>
						<th class="px-3 py-2 font-medium">{i18n.t('set.col.allowPast')}</th>
						<th class="px-3 py-2 font-medium">{i18n.t('set.col.pastDays')}</th>
						<th class="px-3 py-2 font-medium">{i18n.t('set.col.reason')}</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each activeTypes as lt (lt.code)}
						{@const t = lt.code}
						{@const r = p.leave_types[t]}
						{@const label = leaveTypeLabel(i18n, t)}
						<tr>
							<th scope="row" class="py-2 pr-3 font-medium"
								><input type="hidden" name="rule_types" value={t} />{label}</th
							>
							<td class="px-3 py-2"
								><input
									aria-label="{label}: {i18n.t('set.col.notice')}"
									name="{t}.advance_notice_days"
									type="number"
									min="0"
									max="365"
									class="w-24 rounded-lg border-slate-300 text-sm"
									value={r?.advance_notice_days ?? 0}
								/></td
							>
							<td class="px-3 py-2"
								><input
									aria-label="{label}: {i18n.t('set.col.allowPast')}"
									name="{t}.allow_past_dates"
									type="checkbox"
									class="rounded border-slate-300"
									checked={r?.allow_past_dates}
								/></td
							>
							<td class="px-3 py-2"
								><input
									aria-label="{label}: {i18n.t('set.col.pastDays')}"
									name="{t}.max_past_days"
									type="number"
									min="0"
									max="365"
									class="w-24 rounded-lg border-slate-300 text-sm"
									value={r?.max_past_days ?? 0}
								/></td
							>
							<td class="px-3 py-2"
								><input
									aria-label="{label}: {i18n.t('set.col.reason')}"
									name="{t}.reason_required"
									type="checkbox"
									class="rounded border-slate-300"
									checked={r?.reason_required ?? true}
								/></td
							>
						</tr>
						{#if err(`leave_types.${t}`)}<tr
								><td colspan="5" class="pb-2 text-sm text-red-700">{err(`leave_types.${t}`)}</td
								></tr
							>{/if}
					{/each}
				</tbody>
			</table>
		</div>
	</section>

	<section class={card} aria-labelledby="rem-h" hidden={tab !== 'reminders'}>
		<h2 id="rem-h" class="font-semibold">{i18n.t('set.reminders')}</h2>
		<p class="mt-1 text-sm text-slate-600">{i18n.t('set.remindersHelp')}</p>
		<div class="mt-4 grid gap-4 sm:grid-cols-3">
			<div>
				<label for="reminder_interval_hours" class="block text-sm font-medium"
					>{i18n.t('set.every')}</label
				>
				<input
					id="reminder_interval_hours"
					name="reminder_interval_hours"
					type="number"
					min="1"
					max="168"
					required
					class={input}
					value={Math.round(p.reminder_interval_minutes / 60)}
				/>
				{#if err('reminder_interval_minutes')}<p class="mt-1 text-sm text-red-700">
						{err('reminder_interval_minutes')}
					</p>{/if}
			</div>
			<div>
				<label for="reminder_basis" class="block text-sm font-medium"
					>{i18n.t('set.countedIn')}</label
				>
				<select id="reminder_basis" name="reminder_basis" class={input}>
					<option value="WORKING_HOURS" selected={p.reminder_basis === 'WORKING_HOURS'}
						>{i18n.t('set.workingHours')}</option
					>
					<option value="CLOCK_HOURS" selected={p.reminder_basis === 'CLOCK_HOURS'}
						>{i18n.t('set.clockHours')}</option
					>
				</select>
			</div>
			<div>
				<label for="reminder_max_count" class="block text-sm font-medium"
					>{i18n.t('set.maxReminders')}</label
				>
				<input
					id="reminder_max_count"
					name="reminder_max_count"
					type="number"
					min="0"
					max="20"
					required
					class={input}
					value={p.reminder_max_count}
				/>
				{#if err('reminder_max_count')}<p class="mt-1 text-sm text-red-700">
						{err('reminder_max_count')}
					</p>{/if}
			</div>
		</div>
	</section>

	<section class={card}>
		<label for="change_note" class="block text-sm font-medium"
			>{i18n.t('set.changeNote')}
			<span class="font-normal text-slate-500">{i18n.t('set.changeNoteHint')}</span></label
		>
		<input
			id="change_note"
			name="change_note"
			maxlength="500"
			class={input}
			placeholder={i18n.t('set.changeNotePh')}
		/>
		<div class="mt-4">
			<ConfirmButton
				label={i18n.t('set.save')}
				title={i18n.t('set.saveTitle')}
				message={open > 0 ? i18n.t('set.saveMsgOpen', { n: open }) : i18n.t('set.saveMsg')}
				confirmLabel={i18n.t('common.save')}
				class={primary}
			/>
		</div>
	</section>
</form>

{#if tab === 'types'}
	<section class="{card} mt-6" aria-labelledby="lt-h">
		<h2 id="lt-h" class="font-semibold">{i18n.t('set.types')}</h2>
		<p class="mt-1 text-sm text-slate-600">{i18n.t('set.typesHelp')}</p>
		<ul class="mt-4 divide-y divide-slate-100">
			{#each s.leave_types as t (t.code)}
				<li class="flex flex-wrap items-end gap-3 py-3">
					<form
						method="POST"
						action={act('updateType')}
						use:enhance={enhanced}
						class="flex flex-1 flex-wrap items-end gap-3"
					>
						<input type="hidden" name="type_code" value={t.code} />
						<div class="min-w-44 flex-1">
							<label for="tvi-{t.code}" class="block text-xs text-slate-500"
								>{t.code} · {i18n.t('set.nameVi')}</label
							>
							<input
								id="tvi-{t.code}"
								name="name_vi"
								required
								minlength="2"
								maxlength="100"
								class={small}
								value={t.name_vi}
							/>
						</div>
						<div class="min-w-44 flex-1">
							<label for="ten-{t.code}" class="block text-xs text-slate-500"
								>{i18n.t('set.nameEn')}</label
							>
							<input
								id="ten-{t.code}"
								name="name_en"
								required
								minlength="2"
								maxlength="100"
								class={small}
								value={t.name_en}
							/>
						</div>
						<div>
							<label for="tso-{t.code}" class="block text-xs text-slate-500"
								>{i18n.t('set.order')}</label
							>
							<input
								id="tso-{t.code}"
								name="sort_order"
								type="number"
								min="0"
								max="9999"
								class="mt-0.5 w-20 rounded-lg border-slate-300 text-sm"
								value={t.sort_order}
							/>
						</div>
						<label class="flex items-center gap-1.5 py-2 text-sm"
							><input
								type="checkbox"
								name="active"
								checked={t.active}
								class="rounded border-slate-300"
							/>{i18n.t('set.inUse')}</label
						>
						<button class={secondary}
							>{i18n.t('common.save')}<span class="sr-only"> {t.code}</span></button
						>
					</form>
					<form method="POST" action={act('deleteType')} use:enhance={enhanced}>
						<input type="hidden" name="type_code" value={t.code} />
						<ConfirmButton
							label={i18n.t('common.delete')}
							title={i18n.t('set.deleteTypeTitle', { name: leaveTypeLabel(i18n, t.code) })}
							message={i18n.t('set.deleteTypeMsg')}
							confirmLabel={i18n.t('common.delete')}
							tone="danger"
							class={danger}
						/>
					</form>
				</li>
			{/each}
		</ul>
		<details class="mt-4" open={form?.typeForm === 'new' && !!form?.error}>
			<summary class="cursor-pointer text-sm font-medium text-brand-700"
				>{i18n.t('set.addType')}</summary
			>
			<form
				method="POST"
				action={act('createType')}
				use:enhance={enhanced}
				class="mt-3 flex flex-wrap items-end gap-3"
			>
				<div>
					<label for="nt-code" class="block text-xs text-slate-500">{i18n.t('set.code')}</label
					><input
						id="nt-code"
						name="code"
						required
						maxlength="40"
						pattern="[A-Za-z][A-Za-z0-9_]+"
						class="mt-0.5 w-44 rounded-lg border-slate-300 text-sm uppercase"
					/>
				</div>
				<div class="min-w-44 flex-1">
					<label for="nt-vi" class="block text-xs text-slate-500">{i18n.t('set.nameVi')}</label
					><input id="nt-vi" name="name_vi" required minlength="2" maxlength="100" class={small} />
				</div>
				<div class="min-w-44 flex-1">
					<label for="nt-en" class="block text-xs text-slate-500">{i18n.t('set.nameEn')}</label
					><input id="nt-en" name="name_en" required minlength="2" maxlength="100" class={small} />
				</div>
				<div>
					<label for="nt-sort" class="block text-xs text-slate-500">{i18n.t('set.order')}</label
					><input
						id="nt-sort"
						name="sort_order"
						type="number"
						min="0"
						max="9999"
						value="100"
						class="mt-0.5 w-20 rounded-lg border-slate-300 text-sm"
					/>
				</div>
				<button
					class="rounded-lg bg-brand-700 px-3 py-2 text-sm font-medium text-white hover:bg-brand-800"
					>{i18n.t('common.add')}</button
				>
			</form>
			<p class="mt-2 text-xs text-slate-500">{i18n.t('set.newTypeNote')}</p>
		</details>
	</section>
{/if}

{#if tab === 'reasons'}
	<section class="{card} mt-6" aria-labelledby="rs-h">
		<h2 id="rs-h" class="font-semibold">{i18n.t('set.reasons')}</h2>
		<p class="mt-1 text-sm text-slate-600">{i18n.t('set.reasonsHelp')}</p>
		<ul class="mt-4 divide-y divide-slate-100">
			{#each s.reasons as r (r.id)}
				<li class="flex flex-wrap items-end gap-3 py-3">
					<form
						method="POST"
						action={act('updateReason')}
						use:enhance={enhanced}
						class="flex flex-1 flex-wrap items-end gap-3"
					>
						<input type="hidden" name="reason_id" value={r.id} />
						<div class="min-w-44 flex-1">
							<label for="label-{r.id}" class="block text-xs text-slate-500"
								>{r.code} · {i18n.t('set.labelVi')}</label
							>
							<input
								id="label-{r.id}"
								name="label"
								required
								minlength="2"
								maxlength="200"
								class={small}
								value={r.label}
							/>
						</div>
						<div class="min-w-44 flex-1">
							<label for="labelen-{r.id}" class="block text-xs text-slate-500"
								>{i18n.t('set.labelEn')}</label
							>
							<input
								id="labelen-{r.id}"
								name="label_en"
								required
								minlength="2"
								maxlength="200"
								class={small}
								value={r.label_en}
							/>
						</div>
						<div>
							<label for="sort-{r.id}" class="block text-xs text-slate-500"
								>{i18n.t('set.order')}</label
							>
							<input
								id="sort-{r.id}"
								name="sort_order"
								type="number"
								min="0"
								max="9999"
								class="mt-0.5 w-20 rounded-lg border-slate-300 text-sm"
								value={r.sort_order}
							/>
						</div>
						<label class="flex items-center gap-1.5 py-2 text-sm"
							><input
								type="checkbox"
								name="requires_text"
								checked={r.requires_text}
								class="rounded border-slate-300"
							/>{i18n.t('set.needsText')}</label
						>
						<label class="flex items-center gap-1.5 py-2 text-sm"
							><input
								type="checkbox"
								name="active"
								checked={r.active}
								class="rounded border-slate-300"
							/>{i18n.t('set.inUse')}</label
						>
						<button class={secondary}
							>{i18n.t('common.save')}<span class="sr-only"> {r.code}</span></button
						>
					</form>
					{#if r.code !== 'OTHER'}
						<form method="POST" action={act('deleteReason')} use:enhance={enhanced}>
							<input type="hidden" name="reason_id" value={r.id} />
							<ConfirmButton
								label={i18n.t('common.delete')}
								title={i18n.t('set.deleteReasonTitle', { name: i18n.reason(r.code, r.label) })}
								message={i18n.t('set.deleteReasonMsg')}
								confirmLabel={i18n.t('common.delete')}
								tone="danger"
								class={danger}
							/>
						</form>
					{/if}
				</li>
			{/each}
		</ul>
		<details class="mt-4" open={form?.reasonForm === 'new' && !!form?.error}>
			<summary class="cursor-pointer text-sm font-medium text-brand-700"
				>{i18n.t('set.addReason')}</summary
			>
			<form
				method="POST"
				action={act('createReason')}
				use:enhance={enhanced}
				class="mt-3 flex flex-wrap items-end gap-3"
			>
				<div>
					<label for="new-code" class="block text-xs text-slate-500">{i18n.t('set.code')}</label
					><input
						id="new-code"
						name="code"
						required
						maxlength="40"
						pattern="[A-Za-z][A-Za-z0-9_]+"
						class="mt-0.5 w-40 rounded-lg border-slate-300 text-sm uppercase"
					/>
				</div>
				<div class="min-w-44 flex-1">
					<label for="new-label" class="block text-xs text-slate-500">{i18n.t('set.labelVi')}</label
					><input
						id="new-label"
						name="label"
						required
						minlength="2"
						maxlength="200"
						class={small}
					/>
				</div>
				<div class="min-w-44 flex-1">
					<label for="new-label-en" class="block text-xs text-slate-500"
						>{i18n.t('set.labelEn')}</label
					><input
						id="new-label-en"
						name="label_en"
						required
						minlength="2"
						maxlength="200"
						class={small}
					/>
				</div>
				<div>
					<label for="new-sort" class="block text-xs text-slate-500">{i18n.t('set.order')}</label
					><input
						id="new-sort"
						name="sort_order"
						type="number"
						min="0"
						max="9999"
						value="100"
						class="mt-0.5 w-20 rounded-lg border-slate-300 text-sm"
					/>
				</div>
				<label class="flex items-center gap-1.5 py-2 text-sm"
					><input type="checkbox" name="requires_text" class="rounded border-slate-300" />{i18n.t(
						'set.needsText'
					)}</label
				>
				<button
					class="rounded-lg bg-brand-700 px-3 py-2 text-sm font-medium text-white hover:bg-brand-800"
					>{i18n.t('common.add')}</button
				>
			</form>
		</details>
	</section>
{/if}

{#if tab === 'email'}
	<section class="{card} mt-6" aria-labelledby="mail-h">
		<h2 id="mail-h" class="font-semibold">{i18n.t('set.email')}</h2>
		<p class="mt-1 text-sm text-slate-600">{i18n.t('set.emailHelp')}</p>
		<ul class="mt-3 space-y-1 text-sm">
			<li class={s.email.smtp_configured ? 'text-emerald-700' : 'text-amber-800'}>
				{s.email.smtp_configured
					? i18n.t('set.smtpOk', { from: s.email.smtp_from })
					: i18n.t('set.smtpMissing')}
			</li>
			{#if !s.email.live_enabled}<li class="text-amber-800">{i18n.t('set.liveLocked')}</li>{/if}
			<li class={accessText ? 'text-slate-600' : 'text-amber-800'}>
				{accessText ? i18n.t('set.accessList', { list: accessText }) : i18n.t('set.accessOpen')}
			</li>
		</ul>
		<form method="POST" action={act('email')} use:enhance={enhanced} class="mt-4 space-y-4">
			<input type="hidden" name="version" value={s.settings_version} />
			<fieldset>
				<legend class="text-sm font-medium">{i18n.t('set.emailMode')}</legend>
				<div class="mt-2 grid gap-2 sm:grid-cols-2">
					<label
						class="flex items-start gap-2 rounded-lg border border-slate-300 p-3 text-sm has-[:checked]:border-brand-600 has-[:checked]:bg-brand-50"
						><input type="radio" name="mode" value="OFF" bind:group={mailMode} class="mt-0.5" />
						<span
							><span class="font-medium">{i18n.t('set.modeOff')}</span><br /><span
								class="text-slate-600">{i18n.t('set.modeOffHelp')}</span
							></span
						></label
					>
					<label
						class="flex items-start gap-2 rounded-lg border border-slate-300 p-3 text-sm has-[:checked]:border-brand-600 has-[:checked]:bg-brand-50 has-[:disabled]:opacity-60"
						><input
							type="radio"
							name="mode"
							value="LIVE"
							bind:group={mailMode}
							disabled={!s.email.smtp_configured || !s.email.live_enabled}
							class="mt-0.5"
						/>
						<span
							><span class="font-medium">{i18n.t('set.modeLive')}</span><br /><span
								class="text-slate-600">{i18n.t('set.modeLiveHelp')}</span
							></span
						></label
					>
				</div>
				{#if err('mode')}<p class="mt-1 text-sm text-red-700">{err('mode')}</p>{/if}
			</fieldset>
			<ConfirmButton
				label={i18n.t('set.saveEmail')}
				title={mailMode === 'LIVE' ? i18n.t('set.saveLiveTitle') : i18n.t('set.saveOffTitle')}
				message={mailMode === 'LIVE' ? i18n.t('set.saveLiveMsg') : i18n.t('set.saveOffMsg')}
				confirmLabel={i18n.t('common.save')}
				tone={mailMode === 'LIVE' ? 'danger' : 'primary'}
				class={primary}
			/>
		</form>
	</section>

	<section class="{card} mt-6" aria-labelledby="fin-h">
		<h2 id="fin-h" class="font-semibold">{i18n.t('set.finance')}</h2>
		<p class="mt-1 text-sm text-slate-600">{i18n.t('set.financeHelp')}</p>
		<form method="POST" action={act('finance')} use:enhance={enhanced} class="mt-3 space-y-3">
			<input type="hidden" name="version" value={s.settings_version} />
			<label for="finance_cc_emails" class="sr-only">{i18n.t('set.financeLabel')}</label>
			<textarea
				id="finance_cc_emails"
				name="finance_cc_emails"
				rows="3"
				class={input}
				placeholder={i18n.t('set.financePh')}>{s.finance_cc_emails.join('\n')}</textarea
			>
			{#if err('finance_cc_emails')}<p class="text-sm text-red-700">
					{err('finance_cc_emails')}
				</p>{/if}
			<button class={secondary}>{i18n.t('set.saveCc')}</button>
		</form>
	</section>
{/if}

{#if tab === 'history'}
	<section class="{card} mt-6" aria-labelledby="hist-h">
		<h2 id="hist-h" class="font-semibold">
			{i18n.t('set.versions', { n: s.history?.length ?? 0 })}
		</h2>
		{#if s.history?.length}
			<ol class="mt-3 space-y-2 text-sm">
				{#each s.history as h (h.version_no)}
					<li>
						<span class="font-medium">v{h.version_no}</span> · {formatDateTime(h.created_at)} · {h.created_by ===
						'System'
							? i18n.t('common.system')
							: h.created_by}{#if h.change_note}
							— {h.change_note}{/if}
					</li>
				{/each}
			</ol>
		{/if}
	</section>
{/if}
