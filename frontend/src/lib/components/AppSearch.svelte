<script lang="ts">
	import { goto } from '$app/navigation';
	import { tick } from 'svelte';
	import { api, type LeagueSummary } from '$lib/api';
	import { language } from '$lib/language.svelte';
	import { strings } from '$lib/strings';
	import { searchNav } from '$lib/searchNav.svelte';
	import { searchApp, totalSearchResults, type SearchGroup, type SearchResult } from '$lib/search';
	import { tipsStore } from '$lib/tips.svelte';
	import { Network, Search, Trophy, Users, Volleyball, X } from '@lucide/svelte';

	let { compact = false }: { compact?: boolean } = $props();

	let open = $state(false);
	let query = $state('');
	let leagues = $state<LeagueSummary[]>([]);
	let leaguesLoaded = $state(false);
	let leaguesBusy = $state(false);
	let inputEl = $state<HTMLInputElement | null>(null);
	const t = $derived(strings[language.resolved]);

	const results = $derived(
		searchApp(query, {
			matches: tipsStore.matches,
			teams: tipsStore.teams,
			leagues
		})
	);
	const resultCount = $derived(totalSearchResults(results));
	const hasQuery = $derived(query.trim().length > 0);
	const loading = $derived(open && (!tipsStore.loaded || leaguesBusy));

	const sections = $derived<{ key: SearchGroup; label: string; icon: typeof Volleyball }[]>([
		{ key: 'matches', label: t.search.matches, icon: Volleyball },
		{ key: 'teams', label: t.search.teams, icon: Users },
		{ key: 'groups', label: t.search.groups, icon: Network },
		{ key: 'leagues', label: t.search.leagues, icon: Trophy }
	]);

	function portal(node: HTMLElement) {
		if (typeof document === 'undefined') return;
		document.body.appendChild(node);
		return {
			destroy() {
				node.remove();
			}
		};
	}

	async function ensureData() {
		const tasks: Promise<unknown>[] = [];
		if (!tipsStore.loaded) {
			tasks.push(tipsStore.load().catch(() => {}));
		}
		if (!leaguesLoaded && !leaguesBusy) {
			leaguesBusy = true;
			tasks.push(
				api
					.myLeagues()
					.then((res) => {
						leagues = res.leagues ?? [];
						leaguesLoaded = true;
					})
					.catch(() => {
						leagues = [];
						leaguesLoaded = true;
					})
					.finally(() => {
						leaguesBusy = false;
					})
			);
		}
		await Promise.all(tasks);
	}

	async function openSearch() {
		open = true;
		void ensureData();
		await tick();
		inputEl?.focus();
	}

	function closeSearch() {
		open = false;
		query = '';
	}

	function onBackdropClick(event: MouseEvent) {
		if (event.target === event.currentTarget) closeSearch();
	}

	function onKeydown(event: KeyboardEvent) {
		if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
			event.preventDefault();
			void openSearch();
		}
		if (open && event.key === 'Escape') {
			closeSearch();
		}
	}

	function groupItems(group: SearchGroup): SearchResult[] {
		return results[group];
	}

	async function scrollAfterSearchNavigation(href: string) {
		const url = new URL(href, window.location.origin);
		const matchId = url.searchParams.get('match');
		const groupId = url.searchParams.get('group')?.trim().toUpperCase();
		const teamId = url.searchParams.get('team');
		if (!matchId && !groupId && !teamId) return;

		for (let attempt = 0; attempt < 32; attempt += 1) {
			await tick();
			await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));

			const target = matchId
				? document.getElementById(`match-${matchId}`)
				: groupId
					? document.getElementById(`section-group-${groupId}`)
					: document.querySelector('.match.spotlight');
			if (target instanceof HTMLElement) {
				target.scrollIntoView({
					behavior: 'smooth',
					block: matchId ? 'center' : 'start'
				});
				return;
			}
		}
	}

	async function selectResult(event: MouseEvent, item: SearchResult) {
		event.preventDefault();
		closeSearch();
		searchNav.bump();
		await goto(item.href, { keepFocus: true, noScroll: true });
		await scrollAfterSearchNavigation(item.href);
	}
</script>

<svelte:window onkeydown={onKeydown} />

