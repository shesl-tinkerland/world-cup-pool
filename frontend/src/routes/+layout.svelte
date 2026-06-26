<script lang="ts">
	import '../app.css';
	import { browser } from '$app/environment';
	import { auth } from '$lib/auth.svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import Logo from '$lib/components/Logo.svelte';
	import AppSearch from '$lib/components/AppSearch.svelte';
	import UserMenu from '$lib/components/UserMenu.svelte';
	import NavLinks from '$lib/components/NavLinks.svelte';
	import LeagueActivityToast from '$lib/components/LeagueActivityToast.svelte';
	import PwaInstallButton from '$lib/components/PwaInstallButton.svelte';
	import PwaInstallBanner from '$lib/components/PwaInstallBanner.svelte';
	import NotifyPrompt from '$lib/components/NotifyPrompt.svelte';
	import InfoButton from '$lib/components/InfoButton.svelte';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import LanguageToggle from '$lib/components/LanguageToggle.svelte';
	import { serverClock } from '$lib/serverclock.svelte';
	import { language } from '$lib/language.svelte';
	import { tipsStore } from '$lib/tips.svelte';
	import { forecastStore as fs } from '$lib/forecast.svelte';

	let { children } = $props();
	let lastForegroundRefreshAt = 0;
	const foregroundRefreshMinIntervalMs = 10_000;

	function refreshLiveData(force = false) {
		if (!auth.isAuthed) return;
		const now = Date.now();
		if (!force && now - lastForegroundRefreshAt < foregroundRefreshMinIntervalMs) return;
		lastForegroundRefreshAt = now;
		void Promise.all([
			serverClock.refresh(),
			tipsStore.refresh().then(() => tipsStore.subscribe())
		]).catch(() => {});
	}

	// Keep the server clock fresh so long-running sessions lock tips correctly.
	$effect(() => {
		document.documentElement.lang = language.resolved;
	});

	$effect(() => {
		if (!auth.isAuthed) {
			serverClock.stopAutoRefresh();
			tipsStore.unsubscribe();
			return;
		}
		if (!tipsStore.loaded) tipsStore.load().then(() => tipsStore.subscribe()).catch(() => {});
		else void tipsStore.subscribe();
		if (!fs.loaded) fs.load().catch(() => {});

		serverClock.startAutoRefresh();
		return () => {
			serverClock.stopAutoRefresh();
			tipsStore.unsubscribe();
		};
	});

	$effect(() => {
		if (!browser || !auth.isAuthed) return;
		const onVisibilityChange = () => {
			if (document.visibilityState === 'visible') refreshLiveData();
		};
		const onFocus = () => refreshLiveData();
		const onPageShow = (event: PageTransitionEvent) => refreshLiveData(event.persisted);

		document.addEventListener('visibilitychange', onVisibilityChange);
		window.addEventListener('focus', onFocus);
		window.addEventListener('pageshow', onPageShow);
		return () => {
			document.removeEventListener('visibilitychange', onVisibilityChange);
			window.removeEventListener('focus', onFocus);
			window.removeEventListener('pageshow', onPageShow);
		};
	});

	// Signed-out-only pages — visible to anonymous users; signed-in users
	// bounce away to home (or /join if an invite is attached).
	const authPages = ['/login', '/register', '/forgot-password'];
	let path = $derived($page.url.pathname);
	let isAuthPage = $derived(authPages.includes(path));
	// Public routes — anyone can land here regardless of auth state:
	//   /join/<code>                 invite landing
	//   /confirm-password-reset/<t>  email reset target (must work even for
	//                                a still-signed-in user whose token was
	//                                requested by someone with their email)
	let isPublic = $derived(
		(path === '/' && !auth.isAuthed) ||
		path === '/info' ||
		path.startsWith('/join') ||
			path.startsWith('/confirm-password-reset/')
	);
	// No app chrome on the standalone auth / invite / reset screens.
	let chrome = $derived(auth.isAuthed && !isAuthPage && !isPublic);
	let showPublicThemeToggle = $derived(path !== '/');

	// SPA auth guard.
	$effect(() => {
		const invite = $page.url.searchParams.get('invite');
		if (!auth.isAuthed && !isAuthPage && !isPublic) {
			goto('/login', { replaceState: true });
		}
		// Already signed in: skip the auth pages. If they arrived via an
		// invite, send them to the join flow so it auto-joins.
		if (auth.isAuthed && isAuthPage) {
			goto(invite ? `/join/${invite}` : '/', { replaceState: true });
		}
	});
