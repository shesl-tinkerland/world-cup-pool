<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { auth } from '$lib/auth.svelte';
	import { leagueBadges } from '$lib/leagueBadges.svelte';
	import { language } from '$lib/language.svelte';
	import { navItems, isActive } from '$lib/nav';
	import { strings } from '$lib/strings';
	import { tipsStore, isLocked, teamsResolved } from '$lib/tips.svelte';
	import { forecastStore as fs } from '$lib/forecast.svelte';
	import { serverClock } from '$lib/serverclock.svelte';

	let { variant = 'tab' as 'tab' | 'rail' } = $props();
	let path = $derived($page.url.pathname);
	const t = $derived(strings[language.resolved]);

	let now = $derived(serverClock.now());

	$effect(() => {
		if (!auth.isAuthed) {
			leagueBadges.clear();
			return;
		}
		leagueBadges.start();
		return () => leagueBadges.stop();
	});

	let missingMatchTips = $derived.by(() => {
		if (!tipsStore.loaded) return 0;
		return tipsStore.matches.filter(m => {
			if (new Date(m.kickoff).getTime() <= now) return false;
			return teamsResolved(m) && !isLocked(m) && !tipsStore.tips[m.id];
		}).length;
	});

	let vmTipsMissing = $derived.by(() => {
		if (!fs.loaded) return false;
		return !fs.locked && (!fs.recId || !fs.isComplete);
	});

	function getBadgeInfo(href: string): { count: number; show: boolean; isLive?: boolean; attention?: boolean } {
		if (href === '/') {
			return { count: 0, show: false, isLive: tipsStore.liveMatchIds.size > 0 };
		}
		if (href === '/tips') {
			return { count: missingMatchTips, show: missingMatchTips > 0 };
		}
		if (href === '/forecast') {
			return { count: 1, show: vmTipsMissing };
		}
		if (href === '/leagues') {
			const count = leagueBadges.totalCount;
			return { count, show: count > 0, attention: count > 0 };
		}
		return { count: 0, show: false };
	}

	function followNav(event: MouseEvent, href: string) {
		if (href !== '/leagues' || leagueBadges.totalCount <= 0) return;
		const target = leagueBadges.activityHref;
		if (target === href) return;
		event.preventDefault();
		void goto(target);
	}
</script>

<div class="links {variant}">
	{#each navItems as it (it.href)}
		{@const Icon = it.icon}
		{@const badge = getBadgeInfo(it.href)}
		<a href={it.href} class:active={isActive(it.href, path)} class:attention={badge.attention} onclick={(event) => followNav(event, it.href)}>
			<span class="icon-wrap">
				<Icon size={variant === 'rail' ? 20 : 22} />
				{#if variant === 'tab' && badge.show}
					<span class="badge-count">{badge.count}</span>
				{/if}
				{#if badge.isLive}
					<span class="live-dot-nav" aria-hidden="true"></span>
				{/if}
			</span>
			<span class="label-wrap">
				{t.nav[it.labelKey]}
				{#if variant === 'rail' && badge.show}
					<span class="badge-count">{badge.count}</span>
				{/if}
			</span>
		</a>
	{/each}
</div>

<style>
	.links {
		display: flex;
	}
	.links a {
		display: flex;
		align-items: center;
		color: var(--muted);
		position: relative;
		transition:
			background 0.18s ease,
			color 0.18s ease,
			transform 0.18s ease;
	}
	.links a .label-wrap {
		font-weight: 700;
		letter-spacing: 0;
		text-transform: uppercase;
	}
	.links a.active {
		color: var(--accent);
	}
	.links a.attention {
		color: var(--text);
	}

	.icon-wrap {
		position: relative;
		display: inline-flex;
	}

	/* Mobile bottom tab bar */
	.tab {
		flex: 1;
		gap: 0.15rem;
	}
	.tab a {
		flex: 1;
		flex-direction: column;
		justify-content: center;
		gap: 3px;
		min-width: 0;
		height: 100%;
		padding: 0.36rem 0.18rem;
		border-radius: 18px;
		font-size: 0.58rem;
		line-height: 1;
	}
	.tab a .label-wrap {
		max-width: 100%;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		text-transform: none;
	}
	.tab .badge-count {
		position: absolute;
		top: -4px;
		right: -8px;
		background: var(--live);
		color: var(--bg);
		font-size: 0.52rem;
		font-weight: 800;
		padding: 0.1rem 0.28rem;
		border-radius: 99px;
		line-height: 1;
		box-shadow: 0 0 0 2px var(--bg);
	}
	.live-dot-nav {
		position: absolute;
		top: -3px;
		right: -7px;
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: var(--live);
		box-shadow: 0 0 0 2px var(--bg);
		animation: navLivePulse 1.5s ease-in-out infinite;
	}
	@keyframes navLivePulse {
		0%, 100% { opacity: 1; transform: scale(1); }
		50% { opacity: 0.58; transform: scale(0.86); }
	}
	.tab a.active {
		background: color-mix(in srgb, var(--accent) 14%, transparent);
		color: var(--text);
		transform: translateY(-1px);
	}
	.tab a.attention:not(.active) {
		background: color-mix(in srgb, var(--live) 10%, transparent);
	}
	.tab a.attention .icon-wrap {
		animation: navAttentionPop 1.8s ease-in-out infinite;
	}
	.tab a.attention .badge-count {
		min-width: 1rem;
		text-align: center;
		background: var(--live);
		color: var(--bg);
		box-shadow:
			0 0 0 2px var(--bg),
			0 0 0 5px color-mix(in srgb, var(--live) 18%, transparent);
	}
	.tab a.active::before {
		content: '';
		position: absolute;
		top: 0.28rem;
		left: 50%;
		transform: translateX(-50%);
		width: 5px;
		height: 5px;
		border-radius: 50%;
		background: var(--accent);
		box-shadow: none;
	}
	@media (max-width: 360px) {
		.tab a {
			font-size: 0.53rem;
		}
	}

	/* Desktop side rail */
	.rail {
		flex-direction: column;
		gap: 0.15rem;
		width: 100%;
	}
	.rail a {
		gap: 0.85rem;
		padding: 0.7rem 1.5rem;
		font-size: 0.9rem;
	}
	.rail a .label-wrap {
		font-size: 0.82rem;
		text-transform: uppercase;
		letter-spacing: 0.04em;
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
	}
	.rail .badge-count {
		background: color-mix(in srgb, var(--live) 15%, transparent);
		color: var(--live);
		font-family: var(--font-mono);
		font-size: 0.68rem;
		font-weight: 800;
		padding: 0.15rem 0.4rem;
		border-radius: 6px;
		line-height: 1;
	}
	.rail a.attention {
		background: color-mix(in srgb, var(--live) 9%, transparent);
	}
	.rail a.attention .badge-count {
		background: var(--live);
		color: var(--bg);
		border-radius: 999px;
		box-shadow: 0 0 0 3px color-mix(in srgb, var(--live) 14%, transparent);
	}
	.rail a.active { color: var(--text); }
	.rail a.active::before {
		content: '';
		position: absolute;
		left: 0;
		top: 50%;
		transform: translateY(-50%);
		width: 3px;
		height: 60%;
		border-radius: 0 3px 3px 0;
		background: var(--text);
		box-shadow: none;
	}
	@keyframes navAttentionPop {
		0%,
		100% {
			transform: translateY(0) scale(1);
		}
		50% {
			transform: translateY(-1px) scale(1.06);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.live-dot-nav,
		.tab a.attention .icon-wrap {
			animation: none;
		}
	}
</style>
