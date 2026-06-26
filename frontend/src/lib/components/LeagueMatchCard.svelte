<script lang="ts">
	// One match row in the league "Kampar" view. Collapsed it shows the fixture /
	// live score / result plus the signed-in user's own tip; expanded it reveals
	// every league member's tip and points for that match. Other players' tips are
	// only fetched once the match is locked — the backend hides them before
	// kickoff, so an unlocked card shows only your own pick.
	import { onDestroy } from 'svelte';
	import { slide } from 'svelte/transition';
	import { ChevronDown, Lock } from '@lucide/svelte';
	import { language } from '$lib/language.svelte';
	import {
		tipsStore,
		isLiveStatus,
		isLocked,
		type Match,
		type FriendTip,
		type LiveEvent
	} from '$lib/tips.svelte';
	import { decisiveEvents, eventIcon, eventMinute, eventTeam, eventTitle } from '$lib/liveEvents';
	import { serverClock } from '$lib/serverclock.svelte';
	import { stageName } from '$lib/stageLabels';
	import Flag from '$lib/components/Flag.svelte';
	import MatchTipsTable from '$lib/components/MatchTipsTable.svelte';
	import TvLogo from '$lib/components/TvLogo.svelte';

	let {
		match,
		leagueId,
		avatars = {},
		flat = false,
		showTv = false
	}: {
		match: Match;
		leagueId: string;
		avatars?: Record<string, string>;
		flat?: boolean;
		showTv?: boolean;
	} = $props();

	let open = $state(false);
	let friends = $state<FriendTip[] | null>(null);
	let friendsBusy = $state(false);
	let lastRev = -1;
	let goalFlash = $state(false);
	let seenGoalCount: number | null = null;
	let goalFlashTimer: ReturnType<typeof setTimeout> | null = null;

	let home = $derived(tipsStore.team(match.homeTeam));
	let away = $derived(tipsStore.team(match.awayTeam));
	let now = $derived(serverClock.now());
	let live = $derived(isLiveStatus(match.status));
	let played = $derived(match.status === 'finished' || !!match.finalizedAt);
	let locked = $derived(isLocked(match));
	let isKO = $derived(match.stage !== 'group');
	let myTip = $derived(tipsStore.tips[match.id]);
	let myPts = $derived(tipsStore.scores[match.id]);
	let finishedEvents = $state<LiveEvent[] | null>(null);
	let liveEventsList = $derived(tipsStore.liveEvents[match.id] ?? []);
	// The true goal total, so decisiveEvents can drop provider over-counts.
	let expectedGoals = $derived(
		match.etHome || match.etAway ? match.etHome + match.etAway : match.ftHome + match.ftAway
	);
	let visibleEvents = $derived.by(() => {
		if (live) return decisiveEvents(liveEventsList, expectedGoals);
		if (played && finishedEvents) return decisiveEvents(finishedEvents, expectedGoals);
		return [];
	});

	// Which side won — drives the subtle winner emphasis on finished cards.
	let winnerSide = $derived.by(() => {
		if (!played) return '';
		if (match.penHome || match.penAway) return match.penHome > match.penAway ? 'home' : 'away';
		const hh = match.etHome || match.ftHome;
		const aa = match.etAway || match.ftAway;
		if (hh > aa) return 'home';
		if (aa > hh) return 'away';
		return 'draw';
	});

	let liveMin = $derived.by(() => {
		if (match.status === 'HT') return language.text('Pause', 'Pause', 'HT');
		const last = visibleEvents[visibleEvents.length - 1];
		if (last) {
			const created = new Date(last.created).getTime();
			if (Number.isFinite(created)) {
				const minutesSinceEvent = Math.max(0, Math.floor((now - created) / 60_000));
				const minute = Math.max(1, last.elapsed + last.extra + minutesSinceEvent);
				if (match.status === '1H' || match.status === 'LIVE' || match.status === 'live')
					return minute > 45 ? `45+${minute - 45}'` : `${minute}'`;
				if (match.status === '2H')
					return minute > 90 ? `90+${minute - 90}'` : `${minute}'`;
				return `${minute}'`;
			}
			return eventMinute(last);
		}
		if (match.status === '1H' || match.status === 'LIVE' || match.status === 'live')
			return language.text('1. omg', '1. omg', '1st');
		if (match.status === '2H') return language.text('2. omg', '2. omg', '2nd');
		if (match.status === 'ET' || match.status === 'BT')
			return language.text('E.o.', 'E.o.', 'ET');
		if (match.status === 'P') return language.text('Straffer', 'Straffer', 'Pens');
		return '';
	});

	$effect(() => {
		const goalCount = liveEventsList.filter((event) => event.type === 'Goal').length;
		if (!live) {
			seenGoalCount = null;
			goalFlash = false;
			return;
		}
		if (seenGoalCount === null) {
			seenGoalCount = goalCount;
			return;
		}
		if (goalCount <= seenGoalCount) {
			seenGoalCount = goalCount;
			return;
		}
		seenGoalCount = goalCount;
		goalFlash = true;
		if (goalFlashTimer) clearTimeout(goalFlashTimer);
		goalFlashTimer = setTimeout(() => {
			goalFlash = false;
			goalFlashTimer = null;
		}, 1100);
	});

	onDestroy(() => {
		if (goalFlashTimer) clearTimeout(goalFlashTimer);
	});

	function scoreText(m: Match) {
		let s = `${m.ftHome}–${m.ftAway}`;
		if (m.etHome || m.etAway) s = `${m.etHome}–${m.etAway} ${language.text('e.e.o.', 'e.eo.', 'aet')}`;
		if (m.penHome || m.penAway) s += ` (${m.penHome}–${m.penAway} ${language.text('str', 'str', 'pens')})`;
		return s;
	}

	function shortStage(m: Match) {
		if (m.stage === 'group') return `${language.text('Gruppe', 'Gruppe', 'Group')} ${m.groupLetter}`;
		return stageName(m.stage);
	}

	function kickoffLabel(iso: string) {
		const d = new Date(iso);
		const today = new Date();
		const sameDay =
			d.getFullYear() === today.getFullYear() &&
			d.getMonth() === today.getMonth() &&
			d.getDate() === today.getDate();
		const time = d.toLocaleTimeString(language.locale, { hour: '2-digit', minute: '2-digit' });
		if (sameDay) return `${language.text('I dag', 'I dag', 'Today')} ${time}`;
		return (
			d.toLocaleDateString(language.locale, { weekday: 'short', day: 'numeric', month: 'short' }) +
			' ' +
			time
		);
	}

	function homeCode() {
		return home?.fifaCode ?? match.homeLabel;
	}
	function awayCode() {
		return away?.fifaCode ?? match.awayLabel;
	}

	async function loadFriends() {
		friendsBusy = true;
		lastRev = tipsStore.scoreRevision;
		try {
			friends = await tipsStore.friends(match.id, leagueId);
		} catch {
			friends = [];
		} finally {
			friendsBusy = false;
		}
	}

	async function loadFinishedEvents() {
		finishedEvents = await tipsStore.loadMatchEvents(match.id);
	}

	function toggle() {
		open = !open;
		if (!open) return;
		if (locked && friends === null && !friendsBusy) void loadFriends();
		if (played && finishedEvents === null) void loadFinishedEvents();
	}

	// Keep an open, locked card's tips fresh as live results re-score. Mirrors the
	// leaderboard's scoreRevision watcher — re-fetch only when points actually move.
	$effect(() => {
		const rev = tipsStore.scoreRevision;
		if (!open || !locked || friendsBusy) return;
		if (rev === lastRev) return;
		void loadFriends();
	});
