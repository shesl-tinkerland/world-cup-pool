<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { tipsStore, type Match, isLocked, teamsResolved } from '$lib/tips.svelte';
	import TipCard from '$lib/components/TipCard.svelte';
	import GroupStandings from '$lib/components/GroupStandings.svelte';
	import { collapseOnScroll } from '$lib/actions';
	import { serverClock } from '$lib/serverclock.svelte';
	import { searchNav } from '$lib/searchNav.svelte';
	import { stageName, stageOrder } from '$lib/stageLabels';
	import { bestThirds } from '$lib/standings';
	import { language } from '$lib/language.svelte';
	import { LocateFixed } from '@lucide/svelte';
	import { tick } from 'svelte';

	type Section = {
		id: string;
		label: string;
		matches: Match[];
	};
	type Tab = 'missing' | 'all' | 'group' | 'ko';

	let tab = $state<Tab>('missing');
	let userSelectedTab = $state(false);
	let jumpToNowAfterAutoTab = $state(false);
	let requestedTab = $derived.by<Tab | ''>(() => {
		const raw = ($page.url.searchParams.get('tab') ?? '').trim().toLowerCase();
		if (raw === 'missing' || raw === 'all' || raw === 'group' || raw === 'ko') {
			return raw;
		}
		return '';
	});
	let searchMatchId = $derived($page.url.searchParams.get('match') ?? '');
	let searchTeamId = $derived($page.url.searchParams.get('team') ?? '');
	let searchGroupId = $derived(
		($page.url.searchParams.get('group') ?? '').trim().toUpperCase()
	);
	let searchTargetId = $derived.by(() => {
		if (searchMatchId) return searchMatchId;
		if (!searchTeamId) return '';
		return (
			tipsStore.matches.find(
				(m) => m.homeTeam === searchTeamId || m.awayTeam === searchTeamId
			)?.id ?? ''
		);
	});
	let lastSearchJump = '';
	let projectedBestThirds = $derived.by(() => {
		const groups: Record<string, Match[]> = {};
		for (const match of tipsStore.matches) {
			if (match.stage !== 'group') continue;
			(groups[match.groupLetter] ||= []).push(match);
		}
		return bestThirds(Object.values(groups), tipsStore.tips);
	});

	async function selectTab(newTab: Tab) {
		userSelectedTab = true;
		if ($page.url.search) {
			await goto('/tips', { replaceState: true, noScroll: true });
		}
		tab = newTab;
	}

	$effect(() => {
		if (!tipsStore.loaded) tipsStore.load().catch(() => {});
	});
	$effect(() => {
		if (requestedTab) {
			if (tab !== requestedTab) {
				tab = requestedTab;
			}
			return;
		}
		if (searchGroupId && tab !== 'group') {
			tab = 'group';
			return;
		}
		if ((searchMatchId || searchTeamId) && tab !== 'all') tab = 'all';
	});

	let filtered = $derived(
		tipsStore.matches.filter((m) => {
			if (tab === 'missing')
				return teamsResolved(m) && !isLocked(m) && !tipsStore.tips[m.id];
			if (tab === 'group') return m.stage === 'group';
			if (tab === 'ko') return m.stage !== 'group';
			return true;
		})
	);
	let missingOpenMatches = $derived(
		tipsStore.matches.filter(
			(m) => teamsResolved(m) && !isLocked(m) && !tipsStore.tips[m.id]
		)
	);
	let nextMissingMatch = $derived(missingOpenMatches[0]);

	$effect(() => {
		if (
			!tipsStore.loaded ||
			userSelectedTab ||
			requestedTab ||
			searchMatchId ||
			searchTeamId ||
			searchGroupId ||
			tab !== 'missing' ||
			missingOpenMatches.length > 0 ||
			tipsStore.matches.length === 0
		) return;

		tab = 'all';
		jumpToNowAfterAutoTab = true;
	});

	function deadlineLabel(iso: string) {
		return new Date(iso).toLocaleString(language.locale, {
			weekday: 'short',
			day: 'numeric',
			month: 'short',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	// "Now" = the next match not yet kicked off (or the last one if the
	// tournament is over) within the current filter.
	let nowId = $derived.by(() => {
		const now = serverClock.now();
		const next = filtered.find(
			(m) => new Date(m.kickoff).getTime() >= now
		);
		return (next ?? filtered[filtered.length - 1])?.id ?? '';
	});
	let firstUpcomingId = $derived.by(() => {
		const now = serverClock.now();
		return filtered.find((m) => new Date(m.kickoff).getTime() >= now)?.id ?? '';
	});
	let showAllDivider = $derived(
		tab === 'all' && !!firstUpcomingId && filtered[0]?.id !== firstUpcomingId
	);

	function goNow() {
		// Scroll to the day-header of the day holding the "now" match —
		// nicer context, and days hold only a handful of games.
		const targetId = nowDayIndex >= 0 ? sections[nowDayIndex]?.id : '';
		if (!targetId) return;
		document.getElementById(targetId)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
	}

	// Groups tab: by group letter (A..L). Knockout tab: by stage (R32→FINAL).
	// All tab: by calendar day.
	function sectionId(label: string) {
		return label
			.toLocaleLowerCase('no-NO')
			.normalize('NFD')
			.replace(/[\u0300-\u036f]/g, '')
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-+|-+$/g, '');
	}

	let sections = $derived.by<Section[]>(() => {
		const byKickoff = (a: Match, b: Match) =>
			new Date(a.kickoff).getTime() - new Date(b.kickoff).getTime();
		if (tab === 'group') {
			const byGroup: Record<string, Match[]> = {};
			for (const m of filtered) (byGroup[m.groupLetter] ||= []).push(m);
			return Object.keys(byGroup)
				.sort()
				.map((letter) => ({
					id: `section-group-${letter}`,
					label: `${language.text('Gruppe', 'Gruppe', 'Group')} ${letter}`,
					matches: byGroup[letter].sort(byKickoff)
				}));
		}
		if (tab === 'ko') {
			const byStage: Record<string, Match[]> = {};
			for (const m of filtered) (byStage[m.stage] ||= []).push(m);
			return stageOrder
				.filter((s) => byStage[s])
				.map((stage) => ({
					id: `section-stage-${sectionId(stage)}`,
					label: stageName(stage),
					matches: byStage[stage].sort(byKickoff)
				}));
		}
		return Object.entries(
			filtered.reduce<Record<string, Match[]>>((acc, m) => {
				const d = new Date(m.kickoff).toLocaleDateString(language.locale, {
					weekday: 'long',
					day: 'numeric',
					month: 'long'
				});
				(acc[d] ||= []).push(m);
				return acc;
			}, {})
		).map(([label, matches], index) => ({
			id: `section-day-${index}-${sectionId(label)}`,
			label,
			matches
		}));
	});

	let nowDayIndex = $derived(
		sections.findIndex((section) => section.matches.some((m) => m.id === nowId))
	);
	let searchGroupSectionId = $derived(
		searchGroupId ? `section-group-${searchGroupId}` : ''
	);

	async function scrollToElementId(
		elementId: string,
		block: ScrollLogicalPosition = 'start'
	) {
		for (let attempt = 0; attempt < 24; attempt += 1) {
			await tick();
			const element = document.getElementById(elementId);
			if (element) {
				element.scrollIntoView({ behavior: 'smooth', block });
				return true;
			}
			await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
		}
		return false;
	}

	// On first load, instantly jump to the current point in the tournament.
	let didAutoScroll = false;
	$effect(() => {
		if (searchMatchId || searchTeamId || searchGroupId) return;
		const forcedNowJump = jumpToNowAfterAutoTab && tab === 'all';
		if ((didAutoScroll && !forcedNowJump) || !tipsStore.loaded) return;
		const idx = nowDayIndex;
		if (idx < 0) return;
		if (forcedNowJump) jumpToNowAfterAutoTab = false;
		didAutoScroll = true;
		// First matchday: stay at the very top (full header). Otherwise jump
		// to that day's header. If we auto-left an empty Missing tab, still
		// run the same jump users get from the Now button.
		if (idx === 0 && !forcedNowJump) return;
		void scrollToElementId(sections[idx].id);
	});

	$effect(() => {
		const target = searchTargetId;
		const key = `${searchNav.token}:${tab}:${searchMatchId}:${searchTeamId}:${target}`;
		const ready = sections.some((section) => section.matches.some((match) => match.id === target));
		if (
			!tipsStore.loaded ||
			(tab !== 'all' && tab !== 'missing') ||
			!target ||
			!ready ||
			key === lastSearchJump
		) return;
		let cancelled = false;
		void (async () => {
			const scrolled = await scrollToElementId(`match-${target}`);
			if (!cancelled && scrolled) {
				lastSearchJump = key;
			}
		})();
		return () => {
			cancelled = true;
		};
	});

	$effect(() => {
		const target = searchGroupSectionId;
		const key = `${searchNav.token}:${tab}:${searchGroupId}:${target}`;
		const ready = sections.some((section) => section.id === target);
		if (!tipsStore.loaded || tab !== 'group' || !target || !ready || key === lastSearchJump) return;
		let cancelled = false;
		void (async () => {
			const scrolled = await scrollToElementId(target);
			if (!cancelled && scrolled) {
				lastSearchJump = key;
			}
		})();
		return () => {
			cancelled = true;
		};
	});
</script>

<div class="stickyhead" use:collapseOnScroll>
	<p class="kicker">{language.text('Kamptips', 'Kamptips', 'Match tips')}</p>
	<div class="sh-expand">
		<div class="sh-inner">
			<h1>{language.text('Kamptips', 'Kamptips', 'Match tips')}</h1>
			<p class="muted desc">
				{language.text(
					'Tipp resultatet for hver kamp. Du kan endre fram til avspark.',
					'Tipp resultatet for kvar kamp. Du kan endre fram til avspark.',
					'Pick the result for every match. You can change it until kickoff.'
				)}
			</p>
			{#if tipsStore.loaded && missingOpenMatches.length > 0}
				<p class="muted statusline">
					{language.text(
						`${missingOpenMatches.length} ${missingOpenMatches.length === 1 ? 'åpen kamp' : 'åpne kamper'} mangler · neste frist ${deadlineLabel(nextMissingMatch.kickoff)}`,
						`${missingOpenMatches.length} ${missingOpenMatches.length === 1 ? 'open kamp' : 'opne kampar'} manglar · neste frist ${deadlineLabel(nextMissingMatch.kickoff)}`,
						`${missingOpenMatches.length} open match${missingOpenMatches.length === 1 ? '' : 'es'} missing · next deadline ${deadlineLabel(nextMissingMatch.kickoff)}`
					)}
				</p>
			{/if}
		</div>
	</div>
	<div class="tabs">
		<button class:active={tab === 'missing'} onclick={() => selectTab('missing')}
			>{language.text('Mangler', 'Manglar', 'Missing')}</button
		>
		<button class:active={tab === 'all'} onclick={() => selectTab('all')}>{language.text('Alle', 'Alle', 'All')}</button>
		<button class:active={tab === 'group'} onclick={() => selectTab('group')}
			>{language.text('Grupper', 'Grupper', 'Groups')}</button
		>
		<button class:active={tab === 'ko'} onclick={() => selectTab('ko')}
			>{language.text('Sluttspill', 'Sluttspel', 'Knockout')}</button
		>
	</div>
</div>

{#if !tipsStore.loaded}
		<p class="muted">{language.text('Laster kamper...', 'Lastar kampar…', 'Loading matches…')}</p>
{:else if filtered.length === 0}
	<div class="card empty" style="text-align: center; padding: 2.5rem 1rem;">
		<span style="display:block; margin-bottom:1rem; opacity:0.8; color:var(--muted);">
			{#if tab === 'missing'}
				<img
					src="/icons/football-alert.svg"
					alt=""
					style="width:52px;height:52px;display:block;margin:0 auto;"
				/>
			{:else}
				<span style="font-size: 3rem;">⚽</span>
			{/if}
		</span>
		<h3>
			{tab === 'missing'
				? language.text('Ingen kamptips mangler', 'Ingen kamptips manglar', 'No missing match tips')
				: language.text('Ingenting her.', 'Ingenting her.', 'Nothing here.')}
		</h3>
		<p class="muted">
			{tab === 'missing'
				? language.text('Alle åpne kamper som kan tippes nå, er fylt inn.', 'Alle opne kampar som kan tippast no, er fylt inn.', 'All open matches that can be tipped right now are filled in.')
				: language.text('Prøv en annen fane.', 'Prøv ei anna fane.', 'Try another tab.')}
		</p>
		{#if tab === 'missing'}
			<div class="empty-actions">
				<button class="empty-link" onclick={() => selectTab('all')}>
					{language.text('Se alle kampene', 'Sjå alle kampane', 'View all matches')}
				</button>
			</div>
		{/if}
	</div>
{:else}
	{#each sections as section (section.id)}
		<h3 class="day" class:spotlight={section.id === searchGroupSectionId} id={section.id}>
			{section.label}
		</h3>
		{#each section.matches as m (m.id)}
			{#if showAllDivider && m.id === firstUpcomingId}
				<div
					class="now-divider-wrap"
					role="separator"
					aria-label={language.text('Her er vi nå', 'Her er vi no', 'Where the tournament is now')}
				>
					<div class="now-divider">
						<span class="line"></span>
						<span class="badge"><LocateFixed size={14} /> {language.text('Her er vi nå', 'Her er vi no', 'Where we are now')}</span>
						<span class="line"></span>
					</div>
					<p class="now-hint">{language.text('Kommende kamper under', 'Komande kampar under', 'Upcoming matches below')}</p>
				</div>
			{/if}
			<div
				class="match"
				class:spotlight={m.id === searchTargetId ||
					(!!searchTeamId && (m.homeTeam === searchTeamId || m.awayTeam === searchTeamId))}
				id={`match-${m.id}`}
			>
				<TipCard match={m} />
			</div>
		{/each}
		{#if tab === 'group'}
			<GroupStandings matches={section.matches} bestThirds={projectedBestThirds} />
		{/if}
	{/each}
	<div class="fabpad"></div>
{/if}

{#if tipsStore.loaded && nowId}
	<button class="fab" onclick={goNow} aria-label={language.text('Rull til neste kamp', 'Rull til neste kamp', 'Scroll to next match')}>
		<LocateFixed size={18} /> {language.text('Nå', 'No', 'Now')}
	</button>
{/if}

<style>
	.stickyhead {
		position: sticky;
		top: var(--topbar-h);
		z-index: 20;
		margin: 0 -1rem;
		padding: 0.6rem 1rem 0.75rem;
		background: var(--bg);
		border-bottom: 1px solid var(--border);
	}
	.stickyhead h1 {
		margin: 0.1rem 0 0;
	}
	.stickyhead .desc {
		margin: 0.3rem 0 0;
		font-size: 0.9rem;
	}
	.statusline {
		margin: 0.45rem 0 0;
		font-size: 0.82rem;
		font-weight: 650;
	}
	@media (min-width: 900px) {
		.stickyhead {
			top: 0;
			margin: 0 -2rem;
			padding: 0.75rem 2rem 0.85rem;
		}
	}
	.day {
		margin: 1.3rem 0 0.6rem;
		font-size: 0.95rem;
		color: var(--muted);
		/* Land below the fixed top bar + collapsed sticky header. */
		scroll-margin-top: 150px;
	}
	.now-divider-wrap {
		margin: 1rem 0 0.8rem;
	}
	.now-divider {
		display: flex;
		align-items: center;
		gap: 0.7rem;
	}
	.now-divider .line {
		flex: 1;
		height: 1px;
		background: color-mix(in srgb, var(--accent) 24%, var(--border));
	}
	.now-divider .badge {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.38rem 0.8rem;
		border: 1px solid color-mix(in srgb, var(--accent) 34%, var(--border));
		border-radius: var(--radius-pill);
		background: color-mix(in srgb, var(--accent) 12%, transparent);
		color: var(--text);
		font:
			700 0.74rem var(--font);
		letter-spacing: 0.05em;
		text-transform: uppercase;
		white-space: nowrap;
	}
	.now-hint {
		margin: 0.35rem 0 0;
		text-align: center;
		color: var(--muted);
		font-size: 0.78rem;
	}
	.day.spotlight {
		color: var(--text);
		text-decoration: underline;
		text-decoration-thickness: 2px;
		text-underline-offset: 0.2em;
	}
	@media (min-width: 900px) {
		.day {
			scroll-margin-top: 96px;
		}
	}
	.match + .match {
		margin-top: 6px;
	}
	.match {
		scroll-margin-top: 158px;
		border-radius: var(--radius);
	}
	.match.spotlight {
		outline: 2px solid color-mix(in srgb, var(--accent) 54%, transparent);
		outline-offset: 3px;
	}
	@media (min-width: 900px) {
		.match {
			scroll-margin-top: 104px;
		}
	}
	.empty {
		display: grid;
		gap: 0.35rem;
	}
	.empty-actions {
		margin-top: 0.8rem;
	}
	.empty-link {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		padding: 0.7rem 1rem;
		border: 1px solid color-mix(in srgb, var(--accent) 36%, var(--border));
		border-radius: var(--radius-pill);
		background: color-mix(in srgb, var(--accent) 12%, transparent);
		color: var(--text);
		font:
			700 0.78rem var(--font);
		letter-spacing: 0.05em;
		text-transform: uppercase;
		cursor: pointer;
		transition:
			transform 0.12s ease,
			background 0.2s ease,
			border-color 0.2s ease;
	}
	.empty-link:hover {
		transform: translateY(-1px);
		background: color-mix(in srgb, var(--accent) 18%, transparent);
		border-color: color-mix(in srgb, var(--accent) 48%, var(--border));
	}
	.empty h3,
	.empty p {
		margin: 0;
	}
	.fabpad {
		height: 4rem;
	}
	.fab {
		position: fixed;
		right: 1rem;
		bottom: calc(var(--nav-h) + env(safe-area-inset-bottom) + 1.75rem);
		z-index: 40;
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.7rem 1rem;
		border: 1px solid var(--text);
		border-radius: var(--radius-pill);
		background: var(--text);
		color: var(--bg);
		font:
			800 0.8rem var(--font);
		letter-spacing: 0.06em;
		text-transform: uppercase;
		cursor: pointer;
		box-shadow: var(--shadow-pop);
		transition:
			transform 0.12s ease,
			box-shadow 0.2s ease;
	}
	.fab:hover {
		transform: translateY(-2px);
		box-shadow: var(--glow);
	}
	@media (min-width: 900px) {
		.fab {
			bottom: 1.5rem;
			right: 1.5rem;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.empty-link {
			transition: none;
		}
		.fab {
			transition: none;
		}
	}
</style>
