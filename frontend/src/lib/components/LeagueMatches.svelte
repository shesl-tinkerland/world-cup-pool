<script lang="ts">
	// The "Kampar" tab on the league page: a match-by-match feed scoped to one
	// league. Reads the globally-loaded tipsStore (matches/tips/scores/live), so it
	// needs no fetching of its own beyond the per-match tips revealed on expand.
	import { language } from '$lib/language.svelte';
	import { tipsStore, isLiveStatus, type Match } from '$lib/tips.svelte';
	import { serverClock } from '$lib/serverclock.svelte';
	import LeagueMatchCard from '$lib/components/LeagueMatchCard.svelte';

	let { leagueId, avatars = {} }: { leagueId: string; avatars?: Record<string, string> } = $props();

	type Filter = 'liveRecent' | 'upcoming' | 'all';
	let filter = $state<Filter>('liveRecent');

	let now = $derived(serverClock.now());

	function played(m: Match) {
		return m.status === 'finished' || !!m.finalizedAt;
	}

	let liveRecent = $derived.by(() =>
		tipsStore.matches
			.filter((m) => isLiveStatus(m.status) || played(m))
			.sort((a, b) => {
				const al = isLiveStatus(a.status);
				const bl = isLiveStatus(b.status);
				if (al !== bl) return al ? -1 : 1;
				const at = new Date(a.kickoff).getTime();
				const bt = new Date(b.kickoff).getTime();
				return al && bl ? at - bt : bt - at;
			})
	);

	let upcoming = $derived.by(() =>
		tipsStore.matches
			.filter((m) => !played(m) && !isLiveStatus(m.status) && new Date(m.kickoff).getTime() > now)
			.sort((a, b) => new Date(a.kickoff).getTime() - new Date(b.kickoff).getTime())
	);

	// Now-centric: live first, then most-recently-played, then upcoming — so the
	// latest matches are always near the top whichever filter is active.
	let all = $derived.by(() => {
		const rank = (m: Match) => (isLiveStatus(m.status) ? 0 : played(m) ? 1 : 2);
		return [...tipsStore.matches].sort((a, b) => {
			const ra = rank(a);
			const rb = rank(b);
			if (ra !== rb) return ra - rb;
			const at = new Date(a.kickoff).getTime();
			const bt = new Date(b.kickoff).getTime();
			return ra === 1 ? bt - at : at - bt;
		});
	});

	let shown = $derived(filter === 'upcoming' ? upcoming : filter === 'all' ? all : liveRecent);
	let liveCount = $derived(tipsStore.liveMatchIds.size);
</script>

<div class="lm">
	<div class="filters" role="tablist">
		<button class="chip" class:on={filter === 'liveRecent'} role="tab" aria-selected={filter === 'liveRecent'} onclick={() => (filter = 'liveRecent')}>
			{language.text('Live & nylig', 'Live & nyleg', 'Live & recent')}
			{#if liveCount > 0}<span class="livedot" aria-hidden="true"></span>{/if}
		</button>
		<button class="chip" class:on={filter === 'upcoming'} role="tab" aria-selected={filter === 'upcoming'} onclick={() => (filter = 'upcoming')}>
			{language.text('Kommende', 'Komande', 'Upcoming')}
		</button>
		<button class="chip" class:on={filter === 'all'} role="tab" aria-selected={filter === 'all'} onclick={() => (filter = 'all')}>
			{language.text('Alle', 'Alle', 'All')}
		</button>
	</div>

	{#if !tipsStore.loaded}
		<p class="muted small">{language.text('Lastar…', 'Lastar…', 'Loading…')}</p>
	{:else if shown.length === 0}
		<div class="empty">
			{#if filter === 'liveRecent'}
				<p class="muted">{language.text('Ingen kamper har startet ennå.', 'Ingen kampar har starta enno.', 'No matches have started yet.')}</p>
				<button class="linkbtn" onclick={() => (filter = 'upcoming')}>{language.text('Se kommende kamper', 'Sjå komande kampar', 'See upcoming matches')}</button>
			{:else if filter === 'upcoming'}
				<p class="muted">{language.text('Ingen flere kamper igjen.', 'Ingen fleire kampar att.', 'No more matches left.')}</p>
			{:else}
				<p class="muted">{language.text('Ingen kamper ennå.', 'Ingen kampar enno.', 'No matches yet.')}</p>
			{/if}
		</div>
	{:else}
		{#key leagueId}
			<div class="list">
				{#each shown as match (match.id)}
					<LeagueMatchCard {match} {leagueId} {avatars} />
				{/each}
			</div>
		{/key}
	{/if}
</div>

<style>
	.lm {
		margin-top: 0.25rem;
	}
	.filters {
		display: flex;
		flex-wrap: wrap;
		gap: 0.45rem;
		margin-bottom: 0.85rem;
	}
	.chip {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		padding: 0.34rem 0.85rem;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		color: var(--muted);
		font-size: 0.8rem;
		font-weight: 600;
		cursor: pointer;
	}
	.chip.on {
		background: var(--text);
		border-color: var(--text);
		color: var(--bg);
	}
	.chip .livedot {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		background: var(--live);
		animation: lmLivePulse 1.5s ease-in-out infinite;
	}
	.empty {
		text-align: center;
		padding: 1.5rem 0.5rem;
	}
	.muted {
		color: var(--muted);
	}
	.small {
		font-size: 0.8rem;
	}
	.linkbtn {
		margin-top: 0.5rem;
		background: none;
		border: none;
		color: var(--accent);
		font-weight: 600;
		cursor: pointer;
		text-decoration: underline;
	}
	@keyframes lmLivePulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.4; }
	}
	@media (prefers-reduced-motion: reduce) {
		.chip .livedot {
			animation: none;
		}
	}
</style>
