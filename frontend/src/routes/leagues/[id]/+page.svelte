<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { api, type GoldenBootLeagueTable, type LeaderboardRow, type LeagueInvite, type LeagueInviteUser } from '$lib/api';
	import { auth } from '$lib/auth.svelte';
	import { language } from '$lib/language.svelte';
	import { tipsStore } from '$lib/tips.svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import LeagueChatCard from '$lib/components/LeagueChatCard.svelte';
	import {
		Eye,
		EyeOff,
		Check,
		Copy,
		Share2,
		ChevronDown,
		Lock,
		RefreshCw,
		Settings,
		Telescope,
		Mail,
		Search,
		UserMinus,
		UserPlus,
		X
	} from '@lucide/svelte';

	interface Cfg {
		match: {
			tendency: number;
			exact: number;
			totalGoals: number;
			goalDiff: number;
		};
		forecast: {
			groupPosition: number;
			perfectGroupBonus: number;
			advance: number;
			goldenBootWinner?: number;
			round: Record<string, number>;
		};
		tiebreakers: string[];
	}
	let cfg = $state<Cfg | null>(null);

	let tbLabel = $derived.by<Record<string, string>>(() => ({
		points: language.text('Totalpoeng', 'Totalpoeng', 'Total points'),
		exactScores: language.text('Flest eksakte resultater', 'Flest eksakte resultat', 'Most exact scores'),
		correctWinners: language.text('Flest rette vinnere', 'Flest rette vinnarar', 'Most correct winners'),
		goalDiffDeviation: language.text('Minst målforskjell-feil', 'Minst målforskjell-feil', 'Smallest goal-difference error'),
		fewestTips: language.text('Færrest leverte tips', 'Færrast leverte tips', 'Fewest submitted tips'),
		earliestEdit: language.text('Tidligste siste endring (levert først)', 'Tidlegaste siste endring (levert først)', 'Earliest last edit (submitted first)')
	}));
	let roundLabel = $derived.by<Record<string, string>>(() => ({
		R32: language.text('32-delsfinale', '32-delsfinale', 'Round of 32'),
		R16: language.text('Åttedelsfinale', 'Åttedelsfinale', 'Round of 16'),
		QF: language.text('Kvartfinale', 'Kvartfinale', 'Quarter-final'),
		SF: language.text('Semifinale', 'Semifinale', 'Semi-final'),
		FINAL: language.text('Finale', 'Finale', 'Final'),
		CHAMPION: language.text('Vinner', 'Vinnar', 'Winner')
	}));
	let goldenBootLabel = $derived(language.text('Toppscorer', 'Toppscorar', 'Golden Boot'));

	let revealed = $state(false);
	let openRow = $state<string | null>(null);

	let id = $derived($page.params.id ?? '');
	let league = $state<{ id: string; name: string } | null>(null);
	let role = $state('');
	let isPrivate = $state(false);
	let rows = $state<LeaderboardRow[]>([]);
	let goldenBoot = $state<GoldenBootLeagueTable | null>(null);
	let invite = $state('');
	let loaded = $state(false);
	let error = $state('');
	let tab = $state<'total' | 'tipsPoints' | 'forecastPoints'>('total');
	let deleteConfirm = $state('');
	let deleteBusy = $state(false);
	let deleteError = $state('');
	let inviteAdmin = $state(false);
	let inviteQuery = $state('');
	let inviteCandidates = $state<LeagueInviteUser[]>([]);
	let pendingInvites = $state<LeagueInvite[]>([]);
	let inviteSearchBusy = $state(false);
	let inviteSendBusy = $state('');
	let inviteError = $state('');
	let editing = $state(false);
	let nameDraft = $state('');
	let mgmtBusy = $state(false);
	let mgmtError = $state('');
	let confirmRegen = $state(false);
	let isOwner = $derived(role === 'owner');
	let leaderboardRefreshTimer: ReturnType<typeof setTimeout> | null = null;
	let leaderboardRequestSeq = 0;
	let lastScoreRevision = tipsStore.scoreRevision;

	function clearLeaderboardRefreshTimer() {
		if (!leaderboardRefreshTimer) return;
		clearTimeout(leaderboardRefreshTimer);
		leaderboardRefreshTimer = null;
	}

	function applyLeaderboard(
		lid: string,
		lb: {
			league: { id: string; name: string };
			rows: LeaderboardRow[];
			scoring?: Record<string, unknown>;
			goldenBoot?: GoldenBootLeagueTable;
		},
		preserveNameDraft = false
	) {
		if (id !== lid) return;
		league = lb.league;
		if (!preserveNameDraft) nameDraft = lb.league.name;
		rows = lb.rows;
		goldenBoot = lb.goldenBoot ?? null;
		cfg = (lb.scoring as Cfg | undefined) ?? null;
		error = '';
	}

	async function refreshLeaderboard(lid = id) {
		if (!lid) return;
		const requestSeq = ++leaderboardRequestSeq;
		try {
			const lb = await api.leaderboard(lid);
			if (requestSeq !== leaderboardRequestSeq) return;
			applyLeaderboard(lid, lb, editing);
		} catch {
			// Keep the last known table visible; the normal page load path still
			// owns the full error state.
		}
	}

	function queueLeaderboardRefresh(lid = id) {
		if (!lid) return;
		clearLeaderboardRefreshTimer();
		leaderboardRefreshTimer = setTimeout(() => {
			leaderboardRefreshTimer = null;
			void refreshLeaderboard(lid);
		}, 500);
	}

	$effect(() => {
		const lid = id;
		let cancelled = false;
		clearLeaderboardRefreshTimer();
		loaded = false;
		error = '';
		cfg = null;
		goldenBoot = null;
		inviteAdmin = false;
		inviteQuery = '';
		inviteCandidates = [];
		pendingInvites = [];
		inviteError = '';
		isPrivate = false;
		editing = false;
		nameDraft = '';
		mgmtError = '';
		confirmRegen = false;
		Promise.all([api.leaderboard(lid), api.myLeagues()])
			.then(([lb, mine]) => {
				if (cancelled || id !== lid) return;
				applyLeaderboard(lid, lb);
				const mineLeague = mine.leagues.find((l) => l.id === lid);
				invite = mineLeague?.inviteCode ?? '';
				role = mineLeague?.role ?? '';
				isPrivate = mineLeague?.private ?? false;
				if (mineLeague?.role === 'owner' && invite && invite !== 'GLOBAL') {
					void loadInviteManager(lid);
				}
			})
			.catch(() => {
				if (!cancelled && id === lid) {
					error = language.text('Kunne ikke laste ligaen.', 'Kunne ikkje laste ligaen.', 'Could not load this league.');
				}
			})
			.finally(() => {
				if (!cancelled && id === lid) loaded = true;
			});
		return () => {
			cancelled = true;
			clearLeaderboardRefreshTimer();
		};
	});

	$effect(() => {
		const revision = tipsStore.scoreRevision;
		const lid = id;
		if (!loaded || !lid) {
			lastScoreRevision = revision;
			return;
		}
		if (revision === lastScoreRevision) return;
		lastScoreRevision = revision;
		queueLeaderboardRefresh(lid);
	});

	$effect(() => {
		const lid = id;
		const hasLiveMatches = tipsStore.liveMatchIds.size > 0;
		if (!loaded || !lid || !hasLiveMatches) return;
		queueLeaderboardRefresh(lid);
		const timer = setInterval(() => void refreshLeaderboard(lid), 60_000);
		return () => clearInterval(timer);
	});

	$effect(() => {
		const lid = id;
		const q = inviteQuery.trim();
		if (!inviteAdmin || q.length < 2) {
			inviteCandidates = [];
			inviteSearchBusy = false;
			return;
		}
		let cancelled = false;
		const timer = setTimeout(() => {
			inviteSearchBusy = true;
			api.inviteCandidates(lid, q)
				.then((result) => {
					if (!cancelled) inviteCandidates = result.users;
				})
				.catch(() => {
					if (!cancelled) inviteCandidates = [];
				})
				.finally(() => {
					if (!cancelled) inviteSearchBusy = false;
				});
		}, 220);
		return () => {
			cancelled = true;
			clearTimeout(timer);
		};
	});

	let sorted = $derived(
		[...rows].sort((a, b) => b[tab] - a[tab])
	);
	let fcView = $derived(tab === 'forecastPoints');
	let compactLeaderboard = $state(false);
	const leaderboardPreviewRows = 25;
	let leaderboardScrollable = $derived(rows.length > leaderboardPreviewRows);
	let leaderboardColspan = $derived(compactLeaderboard ? 3 : fcView ? 12 : 8);

	onMount(() => {
		const media = window.matchMedia('(max-width: 759px)');
		const syncCompactLeaderboard = () => {
			compactLeaderboard = media.matches;
		};

		syncCompactLeaderboard();
		media.addEventListener('change', syncCompactLeaderboard);

		return () => {
			media.removeEventListener('change', syncCompactLeaderboard);
		};
	});

	function enterEdit() {
		if (!league || !isOwner) return;
		editing = true;
		nameDraft = league.name;
		mgmtError = '';
		confirmRegen = false;
	}

	function exitEdit() {
		editing = false;
		nameDraft = league?.name ?? '';
		mgmtError = '';
		confirmRegen = false;
	}

	async function saveName() {
		const name = nameDraft.trim();
		if (!league || !name) return;
		if (name === league.name.trim()) {
			exitEdit();
			return;
		}
		mgmtBusy = true;
		mgmtError = '';
		try {
			const result = await api.renameLeague(league.id, name);
			league = { ...league, name: result.name };
			nameDraft = result.name;
			exitEdit();
		} catch {
			mgmtError = language.text('Kunne ikkje endre liganamnet.', 'Kunne ikkje endre liganamnet.', 'Could not rename the league.');
		} finally {
			mgmtBusy = false;
		}
	}

	async function setPrivacy(next: boolean) {
		if (!league || next === isPrivate) return;
		mgmtBusy = true;
		mgmtError = '';
		try {
			const result = await api.setLeagueCodePrivacy(league.id, next);
			isPrivate = result.private;
		} catch {
			mgmtError = language.text('Kunne ikkje oppdatere kode-synet.', 'Kunne ikkje oppdatere kode-synet.', 'Could not update code visibility.');
		} finally {
			mgmtBusy = false;
		}
	}

	async function regenerateCode() {
		if (!league) return;
		mgmtBusy = true;
		mgmtError = '';
		try {
			const result = await api.regenerateLeagueCode(league.id);
			invite = result.inviteCode;
			revealed = true;
			confirmRegen = false;
		} catch {
			mgmtError = language.text('Kunne ikkje lage ny kode.', 'Kunne ikkje lage ny kode.', 'Could not regenerate the code.');
		} finally {
			mgmtBusy = false;
		}
	}

	async function removeMember(userId: string, name: string) {
		if (!league) return;
		const confirmed = window.confirm(
			language.text(
				`Fjern ${name} frå ligaen?`,
				`Fjern ${name} frå ligaen?`,
				`Remove ${name} from this league?`
			)
		);
		if (!confirmed) return;
		mgmtBusy = true;
		mgmtError = '';
		try {
			await api.removeLeagueMember(league.id, userId);
			rows = rows.filter((row) => row.userId !== userId);
			if (openRow === userId) openRow = null;
		} catch {
			mgmtError = language.text('Kunne ikkje fjerne medlemmet.', 'Kunne ikkje fjerne medlemmet.', 'Could not remove the member.');
		} finally {
			mgmtBusy = false;
		}
	}

	function copyInvite() {
		navigator.clipboard?.writeText(invite);
	}

	let linkCopied = $state(false);
	let copyTimer: ReturnType<typeof setTimeout>;
	async function shareInvite() {
		const url = new URL(`/join/${encodeURIComponent(invite)}`, window.location.origin).toString();
		const title = language.text(
			'Bli med i min tippekonkurranse for VM på Midttunet!',
			'Bli med i min tippekonkurranse for VM på Midttunet!',
			'Join my World Cup prediction league on Midttunet!'
		);
		const text = language.text('Trykk her for å utfordre meg.', 'Klikk her for å utfordre meg.', 'Tap here to challenge me.');
		try {
			if (navigator.share) {
				await navigator.share({ title, text, url });
				return;
			}
		} catch (e: unknown) {
			if ((e as { name?: string })?.name === 'AbortError') return;
		}
		await navigator.clipboard?.writeText(url);
		linkCopied = true;
		clearTimeout(copyTimer);
		copyTimer = setTimeout(() => (linkCopied = false), 1800);
	}

	async function deleteLeague() {
		if (!league || deleteConfirm.trim() !== league.name.trim()) return;
		deleteBusy = true;
		deleteError = '';
		try {
			await api.deleteLeague(league.id);
			await goto('/leagues');
		} catch {
			deleteError = language.text('Kunne ikke slette ligaen.', 'Kunne ikkje slette ligaen.', 'Could not delete the league.');
		} finally {
			deleteBusy = false;
		}
	}

	async function loadInviteManager(leagueId: string) {
		try {
			const result = await api.leagueInvites(leagueId);
			if (id !== leagueId) return;
			pendingInvites = result.invites;
			inviteAdmin = true;
		} catch {
			if (id !== leagueId) return;
			pendingInvites = [];
			inviteAdmin = false;
		}
	}

	async function sendInvite(user: LeagueInviteUser) {
		inviteSendBusy = user.id;
		inviteError = '';
		try {
			const result = await api.createLeagueInvite(id, user.id);
			pendingInvites = [result.invite, ...pendingInvites];
			inviteCandidates = inviteCandidates.filter((candidate) => candidate.id !== user.id);
			inviteQuery = '';
		} catch {
			inviteError = language.text('Kunne ikke sende invitasjonen.', 'Kunne ikkje sende invitasjonen.', 'Could not send invite.');
		} finally {
			inviteSendBusy = '';
		}
	}

	function inviteDate(iso: string) {
		const date = new Date(iso);
		if (!Number.isFinite(date.getTime())) return '';
		return new Intl.DateTimeFormat(language.locale, {
			day: '2-digit',
			month: 'short',
			hour: '2-digit',
			minute: '2-digit'
		}).format(date);
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
		if (!iso) return language.text('ikke synket ennå', 'ikkje synka enno', 'not synced yet');
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

<a href="/leagues" class="muted back">← {language.text('Ligaer', 'Ligaer', 'Leagues')}</a>

{#if error}
	<p class="error">{error}</p>
{:else if !loaded}
	<p class="muted">{language.text('Laster...', 'Lastar…', 'Loading…')}</p>
{:else if league}
	<div class="lhead">
		<div class="ltitle">
			<p class="kicker">{language.text('Liga', 'Liga', 'League')}</p>
			{#if editing}
				<input
					class="input nameedit"
					bind:value={nameDraft}
					maxlength="64"
					aria-label={language.text('Liganamn', 'Liganamn', 'League name')}
					onkeydown={(event) => event.key === 'Enter' && void saveName()}
				/>
			{:else}
				<h1>{league.name}</h1>
			{/if}
		</div>
		{#if isOwner}
			<div class="lactions">
				{#if editing}
					<button
						class="btn secondary icon"
						onclick={() => void saveName()}
						disabled={mgmtBusy}
						aria-label={language.text('Lagre liganamn', 'Lagre liganamn', 'Save league name')}
					>
						<Check size={18} />
					</button>
					<button
						class="btn secondary icon"
						onclick={exitEdit}
						disabled={mgmtBusy}
						aria-label={language.text('Avslutt redigering', 'Avslutt redigering', 'Stop editing')}
					>
						<X size={18} />
					</button>
				{:else}
					<button
						class="btn secondary icon"
						onclick={enterEdit}
						aria-label={language.text('Administrer liga', 'Administrer liga', 'Manage league')}
					>
						<Settings size={18} />
					</button>
				{/if}
			</div>
		{/if}
	</div>

	{#if mgmtError}<p class="error">{mgmtError}</p>{/if}

	{#if editing}
		<section class="card vis">
			<div class="muted small">{language.text('Synlegheit for invitasjonskode', 'Synlegheit for invitasjonskode', 'Invite code visibility')}</div>
			<div class="tabs vistabs">
				<button class:active={!isPrivate} onclick={() => void setPrivacy(false)} disabled={mgmtBusy}>
					{language.text('Medlemmer', 'Medlemmer', 'Members')}
				</button>
				<button class:active={isPrivate} onclick={() => void setPrivacy(true)} disabled={mgmtBusy}>
					{language.text('Privat', 'Privat', 'Private')}
				</button>
			</div>
			<p class="muted small hint">
				{#if isPrivate}
					{language.text(
						'Berre eigaren kan sjå og dele invitasjonskoden.',
						'Berre eigaren kan sjå og dele invitasjonskoden.',
						'Only the owner can see and share the invite code.'
					)}
				{:else}
					{language.text(
						'Alle medlemmer i ligaen kan sjå og dele invitasjonskoden.',
						'Alle medlemmer i ligaen kan sjå og dele invitasjonskoden.',
						'All league members can see and share the invite code.'
					)}
				{/if}
			</p>
		</section>
	{/if}

	<section class="card">
		<div class="tabs">
			<button class:active={tab === 'total'} onclick={() => (tab = 'total')}>{language.text('Totalt', 'Totalt', 'Total')}</button>
			<button class:active={tab === 'tipsPoints'} onclick={() => (tab = 'tipsPoints')}>{language.text('Kamptips', 'Kamptips', 'Match tips')}</button>
			<button class:active={tab === 'forecastPoints'} onclick={() => (tab = 'forecastPoints')}>{language.text('VM-tips', 'VM-tips', 'World Cup tips')}</button>
		</div>

		{#if leaderboardScrollable}
			<p class="muted small lb-hint">
				{language.text(
					`Tabellen viser rundt ${leaderboardPreviewRows} spelarar om gongen. Scroll i tabellen for resten.`,
					`Tabellen viser rundt ${leaderboardPreviewRows} spelarar om gongen. Scroll i tabellen for resten.`,
					`The table shows about ${leaderboardPreviewRows} players at a time. Scroll inside it for the rest.`
				)}
			</p>
		{/if}

		<div class="lb-shell" class:scrollable={leaderboardScrollable}>
		<table class="lb">
			<thead>
				<tr>
					<th>#</th>
					<th>{language.text('Spiller', 'Spelar', 'Player')}</th>
					{#if fcView}
						<th class="num ext" title={language.text('Rett gruppeplassering', 'Rett gruppeplassering', 'Correct group placement')}>Grp</th>
						<th class="num ext" title={language.text('Lag som gikk videre fra gruppespill', 'Lag som gjekk vidare frå gruppespel', 'Teams that advanced from group stage')}>{language.text('Vid', 'Vid', 'Adv')}</th>
						<th class="num ext" title={language.text('Tippet lag som nådde 32-delsfinale', 'Tippa lag som nådde 32-delsfinale', 'Predicted team that reached Round of 32')}>R32</th>
						<th class="num ext" title={language.text('Tippet lag som nådde åttedelsfinale', 'Tippa lag som nådde åttedelsfinale', 'Predicted team that reached Round of 16')}>R16</th>
						<th class="num ext" title={language.text('Tippet lag som nådde kvartfinale', 'Tippa lag som nådde kvartfinale', 'Predicted team that reached quarter-final')}>QF</th>
						<th class="num ext" title={language.text('Tippet lag som nådde semifinale', 'Tippa lag som nådde semifinale', 'Predicted team that reached semi-final')}>SF</th>
						<th class="num ext" title={language.text('Tippet lag som nådde finale', 'Tippa lag som nådde finale', 'Predicted team that reached final')}>F</th>
						<th class="num ext" title={goldenBootLabel}>{language.text('TS', 'TS', 'GB')}</th>
						<th class="num ext" title={language.text('Rett vinner tippet', 'Rett vinnar tippa', 'Correct winner predicted')}>{language.text('Vinn', 'Vinn', 'Win')}</th>
					{:else}
						<th class="num ext" title={language.text('Kamper tippet', 'Kampar tippa', 'Matches tipped')}>Tips</th>
						<th class="num ext" title={language.text('VM-tipspoeng', 'VM-tipspoeng', 'World Cup tip points')}>{language.text('VM', 'VM', 'WC')}</th>
						<th class="num ext" title={language.text('Eksakte resultater (tie-break 1)', 'Eksakte resultat (tie-break 1)', 'Exact scores (tiebreaker 1)')}>{language.text('Eksakt', 'Eksakt', 'Exact')}</th>
						<th class="num ext" title={language.text('Rette vinnere (tie-break 2)', 'Rette vinnarar (tie-break 2)', 'Correct winners (tiebreaker 2)')}>{language.text('Vinn', 'Vinn', 'Win')}</th>
						<th class="num ext" title={language.text('Målforskjell-feil (tie-break 3, lavere er bedre)', 'Målforskjell-feil (tie-break 3, lågare er betre)', 'Goal-difference error (tiebreaker 3, lower is better)')}>GD&Delta;</th>
					{/if}
					<th class="num pts">{language.text('Poeng', 'Poeng', 'Points')}</th>
				</tr>
			</thead>
			<tbody>
				{#each sorted as r, i (r.userId)}
					{@const f = r.forecast ?? {}}
					<tr
						class:lead={r.userId === auth.user?.id}
						class="main"
						class:open={openRow === r.userId}
						onclick={() =>
							(openRow = openRow === r.userId ? null : r.userId)}
					>
						<td class="rank">
							<span class="rank-main">{#if i === 0}🥇
							{:else if i === 1}🥈
							{:else if i === 2}🥉
							{:else}{i + 1}{/if}</span>
							{#if r.rankDelta > 0}<span class="delta up">↑{r.rankDelta}</span>
							{:else if r.rankDelta < 0}<span class="delta dn">↓{Math.abs(r.rankDelta)}</span>{/if}
						</td>
						<td class="player">
							<div class="pwrap">
								<Avatar name={r.name} src={r.avatarUrl} size={28} />
								<span class="pname">{r.name}</span>
								<ChevronDown size={14} class="rx" />
							</div>
						</td>
						{#if fcView}
							<td class="num ext digits">{f.groups ?? 0}</td>
							<td class="num ext digits">{f.advance ?? 0}</td>
							<td class="num ext digits">{f.R32 ?? 0}</td>
							<td class="num ext digits">{f.R16 ?? 0}</td>
							<td class="num ext digits">{f.QF ?? 0}</td>
							<td class="num ext digits">{f.SF ?? 0}</td>
							<td class="num ext digits">{f.FINAL ?? 0}</td>
							<td class="num ext digits">{f.goldenBoot ? '✓' : '–'}</td>
							<td class="num ext digits">{f.champion ? '✓' : '–'}</td>
						{:else}
							<td class="num ext digits">{r.predicted}</td>
							<td class="num ext digits">{r.forecastPoints}</td>
							<td class="num ext digits">{r.exactScores}</td>
							<td class="num ext digits">{r.correctWinners}</td>
							<td class="num ext digits">{r.gdDeviation}</td>
						{/if}
						<td class="num pts digits">{r[tab]}</td>
					</tr>
					{#if openRow === r.userId}
						<tr class="detail">
							<td colspan={leaderboardColspan}>
								<div class="detail-panel">
									<div class="detail-head">
										<a
											class="detail-link"
											href={`/forecast/${r.userId}`}
											title={language.text(`Se VM-tipset til ${r.name}`, `Sjå VM-tipset til ${r.name}`, `View ${r.name}'s World Cup tips`)}
											onclick={(event) => event.stopPropagation()}
										>
											<Telescope size={15} />
											<span>{language.text('Sjå VM-tipset', 'Sjå VM-tipset', 'View World Cup tip')}</span>
										</a>
										{#if editing && r.userId !== auth.user?.id}
											<button
												class="btn secondary detail-remove"
												disabled={mgmtBusy}
												onclick={() => void removeMember(r.userId, r.name)}
											>
												<UserMinus size={15} />
												<span>{language.text('Fjern medlem', 'Fjern medlem', 'Remove member')}</span>
											</button>
										{/if}
									</div>
									{#if fcView}
										<div class="stats">
											<span><i>{language.text('Rett gruppeplassering', 'Rett gruppeplassering', 'Correct group placement')}</i><b>{f.groups ?? 0}</b></span>
											<span><i>{language.text('Lag videre', 'Lag vidare', 'Advanced team')}</i><b>{f.advance ?? 0}</b></span>
											<span><i>{language.text('Nådde 32-delsfinale', 'Nådde 32-delsfinale', 'Reached Round of 32')}</i><b>{f.R32 ?? 0}</b></span>
											<span><i>{language.text('Nådde åttedelsfinale', 'Nådde åttedelsfinale', 'Reached Round of 16')}</i><b>{f.R16 ?? 0}</b></span>
											<span><i>{language.text('Nådde kvartfinale', 'Nådde kvartfinale', 'Reached quarter-final')}</i><b>{f.QF ?? 0}</b></span>
											<span><i>{language.text('Nådde semifinale', 'Nådde semifinale', 'Reached semi-final')}</i><b>{f.SF ?? 0}</b></span>
											<span><i>{language.text('Nådde finale', 'Nådde finale', 'Reached final')}</i><b>{f.FINAL ?? 0}</b></span>
											<span><i>{goldenBootLabel}</i><b>{f.goldenBoot ? language.text('Ja', 'Ja', 'Yes') : language.text('Nei', 'Nei', 'No')} · {f.goldenBootPoints ?? 0} p</b></span>
											<span><i>{language.text('Rett vinner', 'Rett vinnar', 'Correct winner')}</i><b>{f.champion ? language.text('Ja', 'Ja', 'Yes') : language.text('Nei', 'Nei', 'No')}</b></span>
										</div>
									{:else}
										<div class="stats">
											<span><i>{language.text('Kamper tippet', 'Kampar tippa', 'Matches tipped')}</i><b>{r.predicted}</b></span>
											<span><i>{language.text('Kamptipspoeng', 'Kamptipspoeng', 'Match tip points')}</i><b>{r.tipsPoints}</b></span>
											<span><i>{language.text('VM-tipspoeng', 'VM-tipspoeng', 'World Cup tip points')}</i><b>{r.forecastPoints}</b></span>
											<span><i>{language.text('Eksakte resultater', 'Eksakte resultat', 'Exact scores')}</i><b>{r.exactScores}</b></span>
											<span><i>{language.text('Rette vinnere', 'Rette vinnarar', 'Correct winners')}</i><b>{r.correctWinners}</b></span>
											<span><i>{language.text('Målforskjell-feil', 'Målforskjell-feil', 'Goal-difference error')}</i><b>{r.gdDeviation}</b></span>
										</div>
									{/if}
									</div>
							</td>
						</tr>
					{/if}
				{/each}
			</tbody>
		</table>
		</div>

		<p class="muted small note">
			{language.text('Poengene oppdateres automatisk når resultatene kommer.', 'Poenga blir oppdaterte automatisk når resultata kjem.', 'Points update automatically as results come in.')}
		</p>

		{#if fcView && goldenBoot && goldenBoot.players.length > 0}
			<div class="gb-panel">
				<div class="gb-title">
					<h3>{language.text('Toppscorere', 'Toppscorarar', 'Top scorers')}</h3>
					<span class="muted small">{language.text('Oppdatert', 'Oppdatert', 'Updated')} {updatedAt(goldenBoot.updatedAt)}</span>
				</div>
				<table class="gb-table">
					<thead>
						<tr>
							<th>#</th>
							<th>{language.text('Spiller', 'Spelar', 'Player')}</th>
							<th>{language.text('Lag', 'Lag', 'Team')}</th>
							<th class="num">{language.text('Mål', 'Mål', 'Goals')}</th>
							<th>{language.text('Tippet av', 'Tippa av', 'Picked by')}</th>
						</tr>
					</thead>
					<tbody>
						{#each goldenBoot.players as player (player.id)}
							<tr class:mine={player.picks.some((pick) => pick.id === auth.user?.id)}>
								<td class="digits">{player.rank || '–'}</td>
								<td>
									<span class="gb-player-cell">
										{#if player.photoUrl}<img class="gb-photo" src={player.photoUrl} alt="" loading="lazy" />{:else}<span class="gb-photo fallback">{initials(player.name)}</span>{/if}
										<b>{player.name}</b>
									</span>
								</td>
								<td>{player.teamName}</td>
								<td class="num digits">{player.goals}</td>
								<td>
									{#if player.picks.length > 0}
										<div class="pickers">
											{#each player.picks as pick (pick.id)}
												<span class="pick-chip" class:me={pick.id === auth.user?.id} title={pick.name}>
													<Avatar name={pick.name} src={pick.avatarUrl} size={22} />
													<span>{pick.name}</span>
												</span>
											{/each}
										</div>
									{:else}
										<span class="muted small">–</span>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>

	{#if invite && invite !== 'GLOBAL'}
		<LeagueChatCard leagueId={league.id} />
	{/if}

	{#if invite && invite !== 'GLOBAL'}
		<section class="card invite">
			<div class="invite-head">
				<h3>{language.text('Del liga', 'Del liga', 'Share league')}</h3>
				<p class="muted small">{language.text('Del koden eller lenken med dem du vil invitere.', 'Del koden eller lenka med dei du vil invitere.', 'Share the code or link with the people you want to invite.')}</p>
			</div>
			<div class="irow">
				<div class="ic">
					<div class="muted small">
						{language.text('Invitasjonskode', 'Invitasjonskode', 'Invite code')}
						{#if isPrivate}
							<span class="lockpill"><Lock size={11} /> {language.text('Privat', 'Privat', 'Private')}</span>
						{/if}
					</div>
					<div class="code" class:masked={!revealed}>
						{revealed ? invite : '•'.repeat(invite.length || 6)}
					</div>
				</div>
				<div class="spacer"></div>
				<button
					class="btn secondary eye"
					aria-label={revealed ? language.text('Skjul kode', 'Skjul kode', 'Hide code') : language.text('Vis kode', 'Vis kode', 'Show code')}
					onclick={() => (revealed = !revealed)}
				>
					{#if revealed}<EyeOff size={18} />{:else}<Eye size={18} />{/if}
				</button>
				<button class="btn secondary copy" onclick={copyInvite}>
					<Copy size={16} /> {language.text('Kopier', 'Kopier', 'Copy')}
				</button>
			</div>
			<button class="btn share" onclick={shareInvite}>
				<Share2 size={16} />
				{linkCopied ? language.text('Lenken er kopiert!', 'Lenka er kopiert!', 'Link copied!') : language.text('Del invitasjonslenke', 'Del invitasjonslenke', 'Share invite link')}
			</button>
			{#if editing}
				{#if confirmRegen}
					<p class="muted small hint regwarn">
						{language.text(
							'Dette gjer dagens kode og delte lenker ugyldige.',
							'Dette gjer dagens kode og delte lenker ugyldige.',
							'This makes the current code and shared links invalid.'
						)}
					</p>
					<div class="regrow">
						<button class="btn danger" onclick={() => void regenerateCode()} disabled={mgmtBusy}>
							{language.text('Lag ny kode', 'Lag ny kode', 'Regenerate code')}
						</button>
						<button class="btn secondary" onclick={() => (confirmRegen = false)} disabled={mgmtBusy}>
							{language.text('Avbryt', 'Avbryt', 'Cancel')}
						</button>
					</div>
				{:else}
					<button class="btn ghost regenbtn" onclick={() => (confirmRegen = true)}>
						<RefreshCw size={16} /> {language.text('Lag ny kode', 'Lag ny kode', 'Regenerate code')}
					</button>
				{/if}
			{/if}
		</section>
	{/if}

	{#if invite && invite !== 'GLOBAL' && inviteAdmin}
		<section class="card invite-manager">
			<div class="invite-head">
				<h3><Mail size={17} /> {language.text('Inviter folk', 'Inviter folk', 'Invite people')}</h3>
				<p class="muted small">{language.text('Send en forespørsel i appen til en registrert bruker.', 'Send ei førespurnad i appen til ein registrert brukar.', 'Send an in-app request to a registered user.')}</p>
			</div>

			<label class="field invite-search">
				<span class="muted small">{language.text('Søk etter brukere', 'Søk etter brukarar', 'Search users')}</span>
				<span class="search-shell">
					<Search size={16} />
					<input
						class="input"
						bind:value={inviteQuery}
						placeholder={language.text('Navn eller e-post', 'Namn eller e-post', 'Name or email')}
						autocomplete="off"
					/>
				</span>
			</label>

			{#if inviteQuery.trim().length > 0 && inviteQuery.trim().length < 2}
				<p class="muted small invite-note">{language.text('Skriv minst 2 tegn.', 'Skriv minst 2 teikn.', 'Type at least 2 characters.')}</p>
			{:else if inviteSearchBusy}
				<p class="muted small invite-note">{language.text('Søker...', 'Søkjer...', 'Searching...')}</p>
			{:else if inviteQuery.trim().length >= 2 && inviteCandidates.length === 0}
				<p class="muted small invite-note">{language.text('Fant ingen tilgjengelige brukere.', 'Fann ingen tilgjengelege brukarar.', 'No available users found.')}</p>
			{/if}

			{#if inviteCandidates.length > 0}
				<div class="candidate-list">
					{#each inviteCandidates as candidate (candidate.id)}
						<div class="candidate-row">
							<Avatar name={candidate.name} src={candidate.avatarUrl} size={38} />
							<span class="candidate-main">
								<b>{candidate.name}</b>
								{#if candidate.email}<i>{candidate.email}</i>{/if}
							</span>
							<button
								class="btn secondary invite-person"
								disabled={!!inviteSendBusy}
								onclick={() => sendInvite(candidate)}
							>
								<UserPlus size={16} /> {inviteSendBusy === candidate.id ? language.text('Sender...', 'Sender...', 'Sending...') : language.text('Inviter', 'Inviter', 'Invite')}
							</button>
						</div>
					{/each}
				</div>
			{/if}

			{#if pendingInvites.length > 0}
				<div class="pending-list">
					<p class="kicker">{language.text('Ventende', 'Ventande', 'Pending')}</p>
					{#each pendingInvites as pending (pending.id)}
						<div class="pending-row">
							<Avatar name={pending.invitedUser.name} src={pending.invitedUser.avatarUrl} size={34} />
							<span>
								<b>{pending.invitedUser.name}</b>
								{#if pending.invitedUser.email}<i>{pending.invitedUser.email}</i>{/if}
							</span>
							<em>{inviteDate(pending.created)}</em>
						</div>
					{/each}
				</div>
			{/if}

			{#if inviteError}<p class="error">{inviteError}</p>{/if}
		</section>
	{/if}

	{#if role === 'owner' && invite !== 'GLOBAL'}
		<section class="card danger-zone">
			<h3>{language.text('Slett liga', 'Slett liga', 'Delete league')}</h3>
			<p class="muted">
				{language.text(
					'Dette sletter ligaen permanent og fjerner alle medlemskap. Skriv liganavnet for å bekrefte.',
					'Dette slettar ligaen permanent og fjernar alle medlemskap. Skriv liganamnet for å stadfeste.',
					'This permanently deletes the league and removes all memberships. Type the league name to confirm.'
				)}
			</p>
			<label class="field">
				<span class="muted small">{language.text('Skriv', 'Skriv', 'Type')} {league.name}</span>
				<input class="input" bind:value={deleteConfirm} placeholder={league.name} />
			</label>
			<button
				class="btn danger"
				disabled={deleteBusy || deleteConfirm.trim() !== league.name.trim()}
				onclick={deleteLeague}
			>
				{deleteBusy ? language.text('Sletter...', 'Slettar…', 'Deleting…') : language.text('Slett liga permanent', 'Slett liga permanent', 'Delete league permanently')}
			</button>
			{#if deleteError}<p class="error">{deleteError}</p>{/if}
		</section>
	{/if}

	{#if cfg}
		<details class="card legend">
			<summary>{language.text('Slik fungerer poengene', 'Slik fungerer poenga', 'How points work')}</summary>

			<h4>{language.text('Per kamp (kamptips)', 'Per kamp (kamptips)', 'Per match (match tips)')} — {language.text('maks', 'maks', 'max')} {cfg.match.tendency +
					cfg.match.exact +
					cfg.match.totalGoals +
					cfg.match.goalDiff} p</h4>
			<ul class="leg">
				<li>
					<span>{language.text('Rett resultat - gruppespill: H / U / B; sluttspill: laget som går videre', 'Rett resultat - gruppespel: H / U / B; sluttspel: laget som går vidare', 'Correct result - group stage: H / D / A; knockout: the team that advances')}</span><b>{cfg.match.tendency} p</b>
				</li>
				<li><span>{language.text('Eksakt resultat', 'Eksakt resultat', 'Exact score')}</span><b>+{cfg.match.exact} p</b></li>
				<li><span>{language.text('Rett totalt mål', 'Rett totalt mål', 'Correct total goals')}</span><b>+{cfg.match.totalGoals} p</b></li>
				<li><span>{language.text('Rett målforskjell', 'Rett målforskjell', 'Correct goal difference')}</span><b>+{cfg.match.goalDiff} p</b></li>
			</ul>
			<p class="muted small">
				{language.text(
					'Sluttspillkamper kan ikke ende uavgjort - resultatpoengene går til laget som går videre. Blir en sluttspillkamp avgjort etter ekstraomganger, brukes stillingen etter ekstraomganger til poeng.',
					'Sluttspelkampar kan ikkje ende uavgjort - resultatpoenga går til laget som går vidare. Blir ein sluttspelkamp avgjord etter ekstraomgangar, blir stillinga etter ekstraomgangar brukt til poeng.',
					'Knockout matches cannot end in a draw - the result points go to the team that advances. If a knockout match is decided in extra time, the score after extra time is used for points.'
				)}
			</p>

			<h4>{language.text('VM-tips for turneringen', 'VM-tips for turneringa', 'World Cup tips for the tournament')}</h4>
			<ul class="leg">
				<li><span>{language.text('Hvert lag på rett gruppeplassering', 'Kvart lag på rett gruppeplassering', 'Each team in the correct group position')}</span><b>{cfg.forecast.groupPosition} p</b></li>
				<li><span>{language.text('Hele gruppen i rett rekkefølge (bonus)', 'Heile gruppa i rett rekkjefølgje (bonus)', 'The full group in the correct order (bonus)')}</span><b>+{cfg.forecast.perfectGroupBonus} p</b></li>
				<li>
					<span>{language.text('Hvert lag du tippet videre (topp 2 i en gruppe eller beste treer) som faktisk går videre', 'Kvart lag du tippa vidare (topp 2 i ei gruppe eller beste trear) som faktisk går vidare', 'Each team you picked to advance (top 2 in a group or a best third) that actually goes through')}</span
					><b>{cfg.forecast.advance} p</b>
				</li>
				<li><span>{language.text('Rett toppscorer', 'Rett toppscorar', 'Correct Golden Boot winner')}</span><b>{cfg.forecast.goldenBootWinner ?? 15} p</b></li>
			</ul>
			<p class="muted small">
				{language.text('Nådde sluttspillrunde (per rett tippet lag):', 'Nådde sluttspelet (per rett tippa lag):', 'Reached knockout round (per correctly predicted team):')}
			</p>
			<ul class="leg">
				{#each Object.entries(roundLabel) as [k, lbl] (k)}
					{#if cfg.forecast.round[k] != null}
						<li><span>{lbl}</span><b>{cfg.forecast.round[k]} p</b></li>
					{/if}
				{/each}
			</ul>

			<h4>{language.text('Tie-break (i rekkefølge)', 'Tie-break (i rekkjefølgje)', 'Tiebreakers (in order)')}</h4>
			<ol class="tiebreak">
				{#each cfg.tiebreakers as t (t)}
					<li>{tbLabel[t] ?? t}</li>
				{/each}
			</ol>
		</details>
	{/if}
{/if}

<style>
	.back {
		display: inline-block;
		margin: 0.5rem 0 0.75rem;
	}
	h1 {
		margin: 0 0 1rem;
	}
	.lhead {
		display: flex;
		align-items: flex-end;
		gap: 0.75rem;
		margin-bottom: 1rem;
	}
	.ltitle {
		flex: 1;
		min-width: 0;
	}
	.lhead .kicker {
		margin: 0;
	}
	.lhead h1 {
		margin: 0.1rem 0 0;
	}
	.nameedit {
		width: 100%;
		font-size: 1.5rem;
		font-weight: 700;
		margin-top: 0.15rem;
	}
	.lactions {
		display: flex;
		gap: 0.4rem;
		flex: none;
	}
	.icon {
		width: auto;
		padding: 0.6rem;
	}
	.vis {
		margin-bottom: 1rem;
	}
	.vistabs {
		margin: 0.5rem 0 0;
	}
	.hint {
		margin: 0.5rem 0 0;
	}
	.lockpill {
		display: inline-flex;
		align-items: center;
		gap: 0.2rem;
		margin-left: 0.35rem;
		padding: 0.05rem 0.4rem;
		border: 1px solid var(--border);
		border-radius: 999px;
		font-size: 0.7rem;
		vertical-align: middle;
	}
	.regenbtn {
		width: auto;
		margin-top: 0.85rem;
	}
	.regwarn {
		margin-top: 0.85rem;
	}
	.regrow {
		display: flex;
		gap: 0.5rem;
		margin-top: 0.4rem;
	}
	.regrow .btn {
		width: auto;
	}
	.irow {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.share {
		margin-top: 0.85rem;
	}
	.invite-head {
		display: grid;
		gap: 0.25rem;
		margin-bottom: 0.9rem;
	}
	.invite-head h3,
	.invite-head p {
		margin: 0;
	}
	.invite-head h3 {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}
	.invite-manager {
		display: grid;
		gap: 0.75rem;
	}
	.invite-search {
		display: grid;
		gap: 0.35rem;
	}
	.search-shell {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr);
		align-items: center;
		gap: 0.45rem;
		padding: 0 0.75rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface);
		color: var(--muted);
	}
	.search-shell .input {
		border: 0;
		padding-left: 0;
		background: transparent;
	}
	.search-shell .input:focus {
		outline: none;
	}
	.invite-note {
		margin: -0.25rem 0 0;
	}
	.candidate-list,
	.pending-list {
		display: grid;
		gap: 0.5rem;
	}
	.candidate-row,
	.pending-row {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		align-items: center;
		gap: 0.65rem;
		padding: 0.65rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface);
	}
	.candidate-main,
	.pending-row span {
		display: grid;
		gap: 0.15rem;
		min-width: 0;
	}
	.candidate-main b,
	.pending-row b {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.candidate-main i,
	.pending-row i,
	.pending-row em {
		color: var(--muted);
		font-size: 0.78rem;
		font-style: normal;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.invite-person {
		width: auto;
		padding: 0.6rem 0.75rem;
	}
	.pending-list .kicker {
		margin: 0.25rem 0 0;
	}
	.danger-zone {
		border-color: color-mix(in srgb, var(--danger) 35%, var(--border));
		background: color-mix(in srgb, var(--danger) 6%, var(--surface-1));
	}
	.danger-zone h3 {
		margin: 0 0 0.4rem;
		color: var(--danger);
	}
	.danger-zone .field {
		display: grid;
		gap: 0.35rem;
		margin: 0.9rem 0;
	}
	.btn.danger {
		background: var(--danger);
		color: var(--bg);
		border-color: var(--danger);
	}
	.btn.danger:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}
	.ic {
		min-width: 0;
	}
	.small {
		font-size: 0.8rem;
	}
	.code {
		font-family: var(--font-mono);
		font-weight: 700;
		letter-spacing: 0.2em;
		font-size: 1.3rem;
	}
	.code.masked {
		color: var(--muted);
		letter-spacing: 0.15em;
	}
	.eye {
		width: auto;
		padding: 0.7rem;
	}
	.copy {
		width: auto;
	}
	.tabs {
		display: flex;
		gap: 0.4rem;
		margin-bottom: 0.75rem;
	}
	.tabs button {
		flex: 1;
		padding: 0.5rem;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		color: var(--muted);
		font-weight: 600;
	}
	.tabs button.active {
		color: var(--bg);
		background: var(--text);
		border-color: var(--text);
	}
	.lb-hint {
		margin: -0.2rem 0 0.55rem;
	}
	.lb-shell {
		width: 100%;
	}
	.lb-shell.scrollable {
		max-height: clamp(18rem, 72vh, 52rem);
		overflow: auto;
		overscroll-behavior: contain;
		scrollbar-gutter: stable;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface);
	}
	.lb-shell.scrollable .lb {
		margin: 0;
	}
	.lb {
		width: 100%;
		border-collapse: collapse;
	}
	.lb th,
	.lb td {
		text-align: left;
		padding: 0.6rem 0.4rem;
		border-bottom: 1px solid var(--border);
	}
	.lb th {
		color: var(--muted);
		font-size: 0.8rem;
		font-weight: 600;
	}
	.lb-shell.scrollable .lb th {
		position: sticky;
		top: 0;
		z-index: 1;
		background: color-mix(in srgb, var(--surface-2) 88%, var(--surface));
		backdrop-filter: blur(10px);
	}
	.num {
		text-align: right;
	}
	.rank {
		width: 1.7rem;
		padding-right: 0.15rem;
		text-align: center;
		color: var(--muted);
		font-family: var(--font-mono);
	}
	.rank-main {
		display: block;
		line-height: 1;
	}
	.delta {
		display: block;
		font-size: 0.6rem;
		font-weight: 700;
		line-height: 1;
		margin-top: 0.15rem;
		letter-spacing: 0.01em;
	}
	.delta.up { color: var(--success); }
	.delta.dn { color: var(--danger); }
	tr.lead td {
		background: color-mix(in srgb, var(--accent) 9%, transparent);
	}
	tr.lead .rank {
		color: var(--accent);
		font-weight: 800;
	}
	.lb th.num,
	.lb td.num {
		text-align: right;
	}

	/* Pts is the focus — set it apart from the stat columns. */
	.lb th.pts,
	.lb td.pts {
		padding-left: 1.15rem;
		border-left: 1px solid var(--border);
		font-size: 1.02rem;
	}
	.lb th.pts {
		font-size: 0.8rem;
	}

	/* Extra tiebreaker columns: desktop only. */
	.ext {
		display: none;
	}
	.player {
		width: 100%;
		min-width: 0;
		padding-left: 0.15rem;
	}
	.pwrap {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		min-width: 0;
		width: 100%;
	}
	.pname {
		flex: 1;
		min-width: 0;
		max-width: 100%;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.detail-panel {
		display: grid;
		gap: 0.7rem;
		min-width: 0;
	}
	.detail-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.6rem;
		padding-top: 0.45rem;
		min-width: 0;
	}
	.detail-link {
		display: inline-grid;
		grid-auto-flow: column;
		align-items: center;
		gap: 0.45rem;
		color: var(--muted);
		flex: none;
		font-weight: 700;
		min-width: 0;
		max-width: 100%;
	}
	.detail-link span,
	.detail-remove span {
		min-width: 0;
	}
	.detail-link:hover {
		color: var(--accent);
	}
	.detail-remove {
		width: auto;
		padding: 0.45rem 0.65rem;
		display: inline-grid;
		grid-auto-flow: column;
		align-items: center;
		gap: 0.4rem;
		min-width: 0;
		max-width: 100%;
	}
	:global(.lb .rx) {
		color: var(--muted);
		transition: transform 0.15s ease;
		flex: none;
	}
	tr.main.open :global(.rx) {
		transform: rotate(180deg);
	}
	tr.main {
		cursor: pointer;
	}
	.detail td {
		padding: 0 0.4rem 0.7rem;
		min-width: 0;
	}
	.stats {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.4rem 1rem;
		min-width: 0;
	}
	.stats span {
		display: flex;
		justify-content: space-between;
		gap: 0.6rem;
		padding: 0.35rem 0;
		border-bottom: 1px solid var(--border);
		min-width: 0;
	}
	.stats i {
		color: var(--muted);
		font-style: normal;
		font-size: 0.85rem;
		min-width: 0;
		overflow-wrap: anywhere;
	}
	.stats b {
		font-family: var(--font-mono);
		flex: none;
	}

	@media (min-width: 760px) {
		.ext {
			display: table-cell;
		}
	}
	@media (max-width: 759px) {
		.lb {
			table-layout: fixed;
		}
		.lb th:first-child,
		.lb td.rank {
			width: 2.35rem;
		}
		.lb th,
		.lb td {
			padding-left: 0.3rem;
			padding-right: 0.3rem;
		}
		.lb th.pts,
		.lb td.pts {
			width: 4rem;
			padding-left: 0.6rem;
		}
	}
	@media (max-width: 560px) {
		.stats {
			grid-template-columns: 1fr;
		}
		.stats span {
			align-items: baseline;
		}
		.detail-head {
			flex-direction: column;
			align-items: stretch;
		}
		.detail-link,
		.detail-remove {
			width: 100%;
			justify-content: center;
		}
		.detail-link {
			display: flex;
			flex-wrap: wrap;
			text-align: center;
		}
		.candidate-row,
		.pending-row {
			grid-template-columns: auto minmax(0, 1fr);
		}
		.invite-person,
		.pending-row em {
			grid-column: 1 / -1;
		}
		.invite-person {
			width: 100%;
		}
	}
	@media (max-width: 360px) {
		.pwrap {
			gap: 0.35rem;
		}
	}
	.note {
		margin: 0.75rem 0 0;
	}
	.gb-panel {
		margin-top: 1rem;
		padding-top: 0.9rem;
		border-top: 1px solid var(--border);
		overflow-x: auto;
	}
	.gb-title {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 1rem;
		margin-bottom: 0.65rem;
	}
	.gb-title h3 {
		margin: 0;
	}
	.gb-table {
		width: 100%;
		min-width: 620px;
		border-collapse: collapse;
	}
	.gb-table th,
	.gb-table td {
		padding: 0.55rem 0.4rem;
		border-bottom: 1px solid var(--border);
		text-align: left;
		vertical-align: middle;
	}
	.gb-table th {
		color: var(--muted);
		font-size: 0.78rem;
		font-weight: 700;
	}
	.gb-table tr.mine td {
		background: color-mix(in srgb, var(--accent) 8%, transparent);
	}
	.gb-player-cell {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		min-width: 0;
	}
	.gb-player-cell b {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.gb-photo {
		display: inline-grid;
		place-items: center;
		width: 32px;
		height: 32px;
		border-radius: 50%;
		object-fit: cover;
		border: 1px solid var(--border);
		background: var(--surface-2);
		font-size: 0.7rem;
		flex: none;
	}
	.fallback {
		font-family: var(--font-display);
		font-weight: 800;
		color: var(--muted);
	}
	.pickers {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.35rem;
	}
	.pick-chip {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		max-width: 11rem;
		padding: 0.18rem 0.45rem 0.18rem 0.2rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: var(--surface-2);
		font-size: 0.78rem;
		font-weight: 700;
	}
	.pick-chip.me {
		border-color: color-mix(in srgb, var(--accent) 55%, var(--border));
		background: color-mix(in srgb, var(--accent) 14%, var(--surface-2));
	}
	.pick-chip span {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.legend summary {
		cursor: pointer;
		font-weight: 700;
		letter-spacing: 0.04em;
		text-transform: uppercase;
		font-size: 0.85rem;
		color: var(--accent);
	}
	.legend h4 {
		margin: 1rem 0 0.5rem;
		font-size: 0.95rem;
	}
	.legend .small {
		margin: 0.4rem 0 0;
	}
	ul.leg {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	ul.leg li {
		display: flex;
		align-items: baseline;
		gap: 0.75rem;
		padding: 0.4rem 0;
		border-bottom: 1px solid var(--border);
	}
	ul.leg li span {
		flex: 1;
	}
	ul.leg li b {
		font-family: var(--font-mono);
		color: var(--accent);
		white-space: nowrap;
	}
	ol.tiebreak {
		margin: 0.5rem 0 0;
		padding-left: 1.3rem;
		line-height: 1.8;
	}
	ol.tiebreak li {
		padding-left: 0.3rem;
	}
</style>
