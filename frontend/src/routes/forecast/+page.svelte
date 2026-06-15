<script lang="ts">
	import DeadlineCountdown from '$lib/components/DeadlineCountdown.svelte';
	import { api, type GoldenBootSearchResult } from '$lib/api';
	import { forecastStore as fs, koKey, type GoldenBootPlayer, type KOMatch } from '$lib/forecast.svelte';
	import Flag from '$lib/components/Flag.svelte';
	import { vibrate } from '$lib/haptics';
	import { flip } from 'svelte/animate';
	import { teamDisplayName } from '$lib/teamNames';
	import {
		ChevronUp,
		ChevronDown,
		Lock,
		Check,
		CircleCheck,
		X,
		Trophy
	} from '@lucide/svelte';
	import { collapseOnScroll } from '$lib/actions';
	import { language } from '$lib/language.svelte';
	import { stageName as knockoutStageName } from '$lib/stageLabels';

	let section = $state<'groups' | 'thirds' | 'bracket' | 'goldenboot'>('groups');
	let saveState = $state<'idle' | 'saving' | 'saved' | 'error'>('idle');
	let err = $state('');
	$effect(() => {
		if (!fs.loaded) fs.load().catch((e) => (err = e?.message ?? language.text('Lasting feilet', 'Lasting feila', 'Load failed')));
	});

	// Debounced autosave. The Forecast is a living prediction edited until
	// lock, so changes persist automatically ~1s after the last edit.
	let primed = false;
	let timer: ReturnType<typeof setTimeout>;
	$effect(() => {
		// Track every part of the prediction.
		const snapshot = JSON.stringify([
			fs.groupOrder,
			fs.thirds,
			fs.bracket,
			fs.goldenBootPlayer
		]);
		if (!fs.loaded || fs.locked) return;
		if (!primed) {
			primed = true; // skip the initial hydrate
			return;
		}
		void snapshot;
		clearTimeout(timer);
		timer = setTimeout(async () => {
			saveState = 'saving';
			err = '';
			try {
				await fs.save();
				saveState = 'saved';
			} catch (e: unknown) {
				saveState = 'error';
				err =
					(e as { message?: string })?.message ??
					language.text(
						'Kunne ikke lagre - endringene ble ikke lagret.',
						'Kunne ikkje lagre — endringane er ikkje lagra.',
						'Could not save — your changes were not saved.'
					);
			}
		}, 1000);
		return () => clearTimeout(timer);
	});

	const stages = ['R32', 'R16', 'QF', 'SF', '3RD', 'FINAL'];
	let byStage = $derived(
		stages.map((s) => ({
			stage: s,
			matches: fs.knockout.filter((m) => m.stage === s)
		}))
	);

	let finalMatch = $derived(fs.knockout.find((m) => m.stage === 'FINAL'));
	let champion = $derived(
		finalMatch ? fs.bracket[koKey(finalMatch)] : ''
	);
	let actualThirds = $derived(fs.actualBestThirds());
	let goldenBootById = $derived.by(() => {
		const out: Record<string, GoldenBootPlayer> = {};
		for (const player of [...fs.goldenBoot.shortlist, ...fs.goldenBoot.leaders]) out[player.id] = player;
		return out;
	});
	let goldenBootPick = $derived(goldenBootById[fs.goldenBootPlayer]);
	let goldenBootLeaders = $derived(
		fs.goldenBoot.leaders.length > 0 ? fs.goldenBoot.leaders : fs.goldenBoot.shortlist.slice(0, 10)
	);
	let goldenBootSearchQuery = $state('');
	let goldenBootSearchResults = $state<GoldenBootSearchResult[]>([]);
	let goldenBootSearchLoading = $state(false);
	let goldenBootSearchError = $state('');
	let goldenBootSearchPendingKey = $state('');
	let goldenBootSearchApiAvailable = $state(true);
	let goldenBootSearchTimer: ReturnType<typeof setTimeout>;

	$effect(() => {
		const query = goldenBootSearchQuery.trim();
		void section;
		if (section !== 'goldenboot' || fs.locked) {
			goldenBootSearchLoading = false;
			goldenBootSearchError = '';
			return;
		}
		if (query.length < 2) {
			goldenBootSearchResults = [];
			goldenBootSearchLoading = false;
			goldenBootSearchError = '';
			return;
		}

		let cancelled = false;
		clearTimeout(goldenBootSearchTimer);
		goldenBootSearchTimer = setTimeout(async () => {
			goldenBootSearchLoading = true;
			goldenBootSearchError = '';
			try {
				const response = await api.searchGoldenBootPlayers(query);
				if (cancelled) return;
				goldenBootSearchResults = response.players;
				goldenBootSearchApiAvailable = response.apiAvailable;
			} catch (e: unknown) {
				if (cancelled) return;
				goldenBootSearchResults = [];
				goldenBootSearchError =
					(e as { message?: string })?.message ??
					language.text('Kunne ikke søke etter spillere.', 'Kunne ikkje søkje etter spelarar.', 'Could not search players.');
			} finally {
				if (!cancelled) goldenBootSearchLoading = false;
			}
		}, 280);

		return () => {
			cancelled = true;
			clearTimeout(goldenBootSearchTimer);
		};
	});

	function tname(id: string) {
		return teamDisplayName(fs.team(id));
	}
	function initials(name: string) {
		return name
			.split(/\s+/)
			.filter(Boolean)
			.slice(0, 2)
			.map((part) => part[0]?.toUpperCase() ?? '')
			.join('');
	}
	function updatedAt(iso?: string) {
		if (!iso) return language.text('Ikke synket ennå', 'Ikkje synka enno', 'Not synced yet');
		const date = new Date(iso);
		if (!Number.isFinite(date.getTime())) return '';
		return new Intl.DateTimeFormat(language.locale, {
			day: '2-digit',
			month: 'short',
			hour: '2-digit',
			minute: '2-digit'
		}).format(date);
	}
	function pickGoldenBoot(playerId: string) {
		if (fs.locked) return;
		fs.goldenBootPlayer = playerId;
		vibrate(15);
	}
	function sortGoldenBootPlayers(players: GoldenBootPlayer[]) {
		return [...players].sort((first, second) => {
			const firstRank = first.rank ?? 0;
			const secondRank = second.rank ?? 0;
			if ((firstRank === 0) !== (secondRank === 0)) return firstRank !== 0 ? -1 : 1;
			if (firstRank !== 0 && secondRank !== 0 && firstRank !== secondRank) return firstRank - secondRank;
			if (first.goals !== second.goals) return second.goals - first.goals;
			return first.name.localeCompare(second.name);
		});
	}
	function searchResultToPlayer(player: GoldenBootSearchResult): GoldenBootPlayer {
		return {
			id: player.id ?? '',
			name: player.name,
			teamId: player.teamId,
			teamName: player.teamName,
			photoUrl: player.photoUrl,
			goals: player.goals,
			assists: player.assists,
			rank: player.rank,
			eligible: player.eligible,
			seeded: false,
			syncedAt: fs.goldenBoot.updatedAt
		};
	}
	function upsertGoldenBootCandidate(player: GoldenBootPlayer) {
		const shortlist = sortGoldenBootPlayers([
			...fs.goldenBoot.shortlist.filter((current) => current.id !== player.id),
			player
		]);
		const shouldShowLeader = player.rank > 0 || fs.goldenBoot.leaders.some((current) => current.id === player.id);
		const leaders = shouldShowLeader
			? sortGoldenBootPlayers([
					...fs.goldenBoot.leaders.filter((current) => current.id !== player.id),
					player
				]).slice(0, 10)
			: fs.goldenBoot.leaders;
		fs.goldenBoot = {
			...fs.goldenBoot,
			shortlist,
			leaders,
			updatedAt: player.syncedAt || fs.goldenBoot.updatedAt
		};
	}
	async function chooseGoldenBootSearch(player: GoldenBootSearchResult) {
		if (fs.locked) return;
		goldenBootSearchPendingKey = player.key;
		goldenBootSearchError = '';
		try {
			const chosen = player.id && player.eligible
				? searchResultToPlayer(player)
				: (await api.ensureGoldenBootPlayer(player)).player;
			upsertGoldenBootCandidate(chosen);
			pickGoldenBoot(chosen.id);
			goldenBootSearchQuery = '';
			goldenBootSearchResults = [];
		} catch (e: unknown) {
			goldenBootSearchError =
				(e as { message?: string })?.message ??
				language.text('Kunne ikke legge til denne spilleren.', 'Kunne ikkje leggje til denne spelaren.', 'Could not add this player.');
		} finally {
			goldenBootSearchPendingKey = '';
		}
	}
	const ord = (n: number) =>
		n === 1 ? '1.' : n === 2 ? '2.' : n === 3 ? '3.' : `${n}.`;

	function sideLabel(m: KOMatch, side: 'home' | 'away') {
		const [h, a] = fs.sides(m);
		const id = side === 'home' ? h : a;
		if (id) return { id, name: tname(id), team: fs.team(id) };
		return {
			id: '',
			name: side === 'home' ? m.homeLabel : m.awayLabel,
			team: undefined
		};
	}
