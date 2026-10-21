<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import LangSwitch from '#lib/components/LangSwitch.svelte';
	import Logo from '#lib/components/Logo.svelte';
	import SignOutButton from '#lib/components/SignOutButton.svelte';
	import Toaster from '#lib/components/Toaster.svelte';
	import { useI18n } from '#lib/i18n/index.ts';
	import { isCurrent, navigation } from '#lib/navigation.ts';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();
	const i18n = useI18n();

	// The (app) group is only reachable by active users (enforced in hooks), so user is set here.
	const user = $derived(data.user!);
	const sections = $derived(navigation(user, data.isReviewer));
	// The logo goes to the same landing page as sign-in (admin dashboard for Admins).
	const homeHref = $derived(
		user.role === 'ADMIN' ? resolve('/(app)/admin/dashboard') : resolve('/(app)/dashboard')
	);
	// Opening the notifications page marks everything read, so the badge clears there and stays
	// cleared until the layout data is reloaded with a new count.
	let unread = $derived(data.unread);
	$effect(() => {
		if (isCurrent(page.url.pathname, '/notifications')) unread = 0;
	});
	const initials = $derived(
		user.full_name
			.split(' ')
			.filter(Boolean)
			.slice(-2)
			.map((w: string) => w[0]!.toUpperCase())
			.join('')
	);
</script>

{#snippet nav()}
	<nav aria-label={i18n.t('nav.main')} class="space-y-6">
		{#each sections as section (section.title)}
			<div>
				<p class="px-3 text-xs font-semibold tracking-wider text-brand-100/60 uppercase">
					{i18n.t(section.title)}
				</p>
				<ul class="mt-2 space-y-1">
					{#each section.items as item (item.href)}
						{@const current = isCurrent(page.url.pathname, item.href)}
						<li>
							<!-- eslint-disable svelte/no-navigation-without-resolve -->
							<a
								href={item.href}
								aria-current={current ? 'page' : undefined}
								class="block rounded-md px-3 py-2 text-sm font-medium {current
									? 'bg-white/10 text-white'
									: 'text-brand-100 hover:bg-white/5 hover:text-white'}"
							>
								{i18n.t(item.label)}
							</a>
							<!-- eslint-enable svelte/no-navigation-without-resolve -->
						</li>
					{/each}
				</ul>
			</div>
		{/each}
	</nav>
{/snippet}

<a
	href="#main"
	class="sr-only z-50 rounded bg-white px-3 py-2 text-sm focus:not-sr-only focus:fixed focus:top-2 focus:left-2"
>
	{i18n.t('common.skip')}
</a>

<div class="min-h-dvh lg:grid lg:grid-cols-[16rem_1fr]">
	<aside class="hidden bg-brand-900 px-4 py-6 lg:flex lg:flex-col lg:gap-8">
		<a href={homeHref} class="px-3"><Logo class="h-9 w-auto" /></a>
		{@render nav()}
	</aside>

	<div class="flex min-w-0 flex-col">
		<header
			class="sticky top-0 z-30 border-b border-brand-950/10 bg-brand-900 text-white lg:bg-white lg:text-slate-900"
		>
			<div class="flex h-14 items-center gap-2 px-4 sm:gap-3 sm:px-6">
				<!-- Mobile menu: <details> works without JavaScript -->
				<details class="relative lg:hidden">
					<summary
						class="flex h-10 w-10 cursor-pointer list-none items-center justify-center rounded-md hover:bg-white/10 [&::-webkit-details-marker]:hidden"
						aria-label={i18n.t('common.openMenu')}
					>
						<svg
							aria-hidden="true"
							viewBox="0 0 24 24"
							class="h-6 w-6"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
						>
							<path stroke-linecap="round" d="M4 6h16M4 12h16M4 18h16" />
						</svg>
					</summary>
					<div
						class="absolute top-12 left-0 w-64 max-w-[calc(100vw-2rem)] rounded-lg bg-brand-900 p-4 shadow-xl"
					>
						{@render nav()}
					</div>
				</details>
				<a href={homeHref} class="lg:hidden"><Logo class="h-7 w-auto" /></a>

				<div class="ml-auto flex min-w-0 items-center gap-2 sm:gap-3">
					<a
						href={resolve('/(app)/notifications')}
						class="relative flex h-9 w-9 items-center justify-center rounded-full hover:bg-white/10 lg:hover:bg-slate-100"
						aria-label="{i18n.t('nav.notifications')}{unread ? ` (${unread})` : ''}"
					>
						<svg
							aria-hidden="true"
							viewBox="0 0 24 24"
							class="h-5 w-5"
							fill="none"
							stroke="currentColor"
							stroke-width="1.8"
							><path
								d="M6 8a6 6 0 1 1 12 0c0 7 3 9 3 9H3s3-2 3-9M10.3 21a1.94 1.94 0 0 0 3.4 0"
							/></svg
						>
						{#if unread}
							<span
								class="absolute -top-0.5 -right-0.5 min-w-[1.1rem] rounded-full bg-red-600 px-1 text-center text-[10px] leading-[1.1rem] font-semibold text-white"
								>{unread > 99 ? '99+' : unread}</span
							>
						{/if}
					</a>
					<div class="hidden lg:block"><LangSwitch /></div>
					<div class="lg:hidden"><LangSwitch dark /></div>
					<div class="hidden min-w-0 text-right sm:block">
						<p class="truncate text-sm font-medium">{user.full_name}</p>
						<p class="truncate text-xs opacity-70">{user.email}</p>
					</div>
					<span
						class="hidden h-9 w-9 shrink-0 items-center justify-center rounded-full bg-brand-600 text-sm font-semibold text-white min-[400px]:flex"
						aria-hidden="true">{initials}</span
					>
					<SignOutButton
						csrfToken={data.csrfToken}
						class="rounded-md px-2 py-2 text-sm font-medium whitespace-nowrap hover:bg-white/10 sm:px-3 lg:hover:bg-slate-100"
					/>
				</div>
			</div>
		</header>

		<main id="main" class="mx-auto w-full max-w-6xl flex-1 px-4 py-6 sm:px-6 lg:py-8">
			{@render children()}
		</main>
	</div>
</div>

<Toaster />