</script>

{#if chrome}
	<!-- Mobile: top header (logo / install / user menu) -->
	<header class="topbar">
		<Logo compact />
		<span class="topbar-tag">{language.text('VM 26', 'VM 26', 'World Cup 26')}</span>
		<div class="spacer"></div>
		<AppSearch compact />
		<PwaInstallButton />
		<UserMenu align="right" showThemeAction />
	</header>

	<!-- Desktop: left rail (logo top, links, user menu bottom) -->
	<aside class="siderail">
		<div class="rail-logo"><Logo /></div>
		<div class="rail-search"><AppSearch /></div>
		<NavLinks variant="rail" />
		<div class="spacer"></div>
		<div class="rail-actions"><LanguageToggle compact /><ThemeToggle compact /><InfoButton /></div>
		<div class="rail-user"><UserMenu align="left" up showName /></div>
	</aside>

	<!-- Mobile: bottom tab bar -->
	<nav class="tabbar"><NavLinks variant="tab" /></nav>

	<!-- One-time onboarding popup offering email/push notifications. -->
	<NotifyPrompt />
	<LeagueActivityToast />
{:else}
	<div class="public-topbar">
		<div class="public-topbar-shell">
			<div
				class="auth-actions"
				class:single={!showPublicThemeToggle}
				aria-label={showPublicThemeToggle
					? language.text('Visningsvalg og info', 'Visingsval og info', 'Display options and info')
					: language.text('Språkvalg og info', 'Språkval og info', 'Language options and info')}
			>
				<LanguageToggle compact />
				{#if showPublicThemeToggle}
					<ThemeToggle compact />
				{/if}
				<InfoButton />
			</div>
		</div>
	</div>
{/if}

<div class="app-shell" class:with-chrome={chrome} class:public-shell={!chrome}>
	{#if chrome}
		<PwaInstallBanner />
	{/if}
	{@render children()}
</div>

<style>
	.topbar-tag {
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 0.8rem;
		letter-spacing: 0.01em;
		color: var(--muted);
		white-space: nowrap;
	}
	.public-topbar {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		z-index: 20;
		padding:
			max(0.7rem, calc(env(safe-area-inset-top) + 0.35rem))
			max(0.85rem, calc(env(safe-area-inset-right) + 0.7rem))
			0
			max(0.85rem, calc(env(safe-area-inset-left) + 0.7rem));
		pointer-events: none;
	}
	.public-topbar-shell {
		width: min(var(--maxw), 100%);
		margin: 0 auto;
		display: flex;
		justify-content: flex-end;
	}
	:global(.app-shell.public-shell) {
		padding-top: calc(env(safe-area-inset-top) + 4.4rem);
	}
	.auth-actions {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		max-width: 100%;
		padding: 0.35rem;
		border: 1px solid color-mix(in srgb, var(--border) 78%, transparent);
		border-radius: 999px;
		background: color-mix(in srgb, var(--surface) 88%, transparent);
		box-shadow: 0 14px 30px rgba(3, 9, 14, 0.18);
		backdrop-filter: blur(18px);
		-webkit-backdrop-filter: blur(18px);
		pointer-events: auto;
	}
	.auth-actions.single {
		padding-inline: 0.3rem;
	}
	@media (max-width: 520px) {
		:global(.app-shell.public-shell) {
			padding-top: calc(env(safe-area-inset-top) + 4rem);
		}
		.public-topbar {
			padding-top: max(0.6rem, calc(env(safe-area-inset-top) + 0.25rem));
			padding-inline:
				max(0.75rem, calc(env(safe-area-inset-left) + 0.55rem))
				max(0.75rem, calc(env(safe-area-inset-right) + 0.55rem));
		}
		.auth-actions {
			gap: 0.4rem;
			padding: 0.3rem;
		}
	}
</style>
