<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { useI18n, type MessageKey } from '#lib/i18n/index.ts';
	import { leaveTypeLabel, statusLabel, yearOptions } from '#lib/format.ts';

	// Filters live in the URL (shareable, back button works). Any change resets to page 1.
	// Without JavaScript this is a normal GET form.
	let { showDraft = true }: { showDraft?: boolean } = $props();
	const i18n = useI18n();
	const sp = $derived(page.url.searchParams);
	const v = (k: string) => sp.get(k) ?? '';
	let keyword = $derived(v('q'));
	let code = $derived(v('code'));

	const years = $derived(yearOptions(Number(sp.get('year')) || null));
	const statuses = $derived(
		['DRAFT', 'PENDING', 'NEEDS_CHANGES', 'APPROVED', 'REJECTED', 'CANCELLED'].filter(
			(s) => showDraft || s !== 'DRAFT'
		)
	);
	const advanced = $derived(
		['submitted_from', 'submitted_to', 'reviewed_from', 'reviewed_to', 'from', 'to'].some((k) =>
			sp.has(k)
		)
	);

	function apply(form: HTMLFormElement) {
		// eslint-disable-next-line svelte/prefer-svelte-reactivity -- temporary value, not state
		const u = new URL(page.url.href);
		u.search = '';
		for (const [k, val] of new FormData(form)) {
			if (typeof val === 'string' && val.trim()) u.searchParams.set(k, val.trim());
		}
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		goto(u.pathname + u.search, { replace: true, reset: false });
	}
	let timer: ReturnType<typeof setTimeout> | undefined;
	function debounced(e: Event) {
		const form = (e.currentTarget as HTMLInputElement).form!;
		clearTimeout(timer);
		timer = setTimeout(() => apply(form), 400);
	}
	const input = 'mt-1 w-full rounded-lg border-slate-300 text-sm';
	const label = 'block text-xs font-medium text-slate-600';
</script>

<form
	method="GET"
	class="rounded-xl border border-slate-200 bg-white p-4"
	role="search"
	onsubmit={(e) => {
		e.preventDefault();
		apply(e.currentTarget);
	}}
>
	<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
		<div>
			<label for="f-year" class={label}>{i18n.t('filter.year')}</label>
			<select id="f-year" name="year" class={input} onchange={(e) => apply(e.currentTarget.form!)}>
				<option value="">{i18n.t('filter.allYears')}</option>
				{#each years as y (y)}<option value={y} selected={v('year') === String(y)}>{y}</option
					>{/each}
			</select>
		</div>
		<div>
			<label for="f-month" class={label}>{i18n.t('filter.month')}</label>
			<select
				id="f-month"
				name="month"
				class={input}
				onchange={(e) => apply(e.currentTarget.form!)}
			>
				<option value="">{i18n.t('filter.allMonths')}</option>
				{#each Array.from({ length: 12 }, (_, i) => i + 1) as m (m)}
					<option value={m} selected={v('month') === String(m)}
						>{i18n.t(`month.${m}` as MessageKey)}</option
					>
				{/each}
			</select>
		</div>
		<div>
			<label for="f-status" class={label}>{i18n.t('filter.status')}</label>
			<select
				id="f-status"
				name="status"
				class={input}
				onchange={(e) => apply(e.currentTarget.form!)}
			>
				<option value="">{i18n.t('filter.allStatuses')}</option>
				{#each statuses as s (s)}<option value={s} selected={v('status') === s}
						>{statusLabel(i18n, s)}</option
					>{/each}
			</select>
		</div>
		<div>
			<label for="f-type" class={label}>{i18n.t('filter.type')}</label>
			<select
				id="f-type"
				name="leave_type"
				class={input}
				onchange={(e) => apply(e.currentTarget.form!)}
			>
				<option value="">{i18n.t('filter.allTypes')}</option>
				{#each i18n.leaveTypes as lt (lt.code)}<option
						value={lt.code}
						selected={v('leave_type') === lt.code}>{leaveTypeLabel(i18n, lt.code)}</option
					>{/each}
			</select>
		</div>
		<div>
			<label for="f-code" class={label}>{i18n.t('filter.code')}</label>
			<input
				id="f-code"
				name="code"
				maxlength="30"
				placeholder="LR-2026-"
				class={input}
				bind:value={code}
				oninput={debounced}
			/>
		</div>
		<div>
			<label for="f-q" class={label}>{i18n.t('filter.keyword')}</label>
			<input
				id="f-q"
				name="q"
				type="search"
				maxlength="100"
				class={input}
				bind:value={keyword}
				oninput={debounced}
			/>
		</div>
		<div>
			<label for="f-sort" class={label}>{i18n.t('filter.sort')}</label>
			<select id="f-sort" name="sort" class={input} onchange={(e) => apply(e.currentTarget.form!)}>
				<option value="newest" selected={!v('sort') || v('sort') === 'newest'}
					>{i18n.t('sort.newest')}</option
				>
				<option value="oldest" selected={v('sort') === 'oldest'}>{i18n.t('sort.oldest')}</option>
				<option value="start" selected={v('sort') === 'start'}>{i18n.t('sort.start')}</option>
			</select>
		</div>
		<div class="flex items-end gap-2">
			<button
				class="flex-1 rounded-lg bg-brand-700 px-3 py-2 text-sm font-medium text-white hover:bg-brand-800"
				>{i18n.t('filter.apply')}</button
			>
			<!-- eslint-disable svelte/no-navigation-without-resolve -->
			<a
				href={page.url.pathname}
				class="rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
				>{i18n.t('filter.clear')}</a
			>
			<!-- eslint-enable svelte/no-navigation-without-resolve -->
		</div>
	</div>
	<details class="mt-3" open={advanced}>
		<summary class="cursor-pointer text-sm font-medium text-brand-700"
			>{i18n.t('filter.more')}</summary
		>
		<div class="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
			{#each [['from', 'filter.from'], ['to', 'filter.to'], ['submitted_from', 'filter.submittedFrom'], ['submitted_to', 'filter.submittedTo'], ['reviewed_from', 'filter.reviewedFrom'], ['reviewed_to', 'filter.reviewedTo']] as [k, l] (k)}
				<div>
					<label for="f-{k}" class={label}>{i18n.t(l as MessageKey)}</label>
					<input
						id="f-{k}"
						name={k}
						type="date"
						class={input}
						value={v(k)}
						onchange={(e) => apply(e.currentTarget.form!)}
					/>
				</div>
			{/each}
		</div>
	</details>
</form>