<div class="app-search" class:compact>
	<button
		type="button"
		class="search-trigger"
		class:compact
		aria-label={t.search.trigger}
		aria-haspopup="dialog"
		aria-expanded={open}
		onclick={openSearch}
	>
		<Search size={18} />
		{#if !compact}<span>{t.search.trigger}</span>{/if}
	</button>

	{#if open}
		<div class="search-layer" use:portal onclick={onBackdropClick} role="presentation">
			<div class="search-panel" role="dialog" aria-modal="true" aria-label={t.search.panelAria} tabindex="-1">
				<div class="search-head">
					<div class="search-field">
						<Search size={18} />
						<input
							bind:this={inputEl}
							bind:value={query}
							type="search"
							placeholder={t.search.placeholder}
							aria-label={t.search.placeholder}
						/>
					</div>
					<button type="button" class="close" aria-label={t.search.close} onclick={closeSearch}>
						<X size={19} />
					</button>
				</div>

				<div class="search-body">
					{#if loading}
						<p class="muted state">{t.search.loading}</p>
					{:else if !hasQuery}
						<p class="muted state">{t.search.empty}</p>
					{:else if resultCount === 0}
						<p class="muted state">{t.search.noResults}</p>
					{:else}
						{#each sections as section (section.key)}
							{@const items = groupItems(section.key)}
							{#if items.length > 0}
								{@const Icon = section.icon}
								<div class="result-group">
									<h3><Icon size={15} /> {section.label}</h3>
									<div class="result-list">
										{#each items as item (item.group + item.id)}
											<a class="result" href={item.href} onclick={(event) => selectResult(event, item)}>
												<span class="result-title">{item.title}</span>
												<span class="result-subtitle">{item.subtitle}</span>
											</a>
										{/each}
									</div>
								</div>
							{/if}
						{/each}
					{/if}
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.app-search {
		position: relative;
	}
	.app-search.compact {
		flex: none;
		min-width: 38px;
	}
	.search-trigger {
		display: inline-flex;
		align-items: center;
		justify-content: flex-start;
		gap: 0.55rem;
		width: 100%;
		min-height: 38px;
		padding: 0.55rem 0.8rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: var(--surface-2);
		color: var(--text);
		font: inherit;
		font-weight: 700;
		cursor: pointer;
		transition:
			background 0.15s ease,
			border-color 0.15s ease,
			transform 0.15s ease;
	}
	.search-trigger:hover {
		background: var(--surface-3);
		border-color: var(--border-strong);
	}
	.search-trigger:active {
		transform: scale(0.98);
	}
	.search-trigger.compact {
		justify-content: center;
		width: 38px;
		height: 38px;
		padding: 0;
		flex: none;
	}
	.search-layer {
		position: fixed;
		inset: 0;
		z-index: 100;
		display: grid;
		place-items: start center;
		padding: max(4.5rem, env(safe-area-inset-top)) 1rem 1rem;
		background: color-mix(in srgb, var(--bg) 56%, transparent);
		backdrop-filter: blur(12px);
		animation: layer-in 0.14s ease-out;
	}
	.search-panel {
		width: min(640px, 100%);
		max-height: min(72dvh, 720px);
		display: flex;
		flex-direction: column;
		border: 1px solid var(--border);
		border-radius: 26px;
		background: var(--surface);
		box-shadow: var(--shadow-pop);
		overflow: hidden;
		animation: panel-in 0.18s cubic-bezier(0.22, 1, 0.36, 1);
	}
	.search-head {
		display: flex;
		align-items: center;
		gap: 0.65rem;
		padding: 0.75rem;
		border-bottom: 1px solid var(--border);
	}
	.search-field {
		flex: 1;
		display: flex;
		align-items: center;
		gap: 0.55rem;
		min-width: 0;
		padding: 0.65rem 0.8rem;
		border-radius: 18px;
		background: var(--surface-2);
		color: var(--muted);
	}
	.search-field input {
		width: 100%;
		min-width: 0;
		border: none;
		outline: none;
		background: transparent;
		color: var(--text);
		font: inherit;
		font-size: 1rem;
	}
	.close {
		display: inline-grid;
		place-items: center;
		width: 38px;
		height: 38px;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: var(--surface-2);
		color: var(--text);
		cursor: pointer;
	}
	.search-body {
		overflow: auto;
		padding: 0.65rem;
	}
	.state {
		margin: 0;
		padding: 1.2rem 0.75rem;
		text-align: center;
	}
	.result-group + .result-group {
		margin-top: 0.7rem;
	}
	.result-group h3 {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		margin: 0 0 0.35rem;
		padding: 0 0.25rem;
		font-size: 0.78rem;
		font-weight: 800;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: var(--muted);
	}
	.result-list {
		display: grid;
		gap: 0.28rem;
	}
	.result {
		display: grid;
		gap: 0.18rem;
		padding: 0.78rem 0.85rem;
		border: 1px solid transparent;
		border-radius: 16px;
		background: var(--surface-2);
		color: var(--text);
	}
	.result:hover,
	.result:focus-visible {
		border-color: var(--border-strong);
		background: var(--surface-3);
	}
	.result-title {
		font-weight: 800;
		line-height: 1.2;
	}
	.result-subtitle {
		font-size: 0.82rem;
		line-height: 1.3;
		color: var(--muted);
	}
	@keyframes layer-in {
		from { opacity: 0; }
		to { opacity: 1; }
	}
	@keyframes panel-in {
		from { opacity: 0; transform: translateY(0.6rem) scale(0.98); }
		to { opacity: 1; transform: translateY(0) scale(1); }
	}
	@media (max-width: 560px) {
		.search-layer {
			place-items: end center;
			padding: 1rem 0.75rem calc(var(--nav-h) + env(safe-area-inset-bottom) + 1.6rem);
		}
		.search-panel {
			max-height: min(74dvh, 640px);
			border-radius: 24px;
		}
	}
</style>
