<script lang="ts">
	// League members' tips for a single match, shown as a mini leaderboard: best
	// score first, a gold crown on whoever nailed the match, and a starred gold
	// pill for a perfect 6-point tip. Avatars come from the league standings the
	// parent already loaded, so no extra fetch. Only rendered for locked matches —
	// the backend returns no other-player tips before kickoff.
	import { Crown } from '@lucide/svelte';
	import { language } from '$lib/language.svelte';
	import { tipsStore, type FriendTip } from '$lib/tips.svelte';
	import { teamDisplayName } from '$lib/teamNames';
	import Avatar from '$lib/components/Avatar.svelte';

	let {
		tips,
		isKO = false,
		avatars = {}
	}: { tips: FriendTip[]; isKO?: boolean; avatars?: Record<string, string> } = $props();

	const PREVIEW_COUNT = 8;
	let showAll = $state(false);

	function hasSubmittedTip(tip: FriendTip) {
		if (tip.hasTip === false) return false;
		return Number.isFinite(tip.ftHome) && Number.isFinite(tip.ftAway);
	}

	function hasScoredPoints(tip: FriendTip) {
		return Number.isFinite(tip.points) && tip.points >= 0;
	}

	let sorted = $derived.by<FriendTip[]>(() => {
		const rows = [...tips];
		rows.sort((a, b) => {
			const aHas = hasSubmittedTip(a);
			const bHas = hasSubmittedTip(b);
			if (aHas !== bHas) return aHas ? -1 : 1;
			if (a.points !== b.points) return b.points - a.points;
			if (a.isMe !== b.isMe) return a.isMe ? -1 : 1;
			return a.name.localeCompare(b.name, language.locale);
		});
		return rows;
	});
	// Highest positive score on this match — its holders get the crown.
	let topPoints = $derived(
		Math.max(0, ...tips.filter(hasScoredPoints).map((t) => t.points))
	);
	let hiddenCount = $derived(Math.max(sorted.length - PREVIEW_COUNT, 0));
	let visible = $derived(showAll ? sorted : sorted.slice(0, PREVIEW_COUNT));
</script>

{#if tips.length === 0}
	<p class="muted small empty">{language.text('Ingen i ligaen har tipset denne kampen.', 'Ingen i ligaen har tipsa denne kampen.', 'Nobody in the league tipped this match.')}</p>
{:else}
	<ul class="mtt">
		{#each visible as f, i (f.userId)}
			{@const hasTip = hasSubmittedTip(f)}
			{@const hasPoints = hasScoredPoints(f)}
			<li class="row" class:me={f.isMe} class:winner={hasPoints && f.points > 0 && f.points === topPoints} style="animation-delay:{Math.min(i, 6) * 18}ms">
				<span class="who">
					<Avatar name={f.name} src={avatars[f.userId] || null} size={26} />
					<span class="name">{f.name}</span>
					{#if hasPoints && f.points > 0 && f.points === topPoints}
						<Crown size={13} class="crown" />
					{/if}
					{#if f.isMe}<span class="metag">{language.text('deg', 'deg', 'you')}</span>{/if}
				</span>
				<span class="tip digits">
					{#if !hasTip}
						<span class="muted notip">{language.text('—', '—', '—')}</span>
					{:else}
						{f.ftHome}:{f.ftAway}
						{#if isKO && f.advancer}<span class="adv">→ {teamDisplayName(tipsStore.team(f.advancer))}</span>{/if}
					{/if}
				</span>
				<span class="pts">
					{#if !hasTip}
						<span class="pill none">{language.text('ikke levert', 'ikkje levert', 'no tip')}</span>
					{:else if f.points === 6}
						<span class="pill perfect"><span class="star">★</span>6</span>
					{:else if f.points > 0}
						<span class="pill ok">+{f.points}</span>
					{:else if f.points === 0}
						<span class="pill zero">0</span>
					{:else}
						<span class="muted notip">{language.text('—', '—', '—')}</span>
					{/if}
				</span>
			</li>
		{/each}
	</ul>
	{#if hiddenCount > 0}
		<div class="more">
			<button class="morebtn" onclick={() => (showAll = !showAll)}>
				{#if showAll}
					{language.text('Vis færre', 'Vis færre', 'Show fewer')}
				{:else}
					{language.text(`Vis ${hiddenCount} til`, `Vis ${hiddenCount} til`, `Show ${hiddenCount} more`)}
				{/if}
			</button>
		</div>
	{/if}
{/if}

<style>
	.empty {
		margin: 0.5rem 0 0.2rem;
		text-align: center;
	}
	.mtt {
		list-style: none;
		margin: 0.15rem 0 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	.row {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto auto;
		align-items: center;
		gap: 0.6rem;
		padding: 0.4rem 0.5rem;
		border-radius: 10px;
		animation: rowIn 0.2s ease-out both;
	}
	.row.me {
		background: color-mix(in srgb, var(--accent) 14%, transparent);
	}
	.row.winner:not(.me) {
		background: color-mix(in srgb, var(--gold) 9%, transparent);
	}
	.who {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		min-width: 0;
	}
	.name {
		font-size: 0.88rem;
		font-weight: 600;
		color: var(--text);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.row.me .name {
		font-weight: 800;
	}
	:global(.mtt .crown) {
		color: var(--gold);
		flex: none;
	}
	.metag {
		flex: none;
		font-size: 0.58rem;
		font-weight: 800;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--accent);
		background: color-mix(in srgb, var(--accent) 18%, transparent);
		padding: 0.08rem 0.32rem;
		border-radius: var(--radius-pill);
	}
	.tip {
		font-size: 0.9rem;
		font-weight: 700;
		color: var(--text);
		white-space: nowrap;
		text-align: right;
	}
	.notip {
		font-weight: 500;
	}
	.adv {
		font-size: 0.74rem;
		font-weight: 500;
		color: var(--muted);
	}
	.pts {
		justify-self: end;
	}
	.pill {
		display: inline-flex;
		align-items: center;
		gap: 0.15rem;
		min-width: 2.1rem;
		justify-content: center;
		font-size: 0.74rem;
		font-weight: 800;
		font-variant-numeric: tabular-nums;
		padding: 0.18rem 0.5rem;
		border-radius: var(--radius-pill);
		line-height: 1;
	}
	.pill.ok {
		color: var(--success);
		background: color-mix(in srgb, var(--success) 16%, transparent);
	}
	.pill.zero {
		color: var(--muted);
		background: var(--surface-2);
	}
	.pill.none {
		color: var(--muted-2);
		background: transparent;
		font-weight: 600;
		text-transform: lowercase;
	}
	.pill.perfect {
		color: #3a2a00;
		background: var(--gold);
	}
	:global(:root[data-theme='dark']) .pill.perfect,
	:global(:root[data-theme='worldcup']) .pill.perfect {
		color: #1a1200;
	}
	.pill.perfect .star {
		font-size: 0.7rem;
	}
	.more {
		margin-top: 0.45rem;
		text-align: center;
	}
	.morebtn {
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		color: var(--muted);
		font-size: 0.76rem;
		font-weight: 700;
		padding: 0.3rem 0.95rem;
		cursor: pointer;
		transition: color 0.15s ease, border-color 0.15s ease;
	}
	.morebtn:hover {
		color: var(--text);
		border-color: var(--border-strong);
	}
	.muted {
		color: var(--muted);
	}
	.small {
		font-size: 0.8rem;
	}
	@keyframes rowIn {
		from {
			opacity: 0;
			transform: translateY(6px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.row {
			animation: none;
		}
	}
</style>
