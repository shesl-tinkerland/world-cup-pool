<script lang="ts">
	import type { MatchOdds } from '$lib/tips.svelte';
	import { language } from '$lib/language.svelte';
	import { strings } from '$lib/strings';

	let {
		odds,
		source,
		showDecimal = $bindable(false)
	}: {
		odds: MatchOdds;
		source: string;
		showDecimal?: boolean;
	} = $props();

	const t = $derived(strings[language.resolved].odds);

	function pct(p: number): string {
		return Math.round(p * 100) + '%';
	}

	let pHome = $derived(odds.pHome);
	let pDraw = $derived(odds.pDraw);
	let pAway = $derived(odds.pAway);

	let homeLeader = $derived(pHome >= pAway && pHome >= pDraw);
	let awayLeader = $derived(pAway > pHome && pAway >= pDraw);

	let sourceLabel = $derived(
		source === 'odds_api' ? t.sourceOddsApi : t.sourceRankings
	);
	let hasDecimal = $derived(odds.homeOdds > 0);
</script>

<div class="odds-row">
	<span class="prob-pill" class:leader={homeLeader}>
		H <b>{showDecimal && hasDecimal ? odds.homeOdds.toFixed(2) : pct(pHome)}</b>
	</span>
	<span class="prob-pill">
		D <b>{showDecimal && hasDecimal ? odds.drawOdds.toFixed(2) : pct(pDraw)}</b>
	</span>
	<span class="prob-pill" class:leader={awayLeader}>
		A <b>{showDecimal && hasDecimal ? odds.awayOdds.toFixed(2) : pct(pAway)}</b>
	</span>
	{#if hasDecimal}
		<button
			class="odds-toggle"
			onclick={() => (showDecimal = !showDecimal)}
			aria-label={showDecimal ? t.toggleToPct : t.toggleToDecimal}
			title={showDecimal ? t.toggleToPct : t.toggleToDecimal}
		>
			{showDecimal ? '%' : '1.00'}
		</button>
	{/if}
</div>
<span class="odds-src">{sourceLabel}</span>

<style>
	.odds-row {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		margin: 0.45rem 0 0.2rem;
	}
	.prob-pill {
		display: inline-flex;
		align-items: center;
		gap: 0.2rem;
		padding: 0.18rem 0.55rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: var(--surface-2);
		font-size: 0.78rem;
		color: var(--muted);
		white-space: nowrap;
	}
	.prob-pill b {
		font-family: var(--font-mono);
		font-weight: 700;
		color: var(--text);
	}
	.prob-pill.leader {
		border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
		background: color-mix(in srgb, var(--accent) 10%, var(--surface-2));
	}
	.prob-pill.leader b {
		color: var(--accent);
	}
	.odds-toggle {
		margin-left: auto;
		padding: 0.18rem 0.45rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: var(--surface-2);
		color: var(--muted);
		font: inherit;
		font-size: 0.72rem;
		cursor: pointer;
		line-height: 1;
	}
	.odds-toggle:hover {
		border-color: var(--border-strong);
		color: var(--text);
	}
	.odds-src {
		display: block;
		font-size: 0.7rem;
		color: var(--muted);
		margin-bottom: 0.15rem;
		opacity: 0.7;
	}
</style>
