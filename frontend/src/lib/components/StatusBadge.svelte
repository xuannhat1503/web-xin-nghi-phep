<script lang="ts">
	import Badge from '#lib/components/Badge.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import { statusLabel } from '#lib/format.ts';
	import { STATUS_TONES } from '#lib/leave.ts';

	let { status }: { status: keyof typeof STATUS_TONES } = $props();
	const i18n = useI18n();
</script>

<Badge tone={STATUS_TONES[status] ?? 'slate'} label={statusLabel(i18n, status)} />
