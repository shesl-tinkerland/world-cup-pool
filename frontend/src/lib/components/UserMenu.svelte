<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { serverClock } from '$lib/serverclock.svelte';
	import { theme } from '$lib/theme.svelte';
	import { language } from '$lib/language.svelte';
	import { strings } from '$lib/strings';
	import Avatar from './Avatar.svelte';
	import {
		LogOut,
		ChevronDown,
		FlaskConical,
		Info,
		Languages,
		Moon,
		Settings,
		Sun,
		Trophy
	} from '@lucide/svelte';

	let {
		align = 'right' as 'right' | 'left',
		up = false,
		showName = false,
		showThemeAction = false
	}: { align?: 'right' | 'left'; up?: boolean; showName?: boolean; showThemeAction?: boolean } =
		$props();
	let open = $state(false);
	let root: HTMLElement;
	const isDark = $derived(theme.resolved === 'dark');
	const isWorldCup = $derived(theme.isWorldCup);
	const t = $derived(strings[language.resolved]);

	function toggleTheme() {
		theme.toggle();
		open = false;
	}

	function toggleWorldCupTheme() {
		theme.toggleWorldCup();
		open = false;
	}

	function onDocClick(e: MouseEvent) {
		if (root && !root.contains(e.target as Node)) open = false;
	}
	$effect(() => {
		document.addEventListener('click', onDocClick);
		return () => document.removeEventListener('click', onDocClick);
	});
</script>

<div class="um" bind:this={root}>
	<button
		class="trigger"
		onclick={() => (open = !open)}
		aria-haspopup="menu"
		aria-expanded={open}
	>
		<Avatar name={auth.user?.name ?? '?'} src={auth.user?.avatarUrl} size={36} />
		{#if showName}<span class="tname">{auth.user?.name}</span>{/if}
		<ChevronDown size={16} class="chev {open ? 'up' : ''}" />
	</button>

	{#if open}
		<div
			class="panel"
			class:left={align === 'left'}
			class:up
			role="menu"
		>
			<a class="who" href="/settings" onclick={() => (open = false)}>
				<Avatar name={auth.user?.name ?? '?'} src={auth.user?.avatarUrl} size={40} />
				<div class="meta">
					<div class="name">{auth.user?.name}</div>
					<div class="email">{auth.user?.email}</div>
				</div>
			</a>
			<a class="item" href="/settings" onclick={() => (open = false)}>
				<Settings size={17} /> {t.chrome.settings}
			</a>
			<a class="item" href="/info" onclick={() => (open = false)}>
				<Info size={17} /> {t.chrome.about}
			</a>
			<button class="item" type="button" onclick={() => language.toggle()}>
				<Languages size={17} /> {t.chrome.language}
			</button>
			{#if showThemeAction}
				<button class="item" type="button" onclick={toggleTheme}>
					{#if isDark}
						<Sun size={17} /> {t.chrome.lightTheme}
					{:else}
						<Moon size={17} /> {t.chrome.darkTheme}
					{/if}
				</button>
			{/if}
			<button
				class="item worldcup"
				class:active={isWorldCup}
				type="button"
				onclick={toggleWorldCupTheme}
			>
				<Trophy size={17} /> {isWorldCup ? t.chrome.standardTheme : t.chrome.worldCupTheme}
			</button>
			{#if serverClock.dev}
				<a class="item" href="/dev" onclick={() => (open = false)}>
					<FlaskConical size={17} /> {language.text('Utviklerverktøy', 'Utviklarverktøy', 'Dev tools')}
				</a>
			{/if}
			<button class="item" type="button" onclick={() => auth.logout()}>
				<LogOut size={17} /> {t.chrome.logout}
			</button>
		</div>
	{/if}
</div>

<style>
	.um {
		position: relative;
	}
	.trigger {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		width: 100%;
		background: none;
		border: none;
		padding: 0;
		color: var(--muted);
	}
	.tname {
		flex: 1;
		min-width: 0;
		text-align: left;
		font-weight: 700;
		color: var(--text);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	:global(.um .chev) {
		transition: transform 0.15s ease;
	}
	:global(.um .chev.up) {
		transform: rotate(180deg);
	}
	.panel {
		position: absolute;
		top: calc(100% + 0.5rem);
		right: 0;
		min-width: 220px;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 0.5rem;
		box-shadow: var(--shadow-pop);
		z-index: 60;
	}
	.panel.left {
		right: auto;
		left: 0;
	}
	.panel.up {
		top: auto;
		bottom: calc(100% + 0.5rem);
	}
	.who {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.5rem 0.5rem 0.7rem;
		border-bottom: 1px solid var(--border);
		margin-bottom: 0.4rem;
		color: var(--text);
		border-radius: var(--radius-sm) var(--radius-sm) 0 0;
	}
	.who:hover {
		background: var(--surface);
	}
	.name {
		font-weight: 700;
	}
	.email {
		font-size: 0.8rem;
		color: var(--muted);
	}
	.item {
		display: flex;
		align-items: center;
		gap: 0.55rem;
		width: 100%;
		padding: 0.6rem 0.55rem;
		background: none;
		border: none;
		border-radius: var(--radius-sm);
		color: var(--text);
		font: inherit;
		text-align: left;
		cursor: pointer;
	}
	.item:hover {
		background: var(--surface);
	}
	.item.active {
		background: color-mix(in srgb, var(--accent) 12%, transparent);
		color: var(--accent);
	}
	.item.worldcup :global(svg) {
		color: var(--gold);
	}
</style>