</script>

<div class="stickyhead" use:collapseOnScroll>
	<p class="kicker">{language.text('Hele turneringen', 'Heile turneringa', 'Whole tournament')}</p>
	<div class="sh-expand">
		<div class="sh-inner">
			<h1>{language.text('VM-tips', 'VM-tips', 'World Cup tips')}</h1>
			<p class="muted desc">
				{language.text(
					'VM-tipset ditt for grupper, beste treere og veien til finalen.',
					'VM-tipset ditt for grupper, beste trearar og vegen til finalen.',
					'Your World Cup tip for groups, best thirds, and the road to the final.'
				)}
				{#if fs.locked}<b>{language.text('Låst.', 'Låst.', 'Locked.')}</b
						>{:else}{language.text('Låses ved avspark.', 'Låsast ved avspark.', 'Locks at kickoff.')}{/if}
			</p>
				{#if !fs.locked && fs.tournamentStart}
					<DeadlineCountdown
						deadline={fs.tournamentStart}
						label={language.text('Låses', 'Låsast', 'Locks')}
						compact
					/>
				{/if}
		</div>
	</div>
	{#if fs.loaded}
		<div class="seg">
			<button class:on={section === 'groups'} onclick={() => (section = 'groups')}>{language.text('Grupper', 'Grupper', 'Groups')}</button>
			<button class:on={section === 'thirds'} onclick={() => (section = 'thirds')}>{language.text('Beste treere', 'Beste trearar', 'Best thirds')}</button>
			<button class:on={section === 'bracket'} onclick={() => (section = 'bracket')}>{language.text('Sluttspill', 'Sluttspel', 'Knockout')}</button>
			<button class:on={section === 'goldenboot'} onclick={() => (section = 'goldenboot')}>{language.text('Toppscorer', 'Toppscorar', 'Golden Boot')}</button>
		</div>
	{/if}
</div>

{#if err}<p class="error">{err}</p>{/if}

{#if !fs.loaded}
	<p class="muted">{language.text('Laster...', 'Lastar…', 'Loading…')}</p>
{:else}
	{#if fs.locked}
		<div class="card lockbar"><Lock size={16} /> {language.text('Turneringen har startet - VM-tipset er låst.', 'Turneringa har starta - VM-tipset er endeleg.', 'The tournament has started - the World Cup tip is final.')}</div>
	{/if}

	{#if section === 'groups'}
		<p class="muted small">
			{language.text(
				'Ranger hver gruppe fra 1. til 4. plass. Topp 2 går videre; 3.-plassen kan gå videre som beste treer.',
				'Ranger kvar gruppe frå 1. til 4. plass. Topp 2 går vidare; 3.-plassen kan gå vidare som beste trear.',
				'Rank each group from 1st to 4th. The top 2 advance; 3rd place can advance as a best third.'
			)}
		</p>
		{#each fs.groups as g (g.letter)}
			<section class="card grp">
				<h3>{language.text('Gruppe', 'Gruppe', 'Group')} {g.letter}</h3>
				{#each fs.groupOrder[g.letter] as id, i (id)}
					{@const ao = fs.actualOrder(g.letter)}
					{@const apos = ao ? ao.indexOf(id) + 1 : 0}
					{@const exact = ao ? ao[i] === id : null}
					{@const advanced =
						ao &&
						(apos <= 2 ||
							(apos === 3 && (actualThirds?.has(id) ?? false)))}
					{@const scoredAdv =
						advanced && (i < 2 || (i === 2 && !!fs.thirds[g.letter]))}
					{@const state =
						exact === null
							? 'pending'
							: exact
								? 'ok'
								: scoredAdv
									? 'half'
									: 'miss'}
					<div
						class="trow"
						class:rwin={state === 'ok'}
						class:rhalf={state === 'half'}
						class:rmiss={state === 'miss'}
						animate:flip={{ duration: 240 }}
					>
						<span class="pos">{i + 1}</span>
						<Flag iso2={fs.team(id)?.iso2 ?? ''} code={fs.team(id)?.fifaCode ?? ''} />
						<span class="nm">{tname(id)}</span>
						<span class="tag">
							{#if state === 'ok'}<span class="ind ok"><Check size={15} /></span>
							{:else if state === 'half'}
								<span class="apos half">{language.text('faktisk', 'faktisk', 'actual')} {ord(apos)}</span>
								<span class="ind half"><CircleCheck size={15} /></span>
							{:else if state === 'miss'}
								<span class="apos">{language.text('faktisk', 'faktisk', 'actual')} {ord(apos)}</span>
								<span class="ind no"><X size={15} /></span>
								{:else if i < 2}<span class="pill ok">{language.text('videre', 'vidare', 'through')}</span>
								{:else if i === 2}<span class="pill">{language.text('3.-plass', '3.-plass', '3rd place')}</span>{/if}
						</span>
						{#if !fs.locked}
							<span class="ord">
								<button aria-label={language.text('Flytt opp', 'Flytt opp', 'Move up')} disabled={i === 0} onclick={() => { fs.move(g.letter, i, -1); vibrate(15); }}><ChevronUp size={16} /></button>
								<button aria-label={language.text('Flytt ned', 'Flytt ned', 'Move down')} disabled={i === 3} onclick={() => { fs.move(g.letter, i, 1); vibrate(15); }}><ChevronDown size={16} /></button>
							</span>
						{/if}
					</div>
				{/each}
			</section>
		{/each}
	{:else if section === 'thirds'}
		<div class="thead">
			<p class="muted small">
				{language.text(
					'Velg de 8 av 12 gruppetreerne du tror går videre. Lagene kommer fra grupperangeringen din.',
					'Vel dei 8 av 12 gruppetrearane du trur går vidare. Laga kjem frå grupperangeringa di.',
					'Choose the 8 of 12 group thirds you think will advance. The teams come from your group rankings.'
				)}
			</p>
			<span class="cnt" class:full={fs.chosenThirdLetters.length === 8}>
				{fs.chosenThirdLetters.length} / 8
			</span>
		</div>
		<section class="card tlist">
			{#each fs.groups as g (g.letter)}
				{@const tid = fs.groupThird(g.letter)}
				{@const on = !!fs.thirds[g.letter]}
				{@const adv = actualThirds ? actualThirds.has(tid) : null}
				<label class="trow" class:on>
					<input
						type="checkbox"
						checked={on}
						disabled={fs.locked ||
							(!on && fs.chosenThirdLetters.length >= 8)}
						onchange={() => fs.toggleThird(g.letter)}
					/>
					<span class="gl">{g.letter}</span>
					<Flag iso2={fs.team(tid)?.iso2 ?? ''} code={fs.team(tid)?.fifaCode ?? ''} />
					<span class="nm">{tname(tid) || '—'}</span>
					<span class="spacer"></span>
					{#if on && adv === true}<span class="ind ok"><Check size={15} /></span>
					{:else if on && adv === false}<span class="ind no"><X size={15} /></span>
					{:else if adv === true}<span class="ind dim"><Check size={14} /></span>{/if}
				</label>
			{/each}
		</section>
	{:else if section === 'goldenboot'}
		<div class="gb-head">
			<p class="muted small">
				{language.text(
					'Velg toppscorer, eller søk opp en outsider. Rett tips gir 15 poeng.',
					'Vel toppscorar, eller søk opp ein outsider. Rett tips gir 15 poeng.',
					'Pick a player or search for an outsider. A correct pick gives 15 points.'
				)}
			</p>
			{#if fs.goldenBoot.updatedAt}
				<span class="cnt">{language.text('Oppdatert', 'Oppdatert', 'Updated')} {updatedAt(fs.goldenBoot.updatedAt)}</span>
			{/if}
		</div>

		{#if goldenBootPick}
			<section class="card gb-pick">
				<Trophy size={20} />
				<span class="headshot-wrap">
					{#if goldenBootPick.photoUrl}
						<img class="headshot" src={goldenBootPick.photoUrl} alt="" loading="lazy" />
					{:else}
						<span class="headshot fallback">{initials(goldenBootPick.name)}</span>
					{/if}
				</span>
				<span class="gb-main">
					<i>{language.text('Ditt toppscorertips', 'Ditt toppscorartips', 'Your Golden Boot pick')}</i>
					<b>{goldenBootPick.name}</b>
				</span>
				<Flag iso2={fs.team(goldenBootPick.teamId)?.iso2 ?? ''} code={fs.team(goldenBootPick.teamId)?.fifaCode ?? ''} />
			</section>
		{/if}

		<section class="card gb-search">
			<div class="gb-search-head">
				<div>
					<h3>{language.text('Søk etter flere spillere', 'Søk etter fleire spelarar', 'Search more players')}</h3>
					<p class="muted small">
						{language.text('Legg til en outsider hvis spilleren ikke er på listen.', 'Legg til ein outsider om dei ikkje er på lista.', 'Add an outsider if they are not in the shortlist.')}
					</p>
				</div>
				{#if !goldenBootSearchApiAvailable}
					<span class="muted small api-note">{language.text('Live API-søk er utilgjengelig', 'Live API-søk er utilgjengeleg', 'Live API search unavailable')}</span>
				{/if}
			</div>
			<input
				class="gb-search-input"
				type="search"
				bind:value={goldenBootSearchQuery}
				placeholder={language.text('Søk på spillernavn...', 'Søk på spelarnamn…', 'Search by player name…')}
				aria-label={language.text('Søk etter toppscorerspillere', 'Søk etter toppscorarspelarar', 'Search for a Golden Boot player')}
				disabled={fs.locked}
			/>

			{#if goldenBootSearchError}
				<p class="error small">{goldenBootSearchError}</p>
			{:else if goldenBootSearchQuery.trim().length < 2}
				<p class="muted small">
					{language.text('Søk etter spillere...', 'Søk etter spelarar...', 'Type to search...')}
				</p>
			{:else if goldenBootSearchLoading}
				<p class="muted small">{language.text('Søker...', 'Søkjer…', 'Searching…')}</p>
			{:else if goldenBootSearchResults.length === 0}
				<p class="muted small">
					{goldenBootSearchApiAvailable
						? language.text('Fant ingen VM-kandidater som passet søket.', 'Fann ingen VM-kandidatar som passa søket.', 'No matching World Cup candidates found.')
						: language.text('Fant ingen lokale kandidater, og live API-søk er utilgjengelig.', 'Fann ingen lokale kandidatar, og live API-søk er utilgjengeleg.', 'No local candidates matched, and live API search is unavailable.')}
				</p>
			{:else}
				<div class="gb-search-results">
					{#each goldenBootSearchResults as player (player.key)}
						<button
							class="gb-search-result"
							class:picked={fs.goldenBootPlayer === (player.id ?? '')}
							disabled={fs.locked || (goldenBootSearchPendingKey !== '' && goldenBootSearchPendingKey !== player.key)}
							onclick={() => chooseGoldenBootSearch(player)}
						>
							<span class="headshot-wrap">
								{#if player.photoUrl}
									<img class="headshot" src={player.photoUrl} alt="" loading="lazy" />
								{:else}
									<span class="headshot fallback">{initials(player.name)}</span>
								{/if}
							</span>
							<span class="gb-main">
								<b>{player.name}</b>
								<span class="gb-search-meta">
									<Flag iso2={fs.team(player.teamId)?.iso2 ?? ''} code={fs.team(player.teamId)?.fifaCode ?? ''} />
									<span>{player.teamName}</span>
									<span>·</span>
									<span>{language.text('Mål', 'Mål', 'Goals')} {player.goals}</span>
								</span>
							</span>
							<span class="gb-search-action">
								{#if goldenBootSearchPendingKey === player.key}
									{language.text('Legger til...', 'Legg til…', 'Adding…')}
								{:else if player.id && player.eligible}
									{language.text('Velg', 'Vel', 'Pick')}
								{:else if player.existing}
									{language.text('Legg til i listen', 'Legg til i lista', 'Add to list')}
								{:else}
									{language.text('Legg til spiller', 'Legg til spelar', 'Add player')}
								{/if}
							</span>
						</button>
					{/each}
				</div>
			{/if}
		</section>

		<section class="card gb-list">
			<h3>{language.text('Kandidater', 'Kandidatar', 'Shortlist')}</h3>
			{#if fs.goldenBoot.shortlist.filter(p => p.seeded).length === 0}
				<p class="muted small">{language.text('Ingen kandidater ennå.', 'Ingen kandidatar enno.', 'No candidates yet.')}</p>
			{:else}
				<div class="gb-grid">
					{#each fs.goldenBoot.shortlist.filter(p => p.seeded) as player (player.id)}
						<button
							class="gb-player"
							class:picked={fs.goldenBootPlayer === player.id}
							disabled={fs.locked}
							onclick={() => pickGoldenBoot(player.id)}
						>
							<span class="headshot-wrap">
								{#if player.photoUrl}
									<img class="headshot" src={player.photoUrl} alt="" loading="lazy" />
								{:else}
									<span class="headshot fallback">{initials(player.name)}</span>
								{/if}
							</span>
							<span class="gb-main">
								<b>{player.name}</b>
								<span class="gb-search-meta" style="margin-top: 0.15rem;">
									<Flag iso2={fs.team(player.teamId)?.iso2 ?? ''} code={fs.team(player.teamId)?.fifaCode ?? ''} />
									<span>{player.teamName}</span>
								</span>
							</span>
							{#if fs.goldenBootPlayer === player.id}
								<span class="gb-player-status"><Check size={17} /></span>
							{/if}
						</button>
					{/each}
				</div>
			{/if}
		</section>

		<section class="card gb-live">
			<div class="gb-live-head">
				<h3>{language.text('Toppscorere', 'Toppscorarar', 'Top scorers')}</h3>
				{#if fs.goldenBoot.updatedAt}
					<p class="muted small gb-updated">{language.text('Oppdatert', 'Oppdatert', 'Updated')} {updatedAt(fs.goldenBoot.updatedAt)}</p>
				{/if}
			</div>
			<table class="gb-table">
				<thead>
					<tr>
						<th>#</th>
						<th>{language.text('Spiller', 'Spelar', 'Player')}</th>
						<th>{language.text('Lag', 'Lag', 'Team')}</th>
						<th class="num">{language.text('Mål', 'Mål', 'Goals')}</th>
					</tr>
				</thead>
				<tbody>
					{#each goldenBootLeaders as player (player.id)}
						<tr class:picked={fs.goldenBootPlayer === player.id}>
							<td>{player.rank || '–'}</td>
							<td>
								<span class="gb-row-player">
									{#if player.photoUrl}<img class="mini-headshot" src={player.photoUrl} alt="" loading="lazy" />{:else}<span class="mini-headshot fallback">{initials(player.name)}</span>{/if}
									<b>{player.name}</b>
								</span>
							</td>
							<td>
								<span class="gb-team">
									<Flag iso2={fs.team(player.teamId)?.iso2 ?? ''} code={fs.team(player.teamId)?.fifaCode ?? ''} />
									<span class="tm-full">{player.teamName}</span>
									<span class="tm-short">{fs.team(player.teamId)?.fifaCode ?? player.teamName}</span>
								</span>
							</td>
							<td class="num digits">{player.goals}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</section>
	{:else}
		{#if champion}
			<div class="card champ">
				<Trophy size={20} />
				<span class="lbl">{language.text('Tippet vinner', 'Tippa vinnar', 'Predicted winner')}</span>
				<Flag
					iso2={fs.team(champion)?.iso2 ?? ''}
					code={fs.team(champion)?.fifaCode ?? ''}
					size={26}
				/>
				<b>{tname(champion)}</b>
			</div>
		{/if}
		{#each byStage as col (col.stage)}
			<h3 class="rname">{knockoutStageName(col.stage)}</h3>
			{#each col.matches as m (koKey(m))}
				{@const H = sideLabel(m, 'home')}
				{@const A = sideLabel(m, 'away')}
				{@const w = fs.bracket[koKey(m)]}
				{@const actAdv =
					m.num > 0
						? fs.advancerOf(m.num)
						: (fs.results.find(
								(r) => r.stage === m.stage && r.finished
							)?.advancer ?? '')}
				{@const bok = actAdv ? w === actAdv : null}
				<div class="bm card" class:rwin={bok === true} class:rmiss={bok === false}>
					<button
						class="bteam"
						class:win={w && w === H.id}
						disabled={fs.locked || !H.id}
						onclick={() => fs.pick(m, H.id)}
					>
						{#if H.team}<Flag iso2={H.team.iso2} code={H.team.fifaCode} />{/if}
						<span class="bn" class:ph={!H.id}>{H.name}</span>
					</button>
					<span class="vs">vs</span>
					<button
						class="bteam"
						class:win={w && w === A.id}
						disabled={fs.locked || !A.id}
						onclick={() => fs.pick(m, A.id)}
					>
						{#if A.team}<Flag iso2={A.team.iso2} code={A.team.fifaCode} />{/if}
						<span class="bn" class:ph={!A.id}>{A.name}</span>
					</button>
					{#if bok === true}<span class="ind ok"><Check size={15} /></span>
					{:else if bok === false}<span class="ind no"><X size={15} /></span>{/if}
				</div>
			{/each}
		{/each}
	{/if}

	{#if !fs.locked}
		<div class="savebar">
			<span class="savestat" class:err={saveState === 'error'}>
				{#if saveState === 'saving'}
						{language.text('Lagrer...', 'Lagrar…', 'Saving…')}
				{:else if saveState === 'error'}
						{err || language.text('Lagring feilet', 'Lagring feila', 'Save failed')}
				{:else if saveState === 'saved'}
						<Check size={15} /> {language.text('Lagret · endringer lagres automatisk', 'Lagra · endringar blir lagra automatisk', 'Saved · changes are saved automatically')}
				{:else}
						{language.text('Endringer lagres automatisk', 'Endringar blir lagra automatisk', 'Changes are saved automatically')}
				{/if}
			</span>
		</div>
	{/if}
{/if}

<style>
	h1 {
		margin: 0.25rem 0 0.2rem;
	}
	.small {
		font-size: 0.85rem;
	}
	.lockbar {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		color: var(--warning);
	}
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
	@media (min-width: 900px) {
		.stickyhead {
			top: 0;
			margin: 0 -2rem;
			padding: 0.75rem 2rem 0.85rem;
		}
	}
	.grp h3 {
		margin: 0 0 0.6rem;
	}
	.trow {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.45rem 0;
		border-top: 1px solid var(--border);
	}
	.trow:nth-child(2) {
		border-top: none;
	}
	.pos {
		width: 1.2rem;
		text-align: center;
		font-weight: 800;
		color: var(--muted);
	}
	.nm {
		flex: 1;
		font-weight: 600;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.pill.ok {
		color: var(--success);
		border-color: var(--success);
	}
	.ord button {
		background: var(--surface-2);
		border: 1px solid var(--border);
		color: var(--text);
		border-radius: 7px;
		width: 30px;
		height: 26px;
		margin-left: 2px;
	}
	.ord button:disabled {
		color: var(--muted);
		opacity: 0.5;
	}
	.rname {
		margin: 1.2rem 0 0.5rem;
		color: var(--muted);
		font-size: 0.95rem;
	}
	.bm {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 0.7rem;
	}
	.bm + .bm {
		margin-top: 0.5rem;
	}
	.bteam {
		flex: 1;
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.55rem 0.6rem;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		color: var(--text);
		min-width: 0;
	}
	.bteam:disabled {
		opacity: 0.7;
	}
	.bteam.win {
		background: var(--text);
		border-color: var(--text);
		color: var(--bg);
	}
	.bn {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-weight: 600;
		font-size: 0.9rem;
	}
	.bn.ph {
		color: var(--muted);
		font-weight: 500;
	}
	.vs {
		color: var(--muted);
		font-size: 0.8rem;
	}
	.champ {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		color: var(--gold);
		border-color: var(--border-strong);
		background: var(--surface);
		text-shadow: none;
	}
	.champ .lbl {
		text-transform: uppercase;
		letter-spacing: 0.14em;
		font-size: 0.78rem;
		font-weight: 700;
	}
	.champ b {
		font-family: var(--font-display);
		font-size: 1.15rem;
		letter-spacing: 0.02em;
	}
	.gb-head {
		display: flex;
		align-items: flex-start;
		gap: 1rem;
		margin-bottom: 0.7rem;
	}
	.gb-head .small {
		flex: 1;
		margin: 0;
	}
	.gb-pick {
		display: flex;
		align-items: center;
		gap: 0.7rem;
		border-color: color-mix(in srgb, var(--gold) 42%, var(--border));
	}
	.gb-main {
		display: grid;
		gap: 0.15rem;
		min-width: 0;
		flex: 1;
	}
	.gb-main b,
	.gb-main i {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.gb-main i {
		font-style: normal;
		font-size: 0.78rem;
		color: var(--muted);
	}
	.gb-list h3,
	.gb-search h3,
	.gb-live h3 {
		margin: 0 0 0.7rem;
	}
	.gb-search {
		display: grid;
		gap: 0.75rem;
	}
	.gb-search-head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 1rem;
	}
	.gb-search-head .small {
		margin: 0;
	}
	.gb-search-input {
		width: 100%;
		padding: 0.8rem 0.9rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--text);
	}
	.gb-search-input::placeholder {
		color: var(--muted);
	}
	.gb-search-results {
		display: grid;
		gap: 0.55rem;
	}
	.gb-search-result {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		align-items: center;
		gap: 0.7rem;
		padding: 0.65rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--text);
		text-align: left;
	}
	.gb-search-result.picked {
		border-color: color-mix(in srgb, var(--success) 48%, var(--border));
		background: color-mix(in srgb, var(--success) 9%, var(--surface-2));
	}
	.gb-search-result:disabled {
		opacity: 0.88;
	}
	.gb-search-meta {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		color: var(--muted);
		font-size: 0.8rem;
		min-width: 0;
	}
	.gb-search-meta span {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.gb-search-action {
		font-size: 0.72rem;
		font-weight: 700;
		letter-spacing: 0.05em;
		text-transform: uppercase;
		color: var(--muted);
		white-space: nowrap;
	}
	.api-note {
		text-align: right;
	}
	.gb-updated {
		margin: 0 0 0.6rem;
		text-align: right;
	}
	.gb-live-head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 1rem;
	}
	.gb-live-head h3 {
		margin-bottom: 0.6rem;
	}
	.gb-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
		gap: 0.65rem;
	}
	.gb-player {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		min-height: 56px;
		padding: 0.85rem 0.5rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--text);
		text-align: center;
		position: relative;
	}
	.gb-player .headshot-wrap,
	.gb-player .headshot {
		width: 54px;
		height: 54px;
	}
	.gb-player .gb-main {
		display: flex;
		flex-direction: column;
		align-items: center;
		width: 100%;
	}
	.gb-player-status {
		position: absolute;
		top: 0.4rem;
		right: 0.4rem;
	}
	.gb-player.picked {
		border-color: color-mix(in srgb, var(--success) 48%, var(--border));
		background: color-mix(in srgb, var(--success) 9%, var(--surface-2));
	}
	.gb-player:disabled {
		cursor: default;
		opacity: 0.88;
	}
	.headshot-wrap,
	.headshot,
	.mini-headshot {
		display: inline-grid;
		place-items: center;
		border-radius: 50%;
		background: var(--surface);
		border: 1px solid var(--border);
		object-fit: cover;
		flex: none;
	}
	.headshot,
	.headshot-wrap {
		width: 42px;
		height: 42px;
	}
	.mini-headshot {
		width: 28px;
		height: 28px;
		font-size: 0.65rem;
	}
	.fallback {
		font-family: var(--font-display);
		font-weight: 800;
		color: var(--muted);
	}
	.gb-table {
		width: 100%;
		border-collapse: collapse;
	}
	.gb-table th,
	.gb-table td {
		padding: 0.55rem 0.35rem;
		border-bottom: 1px solid var(--border);
		text-align: left;
	}
	.gb-table th {
		color: var(--muted);
		font-size: 0.78rem;
		font-weight: 700;
	}
	.gb-table tr.picked td {
		background: color-mix(in srgb, var(--accent) 8%, transparent);
	}
	.gb-row-player,
	.gb-team {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		min-width: 0;
	}
	.gb-row-player b,
	.gb-team {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.tm-short { display: none; }
	@media (max-width: 500px) {
		.tm-full { display: none; }
		.tm-short { display: inline; }
	}
	.savebar {
		position: sticky;
		bottom: calc(var(--nav-h) + 0.5rem);
		display: flex;
		justify-content: center;
		margin-top: 1.5rem;
		pointer-events: none;
	}
	.savestat {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.8rem;
		font-weight: 600;
		letter-spacing: 0.04em;
		text-transform: uppercase;
		color: var(--muted);
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		padding: 0.4rem 0.85rem;
	}
	.savestat.err {
		color: var(--danger);
		border-color: var(--danger);
		text-transform: none;
		letter-spacing: 0;
	}
	.thead {
		display: flex;
		align-items: flex-start;
		gap: 1rem;
		margin-bottom: 0.6rem;
	}
	.thead .small {
		flex: 1;
	}
	.cnt {
		font-family: var(--font-mono);
		font-weight: 700;
		padding: 0.2rem 0.6rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		color: var(--muted);
		white-space: nowrap;
	}
	.cnt.full {
		color: var(--bg);
		background: var(--text);
		border-color: var(--text);
	}
	.tlist {
		padding: 0.3rem 0.9rem;
	}
	.trow {
		display: flex;
		align-items: center;
		gap: 0.7rem;
		padding: 0.8rem 0;
		min-height: 44px;
		border-top: 1px solid var(--border);
		cursor: pointer;
	}
	.trow:first-child {
		border-top: none;
	}
	.trow input {
		width: 20px;
		height: 20px;
		accent-color: var(--accent);
	}
	.trow .gl {
		display: grid;
		place-items: center;
		width: 24px;
		height: 24px;
		border-radius: 6px;
		background: var(--surface-2);
		font-family: var(--font-display);
		font-size: 0.85rem;
		color: var(--muted);
	}
	.trow.on {
		color: var(--text);
	}
	.trow.on .gl {
		background: var(--text);
		color: var(--bg);
	}
	.trow .nm {
		font-weight: 600;
	}
	.ind {
		display: inline-grid;
		place-items: center;
	}
	.ind.ok {
		color: var(--success);
	}
	.ind.no {
		color: var(--danger);
	}
	.ind.half {
		color: var(--gold);
	}
	.apos.half {
		color: var(--gold);
	}
	.tag {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
	}
	.apos {
		font-size: 0.72rem;
		font-weight: 700;
		letter-spacing: 0.04em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.ind.dim {
		color: var(--muted);
		opacity: 0.7;
	}
	.trow.rwin,
	.bm.rwin {
		border-color: color-mix(in srgb, var(--success) 45%, var(--border));
	}
	.trow.rhalf {
		border-color: color-mix(in srgb, var(--gold) 45%, var(--border));
	}
	.trow.rmiss,
	.bm.rmiss {
		border-color: color-mix(in srgb, var(--danger) 40%, var(--border));
	}
	.bm.rwin,
	.bm.rmiss {
		border-style: solid;
	}
	:global(:root[data-theme='worldcup']) .grp,
	:global(:root[data-theme='worldcup']) .tlist,
	:global(:root[data-theme='worldcup']) .gb-pick,
	:global(:root[data-theme='worldcup']) .gb-search,
	:global(:root[data-theme='worldcup']) .gb-list,
	:global(:root[data-theme='worldcup']) .gb-live,
	:global(:root[data-theme='worldcup']) .bm,
	:global(:root[data-theme='worldcup']) .champ,
	:global(:root[data-theme='worldcup']) .lockbar {
		background:
			radial-gradient(circle at 14% 0%, rgba(143, 197, 143, 0.075), transparent 32%),
			linear-gradient(180deg, rgba(13, 34, 40, 0.96), rgba(7, 17, 25, 0.98)),
			var(--surface);
		border-color: color-mix(in srgb, var(--accent) 12%, var(--border));
		box-shadow: 0 16px 42px -34px rgba(0, 0, 0, 0.9), inset 0 1px 0 rgba(255, 255, 255, 0.035);
	}
	:global(:root[data-theme='worldcup']) .grp::before,
	:global(:root[data-theme='worldcup']) .tlist::before,
	:global(:root[data-theme='worldcup']) .gb-pick::before,
	:global(:root[data-theme='worldcup']) .gb-search::before,
	:global(:root[data-theme='worldcup']) .gb-list::before,
	:global(:root[data-theme='worldcup']) .gb-live::before,
	:global(:root[data-theme='worldcup']) .bm::before,
	:global(:root[data-theme='worldcup']) .champ::before,
	:global(:root[data-theme='worldcup']) .lockbar::before {
		display: none;
	}
	:global(:root[data-theme='worldcup']) .trow {
		border-top-color: color-mix(in srgb, var(--accent) 11%, var(--border));
	}
	:global(:root[data-theme='worldcup']) .trow .gl,
	:global(:root[data-theme='worldcup']) .bteam,
	:global(:root[data-theme='worldcup']) .gb-search-input,
	:global(:root[data-theme='worldcup']) .gb-search-result,
	:global(:root[data-theme='worldcup']) .cnt,
	:global(:root[data-theme='worldcup']) .savestat {
		background: color-mix(in srgb, var(--surface-2) 78%, transparent);
		border-color: color-mix(in srgb, var(--accent) 12%, var(--border));
	}
	:global(:root[data-theme='worldcup']) .bteam.win,
	:global(:root[data-theme='worldcup']) .trow.on .gl,
	:global(:root[data-theme='worldcup']) .cnt.full {
		background: linear-gradient(180deg, color-mix(in srgb, var(--accent) 42%, var(--surface-2)), var(--surface-2));
		border-color: color-mix(in srgb, var(--accent) 36%, var(--border));
		color: var(--text);
	}
</style>
