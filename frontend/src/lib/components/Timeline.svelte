<script lang="ts">
	import { useI18n, hasKey } from '#lib/i18n/index.ts';
	import {
		formatDate,
		formatDateTime,
		formatMinutes,
		leaveTypeLabel,
		reasonLabel
	} from '#lib/format.ts';
	import type { TimelineEvent, VersionSnapshot } from '#lib/leave.ts';

	let { events, versions }: { events: TimelineEvent[]; versions: VersionSnapshot[] } = $props();
	const i18n = useI18n();
	const label = (a: string) => (hasKey(`ev.${a}`) ? i18n.t(`ev.${a}` as 'ev.CREATED') : a);
	const dot: Record<string, string> = {
		APPROVED: 'bg-emerald-600',
		REJECTED: 'bg-red-600',
		CHANGES_REQUESTED: 'bg-violet-600',
		CANCELLED: 'bg-slate-400'
	};
</script>

<section class="rounded-xl border border-slate-200 bg-white p-5" aria-labelledby="tl-h">
	<h2 id="tl-h" class="font-semibold">{i18n.t('req.timeline')}</h2>
	<ol class="mt-4 space-y-4 border-l border-slate-200 pl-5">
		{#each events as e, i (i)}
			<li class="relative text-sm">
				<span
					class="absolute top-1.5 -left-[25px] h-2.5 w-2.5 rounded-full {dot[e.action] ??
						'bg-brand-600'}"
				></span>
				<p class="font-medium">
					{label(e.action)}{e.submit_no && e.action.endsWith('SUBMITTED')
						? ` · ${i18n.t('req.versionN', { n: e.submit_no })}`
						: ''}
				</p>
				<p class="text-xs text-slate-500">
					{formatDateTime(e.created_at)} · {i18n.t('common.by', {
						name: e.actor === 'System' ? i18n.t('common.system') : e.actor
					})}
				</p>
				{#if e.comment}<p
						class="mt-1 rounded-md bg-slate-50 px-3 py-2 break-words whitespace-pre-line text-slate-700"
					>
						“{e.comment}”
					</p>{/if}
			</li>
		{/each}
	</ol>

	{#if versions.length > 1}
		<details class="mt-5">
			<summary class="cursor-pointer text-sm font-medium text-brand-700"
				>{i18n.t('req.versions')} ({versions.length})</summary
			>
			<ul class="mt-3 space-y-3 text-sm">
				{#each versions as v (v.submit_no)}
					<li class="rounded-lg border border-slate-200 p-3">
						<p class="font-medium">
							{i18n.t('req.versionN', { n: v.submit_no })} · {formatDateTime(v.created_at)}
						</p>
						<p>
							{leaveTypeLabel(i18n, v.leave_type)} · {formatDate(v.start_date)}
							{v.start_time} → {formatDate(v.end_date)}
							{v.end_time} · {formatMinutes(i18n, v.total_minutes)}
						</p>
						<p class="break-words text-slate-600">
							{reasonLabel(i18n, v.reason_code, v.reason_code)}{v.reason_text
								? ` — ${v.reason_text}`
								: ''}
						</p>
					</li>
				{/each}
			</ul>
		</details>
	{/if}
</section>
