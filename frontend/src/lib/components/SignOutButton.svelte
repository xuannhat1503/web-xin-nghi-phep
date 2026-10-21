<script lang="ts">
	import { useI18n } from '#lib/i18n/index.ts';

	// Plain form POST so sign-out works without JavaScript. Go validates Origin + csrf_token.
	let { csrfToken, class: className = '' }: { csrfToken: string | null; class?: string } = $props();
	const i18n = useI18n();
</script>

<form method="POST" action="/auth/logout" data-sveltekit-reload>
	<input type="hidden" name="csrf_token" value={csrfToken ?? ''} />
	<button type="submit" class={className}>{i18n.t('common.signOut')}</button>
</form>