</script>

<div class="lmc" class:live class:open class:played class:flat>
	<button class="lmc-head" onclick={toggle} aria-expanded={open}>
		<div class="lmc-top">
			{#if live}
				<span class="status live-pill"><span class="dot" aria-hidden="true"></span>{language.text('Live', 'Live', 'Live')}{#if liveMin}<span class="min">{liveMin}</span>{/if}</span>
			{:else if played}
				<span class="status done">{language.text('Slutt', 'Slutt', 'Full time')}</span>
			{/if}
			<span class="meta">
				<span>{kickoffLabel(match.kickoff)} · {shortStage(match)}</span>
				{#if showTv && match.tvChannel}<TvLogo channel={match.tvChannel} compact />{/if}
			</span>
		</div>

		<div class="lmc-teams">
			<span class="team" class:dim={winnerSide === 'away'}>
				<Flag iso2={home?.iso2 ?? ''} code={home?.fifaCode ?? ''} size={22} />
				<b>{homeCode()}</b>
			</span>
			<span class="score" class:big={played || live} class:livescore={live} class:goal-flash={goalFlash}>
				{#if played || live}{scoreText(match)}{:else}<span class="vs">{language.text('mot', 'mot', 'vs')}</span>{/if}
			</span>
			<span class="team r" class:dim={winnerSide === 'home'}>
				<b>{awayCode()}</b>
				<Flag iso2={away?.iso2 ?? ''} code={away?.fifaCode ?? ''} size={22} />
			</span>
		</div>

		<div class="lmc-foot">
			<span class="mytip">
				{#if myTip}
					<span class="mt-label">{language.text('Ditt tips', 'Ditt tips', 'Your tip')}</span>
					<b class="digits">{myTip.ftHome}–{myTip.ftAway}</b>
					{#if played && myPts !== undefined}
						{#if myPts === 6}
							<span class="mpill perfect"><span class="star">★</span>6 p</span>
						{:else if myPts > 0}
							<span class="mpill ok">+{myPts} p</span>
						{:else}
							<span class="mpill zero">0 p</span>
						{/if}
					{/if}
				{:else}
					<span class="muted">{language.text('Ikke tipset', 'Ikkje tipsa', 'Not tipped')}</span>
				{/if}
			</span>
			<span class="hint">
				{#if locked}
					<span class="hint-txt">{language.text('tips', 'tips', 'tips')}</span>
				{:else}
					<Lock size={12} />
				{/if}
				<ChevronDown size={16} class="chev" />
			</span>
		</div>
	</button>

	{#if open}
		<div class="lmc-body" transition:slide={{ duration: 150 }}>
			{#if visibleEvents.length > 0}
				<div class="events">
					{#each visibleEvents as event (event.id || event.providerKey)}
						{@const evTeam = eventTeam(event, [home, away])}
						<span class="event" title={eventTitle(event, language.text('Assist', 'Assist', 'Assist'))}>
							<span class="emin">{eventMinute(event)}</span>
							<span class="eicon">{eventIcon(event)}</span>
							{#if evTeam}<Flag iso2={evTeam.iso2} code={evTeam.fifaCode} size={13} />{/if}
							<span class="eplayer">{event.player || event.detail}</span>
							{#if event.detail === 'Own Goal'}<span class="eog">{language.text('selvmål', 'sjølvmål', 'OG')}</span>{/if}
						</span>
					{/each}
				</div>
			{/if}

			{#if locked}
				{#if friendsBusy && friends === null}
					<p class="muted small loadrow">{language.text('Laster…', 'Lastar…', 'Loading…')}</p>
				{:else if friends}
					<MatchTipsTable tips={friends} {isKO} {avatars} />
				{/if}
			{:else}
				<p class="muted small lockrow">
					<Lock size={13} />
					{language.text(
						'Andre sine tips er skjult til avspark.',
						'Andre sine tips er skjult til avspark.',
						"Other players' tips are hidden until kickoff."
					)}
				</p>
			{/if}
		</div>
	{/if}
</div>

<style>
	.lmc {
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface);
		margin-bottom: 0.6rem;
		overflow: hidden;
		box-shadow: var(--shadow-tile);
		transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease;
	}
	.lmc:hover {
		transform: translateY(-1px);
		box-shadow: var(--shadow-pop);
	}
	.lmc.open {
		border-color: var(--border-strong);
	}
	.lmc.live {
		border-color: color-mix(in srgb, var(--live) 55%, var(--border));
	}
	/* Flat variant: an expandable row inside a parent card (e.g. Home "Latest
	   results"), so we drop the card's own border/shadow/padding. */
	.lmc.flat {
		border: none;
		border-radius: 0;
		background: none;
		box-shadow: none;
		margin-bottom: 0;
		border-bottom: 1px solid var(--border);
	}
	.lmc.flat:hover {
		transform: none;
		box-shadow: none;
	}
	.lmc.flat:last-child {
		border-bottom: none;
	}
	.lmc.flat .lmc-head {
		padding: 0.55rem 0;
	}
	.lmc.flat .lmc-body {
		padding: 0.1rem 0 0.65rem;
	}
	.lmc-head {
		display: block;
		width: 100%;
		text-align: left;
		background: none;
		border: none;
		padding: 0.65rem 0.85rem 0.55rem;
		cursor: pointer;
		color: var(--text);
	}
	.lmc-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
		margin-bottom: 0.45rem;
	}
	.status {
		font-size: 0.7rem;
		font-weight: 700;
	}
	.live-pill {
		display: inline-flex;
		align-items: center;
		gap: 0.32rem;
		color: var(--live);
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}
	.live-pill .dot {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		background: var(--live);
		animation: livePulse 1.5s ease-in-out infinite;
	}
	.live-pill .min {
		color: var(--muted);
		font-weight: 700;
	}
	.status.done {
		color: var(--muted);
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}
	.meta {
		margin-left: auto;
		display: inline-flex;
		align-items: center;
		justify-content: flex-end;
		gap: 0.45rem;
		font-size: 0.68rem;
		font-weight: 600;
		color: var(--muted-2);
		letter-spacing: 0.02em;
		text-align: right;
	}
	.meta :global(.tv-logo.compact) {
		width: 60px;
		height: 19px;
	}
	.lmc-teams {
		display: grid;
		grid-template-columns: 1fr auto 1fr;
		align-items: center;
		gap: 0.6rem;
	}
	.team {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		min-width: 0;
		transition: opacity 0.18s ease;
	}
	.team.r {
		justify-content: flex-end;
	}
	.team.dim {
		opacity: 0.5;
	}
	.team b {
		font-weight: 800;
		font-size: 1rem;
		letter-spacing: 0.01em;
	}
	.score {
		font-size: 1rem;
		font-weight: 700;
		text-align: center;
		white-space: nowrap;
		color: var(--text);
		font-variant-numeric: tabular-nums;
	}
	.score.big {
		font-size: 1.25rem;
		font-weight: 800;
	}
	.score.livescore {
		color: var(--live);
	}
	.score.goal-flash {
		animation: scoreFlash 1.1s ease-out;
	}
	.score .vs {
		font-size: 0.82rem;
		font-weight: 600;
		color: var(--muted-2);
	}
	.lmc-foot {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
		margin-top: 0.5rem;
		padding-top: 0.45rem;
		border-top: 1px solid var(--border);
		font-size: 0.78rem;
		color: var(--muted);
	}
	.mytip {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		min-width: 0;
	}
	.mt-label {
		color: var(--muted);
	}
	.mytip b {
		color: var(--text);
		font-weight: 800;
		font-variant-numeric: tabular-nums;
	}
	.mpill {
		display: inline-flex;
		align-items: center;
		gap: 0.12rem;
		font-weight: 800;
		font-size: 0.72rem;
		padding: 0.12rem 0.45rem;
		border-radius: var(--radius-pill);
		line-height: 1;
	}
	.mpill.ok {
		color: var(--success);
		background: color-mix(in srgb, var(--success) 16%, transparent);
	}
	.mpill.zero {
		color: var(--muted);
		background: var(--surface-2);
	}
	.mpill.perfect {
		color: #3a2a00;
		background: var(--gold);
	}
	:global(:root[data-theme='dark']) .mpill.perfect,
	:global(:root[data-theme='worldcup']) .mpill.perfect {
		color: #1a1200;
	}
	.mpill .star {
		font-size: 0.68rem;
	}
	.hint {
		display: inline-flex;
		align-items: center;
		gap: 0.22rem;
		flex: none;
		color: var(--muted-2);
		font-weight: 600;
	}
	.hint-txt {
		text-transform: uppercase;
		font-size: 0.66rem;
		letter-spacing: 0.04em;
	}
	.lmc.open :global(.chev) {
		transform: rotate(180deg);
		color: var(--text);
	}
	:global(.lmc .hint .chev) {
		transition: transform 0.2s ease, color 0.2s ease;
	}
	.lmc-body {
		padding: 0.1rem 0.85rem 0.75rem;
	}
	.loadrow,
	.lockrow {
		margin: 0.4rem 0 0.1rem;
	}
	.lockrow {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}
	.muted {
		color: var(--muted);
	}
	.small {
		font-size: 0.8rem;
	}
	.events {
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
		margin: 0.15rem 0 0.7rem;
		padding-bottom: 0.6rem;
		border-bottom: 1px solid var(--border);
	}
	.event {
		display: inline-flex;
		align-items: center;
		gap: 0.42rem;
		font-size: 0.8rem;
	}
	.emin {
		min-width: 2.4rem;
		color: var(--muted);
		font-variant-numeric: tabular-nums;
		font-weight: 700;
	}
	.eicon {
		width: 1.1rem;
		text-align: center;
	}
	.eplayer {
		color: var(--text);
	}
	.eog {
		font-size: 0.66rem;
		color: var(--muted);
		text-transform: uppercase;
	}
	@keyframes livePulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.4; }
	}
	@keyframes scoreFlash {
		0% {
			transform: scale(1);
		}
		18% {
			transform: scale(1.18);
			color: var(--accent);
		}
		55% {
			color: var(--accent);
		}
		100% {
			transform: scale(1);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.lmc,
		.lmc:hover {
			transition: none;
			transform: none;
		}
		.live-pill .dot,
		.score.goal-flash {
			animation: none;
		}
	}
</style>
