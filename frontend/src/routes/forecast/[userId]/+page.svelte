<script lang="ts">
	import { page } from '$app/stores';
	import { ForecastStore, koKey, type GoldenBootPlayer, type KOMatch } from '$lib/forecast.svelte';
	import Flag from '$lib/components/Flag.svelte';
	import { teamDisplayName } from '$lib/teamNames';
	import { collapseOnScroll } from '$lib/actions';
	import { Check, CircleCheck, X, Trophy, ArrowLeft } from '@lucide/svelte';
	import { language } from '$lib/language.svelte';
	import { stageName as knockoutStageName } from '$lib/stageLabels';

	const fs = new ForecastStore();
	let section = $state<'groups' | 'thirds' | 'bracket' | 'goldenboot'>('groups');
	let err = $state('');
	$effect(() => {
		const uid = $page.params.userId;
		if (uid) fs.loadView(uid).catch((e) => (err = e?.message ?? language.text('Ikke tilgang', 'Ikkje tilgang', 'Not allowed')));
	});

	const ord = (n: number) =>
		n === 1 ? '1.' : n === 2 ? '2.' : n === 3 ? '3.' : `${n}.`;
	const tname = (id: string) => teamDisplayName(fs.team(id));

	const stages = ['R32', 'R16', 'QF', 'SF', '3RD', 'FINAL'];
	let byStage = $derived(
		stages.map((s) => ({
			stage: s,
			matches: fs.knockout.filter((m) => m.stage === s)
		}))
	);
	let finalMatch = $derived(fs.knockout.find((m) => m.stage === 'FINAL'));
	let champion = $derived(finalMatch ? fs.bracket[koKey(finalMatch)] : '');
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

	function side(m: KOMatch, s: 'home' | 'away') {
		const [h, a] = fs.sides(m);
		const id = s === 'home' ? h : a;
		if (id) return { id, name: tname(id), team: fs.team(id) };
		return {
			id: '',
			name: s === 'home' ? m.homeLabel : m.awayLabel,
			team: undefined
		};
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
</script>

<button class="muted back" type="button" onclick={() => history.back()}>
	<ArrowLeft size={15} /> {language.text('Tilbake', 'Tilbake', 'Back')}
</button>

<div class="stickyhead" use:collapseOnScroll>
	<p class="kicker">{language.text('VM-tips', 'VM-tips', 'World Cup tips')}</p>
	<div class="sh-expand">
		<div class="sh-inner">
			<h1>{fs.viewName || '…'}</h1>
			<p class="muted desc">{language.text('Skrivebeskyttet - VM-tipset til en venn.', 'Skriveverna - VM-tipset til ein ven.', "Read-only - a friend's World Cup tips.")}</p>
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

{#if err}
	<p class="error">{err}</p>
{:else if !fs.loaded}
	<p class="muted">{language.text('Laster...', 'Lastar…', 'Loading…')}</p>
{:else if section === 'groups'}
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
				<div class="trow" class:rwin={state === 'ok'} class:rhalf={state === 'half'} class:rmiss={state === 'miss'}>
					<span class="pos">{i + 1}</span>
					<Flag iso2={fs.team(id)?.iso2 ?? ''} code={fs.team(id)?.fifaCode ?? ''} />
					<span class="nm">{tname(id)}</span>
					<span class="tag">
						{#if state === 'ok'}<span class="ind ok"><Check size={15} /></span>
						{:else if state === 'half'}<span class="apos half">{language.text('faktisk', 'faktisk', 'actual')} {ord(apos)}</span><span class="ind half"><CircleCheck size={15} /></span>
						{:else if state === 'miss'}<span class="apos">{language.text('faktisk', 'faktisk', 'actual')} {ord(apos)}</span><span class="ind no"><X size={15} /></span>
						{:else if i < 2}<span class="pill ok">{language.text('går videre', 'går vidare', 'advances')}</span>
						{:else if i === 2}<span class="pill">{language.text('3.', '3.', '3rd')}</span>{/if}
					</span>
				</div>
			{/each}
		</section>
	{/each}
{:else if section === 'thirds'}
	<section class="card tlist">
		{#each fs.groups as g (g.letter)}
			{@const tid = fs.groupThird(g.letter)}
			{@const on = !!fs.thirds[g.letter]}
			{@const adv = actualThirds ? actualThirds.has(tid) : null}
			{#if on}
				<div class="trow">
					<span class="gl">{g.letter}</span>
					<Flag iso2={fs.team(tid)?.iso2 ?? ''} code={fs.team(tid)?.fifaCode ?? ''} />
					<span class="nm">{tname(tid) || '—'}</span>
					<span class="spacer"></span>
					{#if adv === true}<span class="ind ok"><Check size={15} /></span>
					{:else if adv === false}<span class="ind no"><X size={15} /></span>{/if}
				</div>
			{/if}
		{/each}
		{#if Object.keys(fs.thirds).length === 0}
			<p class="muted small">{language.text('Ingen beste treere er valgt.', 'Ingen beste-trearar er valde.', 'No best-third picks.')}</p>
		{/if}
	</section>
{:else if section === 'goldenboot'}
	<div class="gb-head">
		<p class="muted small">
			{language.text('Toppscorertips og toppscorerliste.', 'Toppscorartips og toppscorarliste.', 'Golden Boot pick and current top-scorer table.')}
		</p>
		<span class="cnt">{language.text('Oppdatert', 'Oppdatert', 'Updated')} {updatedAt(fs.goldenBoot.updatedAt)}</span>
	</div>

	{#if goldenBootPick}
		<section class="card gb-pick">
			<Trophy size={20} />
			<span class="headshot-wrap">
				{#if goldenBootPick.photoUrl}<img class="headshot" src={goldenBootPick.photoUrl} alt="" loading="lazy" />{:else}<span class="headshot fallback">{initials(goldenBootPick.name)}</span>{/if}
			</span>
			<span class="gb-main">
				<i>{language.text('Toppscorertips', 'Toppscorartips', 'Golden Boot pick')}</i>
				<b>{goldenBootPick.name}</b>
			</span>
			<Flag iso2={fs.team(goldenBootPick.teamId)?.iso2 ?? ''} code={fs.team(goldenBootPick.teamId)?.fifaCode ?? ''} />
		</section>
	{:else}
		<section class="card gb-pick empty"><p class="muted small">{language.text('Ingen toppscorertips.', 'Ingen toppscorartips.', 'No Golden Boot pick.')}</p></section>
	{/if}

	<section class="card gb-live">
		<h3>{language.text('Toppscorere', 'Toppscorarar', 'Top scorers')}</h3>
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
						<td><span class="gb-row-player">{#if player.photoUrl}<img class="mini-headshot" src={player.photoUrl} alt="" loading="lazy" />{:else}<span class="mini-headshot fallback">{initials(player.name)}</span>{/if}<b>{player.name}</b></span></td>
						<td><span class="gb-team"><Flag iso2={fs.team(player.teamId)?.iso2 ?? ''} code={fs.team(player.teamId)?.fifaCode ?? ''} /> {player.teamName}</span></td>
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
			<span class="lbl">{language.text('Tippet mester', 'Tippa meister', 'Predicted champion')}</span>
			<Flag iso2={fs.team(champion)?.iso2 ?? ''} code={fs.team(champion)?.fifaCode ?? ''} size={26} />
			<b>{tname(champion)}</b>
		</div>
	{/if}
	{#each byStage as col (col.stage)}
		<h3 class="rname">{knockoutStageName(col.stage)}</h3>
		{#each col.matches as m (koKey(m))}
			{@const H = side(m, 'home')}
			{@const A = side(m, 'away')}
			{@const w = fs.bracket[koKey(m)]}
			{@const actAdv =
				m.num > 0
					? fs.advancerOf(m.num)
					: (fs.results.find((r) => r.stage === m.stage && r.finished)
							?.advancer ?? '')}
			{@const bok = actAdv ? w === actAdv : null}
			<div class="bm card" class:rwin={bok === true} class:rmiss={bok === false}>
				<div class="bteam" class:win={w && w === H.id}>
					{#if H.team}<Flag iso2={H.team.iso2} code={H.team.fifaCode} />{/if}
					<span class="bn" class:ph={!H.id}>{H.name}</span>
				</div>
				<span class="vs">vs</span>
				<div class="bteam right" class:win={w && w === A.id}>
					<span class="bn" class:ph={!A.id}>{A.name}</span>
					{#if A.team}<Flag iso2={A.team.iso2} code={A.team.fifaCode} />{/if}
				</div>
				{#if bok === true}<span class="ind ok"><Check size={15} /></span>
				{:else if bok === false}<span class="ind no"><X size={15} /></span>{/if}
			</div>
		{/each}
	{/each}
{/if}

<style>
	.back {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		margin: 0.25rem 0 0.5rem;
		background: none;
		border: none;
		padding: 0;
		font: inherit;
		color: var(--muted);
		cursor: pointer;
	}
	h1 {
		margin: 0.1rem 0 0;
	}
	.desc {
		margin: 0.3rem 0 0;
		font-size: 0.9rem;
	}
	.grp h3,
	.rname {
		margin: 0 0 0.6rem;
	}
	.rname {
		font-family: var(--font-display);
		text-transform: uppercase;
		color: var(--muted);
		margin: 1.4rem 0 0.6rem;
	}
	.trow {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.5rem 0;
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
	.tag {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
	}
	.gl {
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
	.pill.ok {
		color: var(--accent);
		border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
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
	.apos {
		font-size: 0.72rem;
		font-weight: 700;
		text-transform: uppercase;
		color: var(--muted);
	}
	.apos.half {
		color: var(--gold);
	}
	.trow.rwin {
		border-color: color-mix(in srgb, var(--success) 45%, var(--border));
	}
	.trow.rhalf {
		border-color: color-mix(in srgb, var(--gold) 45%, var(--border));
	}
	.trow.rmiss {
		border-color: color-mix(in srgb, var(--danger) 40%, var(--border));
	}
	.champ {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		color: var(--gold);
		border-color: var(--border-strong);
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
	.cnt {
		font-family: var(--font-mono);
		font-weight: 700;
		padding: 0.2rem 0.6rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		color: var(--muted);
		white-space: nowrap;
	}
	.gb-pick {
		display: flex;
		align-items: center;
		gap: 0.7rem;
		border-color: color-mix(in srgb, var(--gold) 42%, var(--border));
	}
	.gb-pick.empty {
		display: block;
	}
	.gb-pick.empty p {
		margin: 0;
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
	.gb-live h3 {
		margin: 0 0 0.7rem;
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
	.bm {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.6rem 0.8rem;
	}
	.bm + .bm {
		margin-top: 0.5rem;
	}
	.bteam {
		flex: 1;
		display: flex;
		align-items: center;
		gap: 0.5rem;
		min-width: 0;
	}
	.bteam.right {
		justify-content: flex-end;
	}
	.bteam.win .bn {
		color: var(--accent);
		font-weight: 700;
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
	.small {
		font-size: 0.85rem;
	}
</style>
