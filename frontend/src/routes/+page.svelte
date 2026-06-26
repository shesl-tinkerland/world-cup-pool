<script lang="ts">
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { tick } from 'svelte';
	import { auth } from '$lib/auth.svelte';
	import { homeIntro } from '$lib/homeIntro.svelte';
	import { featureBanner } from '$lib/featureBanner.svelte';
	import LeagueMatchCard from '$lib/components/LeagueMatchCard.svelte';
	import { api, type ChatOverviewItem, type LeagueProgress, type LeagueSummary, type LeaderboardRow } from '$lib/api';
	import { searchNav } from '$lib/searchNav.svelte';
	import { tipsStore, type Match, isLiveStatus, isLocked, teamsResolved } from '$lib/tips.svelte';
	import { forecastStore as fs, koKey } from '$lib/forecast.svelte';
	import { serverClock } from '$lib/serverclock.svelte';
	import { teamDisplayName } from '$lib/teamNames';
	import { language } from '$lib/language.svelte';
	import { strings } from '$lib/strings';
	import { matchStageLabel } from '$lib/stageLabels';
	import Avatar from '$lib/components/Avatar.svelte';
	import DeadlineCountdown from '$lib/components/DeadlineCountdown.svelte';
	import Flag from '$lib/components/Flag.svelte';
	import PendingInvites from '$lib/components/PendingInvites.svelte';
	import PublicLanding from '$lib/components/PublicLanding.svelte';
	import TvLogo from '$lib/components/TvLogo.svelte';
	import {
		Clock,
		Crown,
		Radio,
		Telescope,
		ListChecks,
		CheckCircle2,
		MessageCircle,
		ArrowUpRight,
		ChevronDown,
		Trophy,
		Volleyball,
		TrendingUp,
		Minus,
		X
	} from '@lucide/svelte';
	type NowHero = {
		tone: 'loading' | 'urgent' | 'forecast' | 'live' | 'result' | 'ready' | 'done' | 'gold';
		kicker: string;
		title: string;
		body: string;
		href: string;
		label: string;
		deadline?: string;
		deadlineLabel?: string;
		match?: Match;
		matches?: Match[];
		resultPoints?: number;
	};
	type HomeLeagueResult = {
		id: string;
		name: string;
		members: number;
		rank: number;
		total: number;
		medal: string;
		href: string;
	};

	const locale = $derived(language.locale);
	const introCopy = $derived(strings[language.resolved].introCard);
	let introCardOpen = $state(false);
	let introCardUser = $state('');

	// ----- Data load -------------------------------------------------------
	let leagues = $state<LeagueSummary[]>([]);
	let leaguesLoaded = $state(false);
	let lb = $state<LeaderboardRow[]>([]);
	let activeLeague = $state<LeagueSummary | null>(null);
	let leaderboards = $state<Record<string, LeaderboardRow[]>>({});
	let goldenBoots = $state<Record<string, import('$lib/api').GoldenBootLeagueTable>>({});
	let chatItems = $state<ChatOverviewItem[]>([]);
	let chatLoaded = $state(false);
	let chatError = $state(false);
	let leaguesError = $state(false);
	let leagueProgress = $state<LeagueProgress | null>(null);
	let homeLeagueRefreshTimer: ReturnType<typeof setTimeout> | null = null;
	let homeBackgroundLeaderboardsTimer: ReturnType<typeof setTimeout> | null = null;
	let homeLeagueRequestSeq = 0;
	let lastHomeScoreRevision = tipsStore.scoreRevision;
	let leaderboardFetchedAt = $state<Record<string, number>>({});

	const homeLeaderboardTtlMs = 30_000;

	async function refreshChatOverview() {
		chatError = false;
		try {
			const result = await api.chatOverview();
			chatItems = result.items.slice(0, 4);
		} catch {
			chatItems = [];
			chatError = true;
		} finally {
			chatLoaded = true;
		}
	}

	function clearHomeLeagueRefreshTimer() {
		if (!homeLeagueRefreshTimer) return;
		clearTimeout(homeLeagueRefreshTimer);
		homeLeagueRefreshTimer = null;
	}

	function clearHomeBackgroundLeaderboardsTimer() {
		if (!homeBackgroundLeaderboardsTimer) return;
		clearTimeout(homeBackgroundLeaderboardsTimer);
		homeBackgroundLeaderboardsTimer = null;
	}

	function applyHomeLeaderboard(
		leagueId: string,
		result: Awaited<ReturnType<typeof api.leaderboard>>
	) {
		leaderboards[leagueId] = result.rows;
		leaderboardFetchedAt[leagueId] = Date.now();
		if (result.goldenBoot) goldenBoots[leagueId] = result.goldenBoot;
		if (activeLeague?.id === leagueId) {
			lb = result.rows;
		}
	}

	function hasFreshHomeLeaderboard(leagueId: string) {
		const fetchedAt = leaderboardFetchedAt[leagueId] ?? 0;
		return fetchedAt > 0 && Date.now() - fetchedAt < homeLeaderboardTtlMs;
	}

	async function refreshHomeLeaderboard(leagueId: string, force = false) {
		if (!auth.isAuthed || !leagueId) return;
		if (!force && hasFreshHomeLeaderboard(leagueId)) {
			if (activeLeague?.id === leagueId) {
				lb = leaderboards[leagueId] ?? [];
			}
			return;
		}
		const result = await api.leaderboard(leagueId);
		applyHomeLeaderboard(leagueId, result);
	}

	async function refreshHomeLeagueData(
		leagueId = activeLeague?.id ?? '',
		options: { forceLeaderboard?: boolean } = {}
	) {
		if (!auth.isAuthed || !leagueId) return;
		const requestSeq = ++homeLeagueRequestSeq;

		try {
			await refreshHomeLeaderboard(leagueId, options.forceLeaderboard ?? false);
			if (requestSeq !== homeLeagueRequestSeq) return;
		} catch {
			// Keep the last known home table visible during transient network errors.
		}

		try {
			const result = await api.leagueProgress(leagueId);
			if (requestSeq !== homeLeagueRequestSeq) return;
			if (activeLeague?.id === leagueId) {
				leagueProgress = result;
			}
		} catch {
			// Keep the previous progress card; active-league changes still clear it.
		}
	}

	function queueHomeLeagueRefresh(leagueId = activeLeague?.id ?? '') {
		if (!leagueId) return;
		clearHomeLeagueRefreshTimer();
		homeLeagueRefreshTimer = setTimeout(() => {
			homeLeagueRefreshTimer = null;
			void refreshHomeLeagueData(leagueId, { forceLeaderboard: true });
		}, 500);
	}

	function queueOtherLeagueLeaderboards(activeLeagueId: string) {
		clearHomeBackgroundLeaderboardsTimer();
		homeBackgroundLeaderboardsTimer = setTimeout(() => {
			homeBackgroundLeaderboardsTimer = null;
			void refreshOtherLeagueLeaderboards(activeLeagueId);
		}, 1500);
	}

	async function refreshOtherLeagueLeaderboards(activeLeagueId: string) {
		for (const league of leagues) {
			if (league.id === activeLeagueId) continue;
			if (hasFreshHomeLeaderboard(league.id)) continue;
			try {
				await refreshHomeLeaderboard(league.id);
			} catch {
				// Background tables are best-effort; the active league stays prioritized.
			}
		}
	}

	$effect(() => {
		const leagueId = activeLeague?.id;
		if (!auth.isAuthed || !leagueId) {
			leagueProgress = null;
			return;
		}
		leagueProgress = null;
		void refreshHomeLeagueData(leagueId);
	});

	$effect(() => {
		const userId = auth.user?.id ?? '';
		if (!homeIntro.ready) return;
		if (introCardUser !== userId) {
			introCardUser = userId;
			introCardOpen = !!userId && homeIntro.visible;
		}
	});

	$effect(() => {
		if (!auth.isAuthed) return;
		const userId = auth.user?.id ?? '';
		if (!userId) return;
		leagues = [];
		leaguesLoaded = false;
		lb = [];
		activeLeague = null;
		leaderboards = {};
		goldenBoots = {};
		leaderboardFetchedAt = {};
		leagueProgress = null;
		chatLoaded = false;
		void refreshChatOverview();
		const chatTimer = setInterval(() => void refreshChatOverview(), 45_000);
		leaguesError = false;
		api
			.myLeagues()
			.then((r) => {
				leagues = r.leagues;
				const current =
					activeLeague && r.leagues.find((l) => l.id === activeLeague?.id);
				const storedLeagueId = readStoredLeagueId();
				const stored =
					storedLeagueId ? r.leagues.find((l) => l.id === storedLeagueId) : null;
				const pref =
					current ??
					stored ??
					r.leagues.find((l) => l.inviteCode !== 'GLOBAL') ??
					r.leagues.find((l) => l.inviteCode === 'GLOBAL') ??
					r.leagues[0];
				if (pref) {
					selectLeague(pref.id, false);
					queueOtherLeagueLeaderboards(pref.id);
				}
				leaguesLoaded = true;
			})
			.catch(() => {
				leaguesError = true;
				leaguesLoaded = true;
			});
		return () => {
			clearInterval(chatTimer);
			clearHomeLeagueRefreshTimer();
			clearHomeBackgroundLeaderboardsTimer();
			homeLeagueRequestSeq++;
		};
	});

	$effect(() => {
		const revision = tipsStore.scoreRevision;
		const leagueId = activeLeague?.id ?? '';
		if (!auth.isAuthed || !leaguesLoaded || !leagueId) {
			lastHomeScoreRevision = revision;
			return;
		}
		if (revision === lastHomeScoreRevision) return;
		lastHomeScoreRevision = revision;
		queueHomeLeagueRefresh(leagueId);
	});

	$effect(() => {
		const leagueId = activeLeague?.id ?? '';
		const hasLiveMatches = tipsStore.liveMatchIds.size > 0;
		if (!auth.isAuthed || !leaguesLoaded || !leagueId || !hasLiveMatches) return;
		queueHomeLeagueRefresh(leagueId);
		const timer = setInterval(
			() => void refreshHomeLeagueData(leagueId, { forceLeaderboard: true }),
			60_000
		);
		return () => clearInterval(timer);
	});

	// ----- Derived ---------------------------------------------------------
	let now = $derived(serverClock.now());

	function playedM(m: Match) {
		return m.status === 'finished' || !!m.finalizedAt;
	}

	let totalMatches = $derived(tipsStore.matches.length);
	let tournamentStarted = $derived(
		tipsStore.matches.some((m) => new Date(m.kickoff).getTime() <= now)
	);

	let upcoming = $derived(
		tipsStore.matches
			.filter(
				(m) =>
					new Date(m.kickoff).getTime() > now &&
					!isLiveStatus(m.status) &&
					!playedM(m)
			)
			.sort(
				(a, b) =>
					new Date(a.kickoff).getTime() - new Date(b.kickoff).getTime()
			)
	);
	let nextMatchesPreview = $derived(upcoming.slice(0, 3));
	let nextMatch = $derived(upcoming[0]);
	let openMatchTips = $derived(
		upcoming.filter((m) => teamsResolved(m) && !isLocked(m))
	);
	let missingMatchTips = $derived(
		openMatchTips.filter((m) => !tipsStore.tips[m.id])
	);
	let vmTipsMissing = $derived(fs.loaded && !fs.locked && (!fs.recId || !fs.isComplete));
	let missingTaskCount = $derived(
		missingMatchTips.length + (vmTipsMissing ? 1 : 0)
	);
	let openMatchTipCount = $derived(openMatchTips.length);
	let submittedOpenMatchTipCount = $derived(
		openMatchTipCount > 0 ? openMatchTipCount - missingMatchTips.length : 0
	);
	let progressPct = $derived(
		openMatchTipCount > 0
			? Math.round((submittedOpenMatchTipCount / openMatchTipCount) * 100)
			: 0
	);
	let recentResults = $derived(
		[...tipsStore.matches]
			.filter(playedM)
			.sort(
				(a, b) =>
					new Date(b.kickoff).getTime() - new Date(a.kickoff).getTime()
			)
			.slice(0, 3)
	);
	let homeAvatars = $derived(
		Object.fromEntries(lb.map((r) => [r.userId, r.avatarUrl ?? '']))
	);
	let activeLeagueRow = $derived(
		lb.find((r) => r.userId === auth.user?.id) ?? null
	);
	let totalPoints = $derived(activeLeagueRow?.total ?? 0);
	let myRank = $derived(
		activeLeagueRow ? lb.findIndex((r) => r.userId === auth.user?.id) + 1 : 0
	);
	let personAbove = $derived(myRank > 1 ? (lb[myRank - 2] ?? null) : null);
	let personBelow = $derived(myRank > 0 ? (lb[myRank] ?? null) : null);
	let gapToAbove = $derived(personAbove ? personAbove.total - totalPoints : 0);
	let gapToBelow = $derived(personBelow ? totalPoints - personBelow.total : 0);

	let leagueHref = $derived(activeLeague ? `/leagues/${activeLeague.id}` : '/leagues');
	let globalLeague = $derived(leagues.find((l) => l.inviteCode === 'GLOBAL'));
	let pointsHref = $derived(globalLeague ? `/leagues/${globalLeague.id}` : leagueHref);

	function kickoffLabel(iso: string) {
		const d = new Date(iso);
		const today = new Date();
		const sameDay =
			d.getFullYear() === today.getFullYear() &&
			d.getMonth() === today.getMonth() &&
			d.getDate() === today.getDate();
		const time = d.toLocaleTimeString(locale, {
			hour: '2-digit',
			minute: '2-digit'
		});
		if (sameDay) return `${language.text('I dag', 'I dag', 'Today')}, ${time}`;
		return (
			d.toLocaleDateString(locale, {
				weekday: 'short',
				day: 'numeric',
				month: 'short'
			}) +
			', ' +
			time
		);
	}

	function greeting() {
		const h = new Date().getHours();
		if (h < 6) return language.text('God natt', 'God natt', 'Good night');
		if (h < 11) return language.text('God morgen', 'God morgon', 'Good morning');
		if (h < 17) return language.text('Hei', 'Hei', 'Hi');
		if (h < 22) return language.text('God kveld', 'God kveld', 'Good evening');
		return language.text('God natt', 'God natt', 'Good night');
	}

	function firstName(name: string) {
		return name.trim().split(/\s+/)[0] ?? '';
	}
	function initials(name: string) {
		return name
			.split(/\s+/)
			.filter(Boolean)
			.slice(0, 2)
			.map((part) => part[0]?.toUpperCase() ?? '')
			.join('');
	}

	function team(id: string) {
		return tipsStore.team(id);
	}
	function dismissIntroCard() {
		introCardOpen = false;
		homeIntro.dismiss();
	}
	function teamAny(id: string) {
		return tipsStore.team(id) ?? fs.team(id);
	}
	function teamLabel(m: Match, side: 'h' | 'a') {
		const t = side === 'h' ? team(m.homeTeam) : team(m.awayTeam);
		return teamDisplayName(t, side === 'h' ? m.homeLabel : m.awayLabel);
	}
	function scoreText(m: Match) {
		let s = `${m.ftHome}–${m.ftAway}`;
		if (m.etHome || m.etAway) s = `${m.etHome}–${m.etAway} ${language.text('e.e.o.', 'e.eo.', 'aet')}`;
		if (m.penHome || m.penAway) s += ` (${m.penHome}–${m.penAway} ${language.text('str', 'str', 'pens')})`;
		return s;
	}
	function chatTimeLabel(iso: string) {
		const then = new Date(iso).getTime();
		if (!Number.isFinite(then)) return '';
		const diff = now - then;
		if (diff < 60_000) return language.text('nå', 'no', 'now');
		if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} min`;
		if (diff < 86_400_000) return new Intl.DateTimeFormat(locale, { hour: '2-digit', minute: '2-digit' }).format(then);
		return new Intl.DateTimeFormat(locale, { day: '2-digit', month: 'short' }).format(then);
	}
	function chatPreview(text: string) {
		return text.length > 82 ? `${text.slice(0, 79).trim()}…` : text;
	}
	function unreadLabel(count: number) {
		if (count <= 0) return '';
		return count === 1 ? language.text('Ny', 'Ny', 'New') : `${Math.min(count, 99)} ${language.text('nye', 'nye', 'new')}`;
	}
	function stageLabel(match: Match) {
		return matchStageLabel(match);
	}
	function matchTipHref(match: Match) {
		return `/tips?match=${encodeURIComponent(match.id)}`;
	}
	function missingMatchTipHref(match: Match) {
		return `/tips?tab=missing&match=${encodeURIComponent(match.id)}`;
	}
	async function scrollAfterTipsNavigation(href: string) {
		if (!browser) return;
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
	async function followTipsLink(event: MouseEvent, href: string) {
		if (!href.startsWith('/tips?')) return;
		event.preventDefault();
		searchNav.bump();
		await goto(href, { keepFocus: true, noScroll: true });
		await scrollAfterTipsNavigation(href);
	}
	function leagueSelectionKey() {
		return auth.user?.id ? `home-league-v1:${auth.user.id}` : '';
	}
	function readStoredLeagueId() {
		if (!browser) return '';
		const key = leagueSelectionKey();
		if (!key) return '';
		try {
			return localStorage.getItem(key) ?? '';
		} catch {
			return '';
		}
	}
	function rememberLeagueSelection(leagueId: string) {
		if (!browser) return;
		const key = leagueSelectionKey();
		if (!key) return;
		try {
			localStorage.setItem(key, leagueId);
		} catch {
			/* localStorage can be unavailable in private mode */
		}
	}
	function selectLeague(leagueId: string, remember = true) {
		const league = leagues.find((l) => l.id === leagueId);
		if (!league) return;
		activeLeague = league;
		lb = leaderboards[league.id] ?? [];
		if (remember) rememberLeagueSelection(league.id);
	}
	function onLeagueSelect(event: Event) {
		selectLeague((event.currentTarget as HTMLSelectElement).value);
	}
	function resultTeams(match: Match) {
		return `${teamLabel(match, 'h')} - ${teamLabel(match, 'a')}`;
	}
	function tipText(match: Match) {
		const tip = tipsStore.tips[match.id];
		if (!tip) return language.text('Ikke tippet', 'Ikkje tipset', 'Not tipped');
		return `${language.text('Ditt tips', 'Ditt tips', 'Your tip')}: ${tip.ftHome}-${tip.ftAway}`;
	}
	function matchTipsMissingText(count: number) {
		return language.text(
			`${count} kamptips mangler`,
			`${count} kamptips manglar`,
			`${count} match tip${count === 1 ? '' : 's'} missing`
		);
	}
	function lastMatchTitlePrefix(points: number) {
		if (points === 6) {
			return language.text(
				'Perfekt! ',
				'Perfekt! ',
				'Perfect! '
			);
		}
		return language.text(
			'Du fikk ',
			'Du fekk ',
			'You got '
		);
	}
	function lastMatchTitleSuffix(points: number) {
		if (points === 6) {
			return language.text(
				' på siste kamp',
				' på siste kamp',
				' on the last match'
			);
		}
		return language.text(
			' på siste kamp',
			' på siste kamp',
			' on the last match'
		);
	}
	function forecastPointsText(points: number) {
		return language.text(`${points} VM-tips-poeng`, `${points} VM-tips-poeng`, `${points} forecast points`);
	}
	function shortPoints(points: number, signed = false) {
		const prefix = signed && points > 0 ? '+' : '';
		return language.text(`${prefix}${points} p`, `${prefix}${points} p`, `${prefix}${points} pts`);
	}
	function medalForRank(rank: number) {
		if (rank === 1) return '🥇';
		if (rank === 2) return '🥈';
		if (rank === 3) return '🥉';
		return `#${rank}`;
	}
	function rankSummary(rank: number, members: number) {
		return language.text(`#${rank} av ${members}`, `#${rank} av ${members}`, `#${rank} of ${members}`);
	}
	function formatLeagueList(names: string[]) {
		if (names.length === 0) return '';
		if (names.length === 1) return names[0];
		const conjunction = language.text('og', 'og', 'and');
		if (names.length === 2) return `${names[0]} ${conjunction} ${names[1]}`;
		return `${names.slice(0, -1).join(', ')} ${conjunction} ${names.at(-1) ?? ''}`;
	}
	function pointText(points: number) {
		if (points === 6) return language.text(`Perfekt! ${shortPoints(points, true)}`, `Perfekt! ${shortPoints(points, true)}`, `Perfect! ${shortPoints(points, true)}`);
		if (points > 0) return shortPoints(points, true);
		return shortPoints(points);
	}
	function teamStillAlive(id: string) {
		if (!id) return false;
		if (!tournamentStarted) return true;
		if (tournamentFinished) return id === realChampionId;
		return tipsStore.matches.some(
			(match) => !playedM(match) && (match.homeTeam === id || match.awayTeam === id)
		);
	}

	let liveMatches = $derived(tipsStore.matches.filter((m) => isLiveStatus(m.status)));
	let latestResult = $derived(recentResults[0] ?? null);
	let unreadChatItems = $derived(chatItems.filter((item) => item.unread > 0));
	let recentResultWithPoints = $derived.by(() => {
		const match = latestResult;
		if (!match) return null;
		return { match, points: tipsStore.scores[match.id] ?? 0 };
	});
	let nowHero = $derived.by<NowHero>(() => {
		if (!tipsStore.loaded || !fs.loaded) {
			return {
				tone: 'loading',
				kicker: language.text('Akkurat nå', 'Akkurat no', 'Right now'),
				title: language.text('Sjekker tipsene dine', 'Sjekkar tipsa dine', 'Checking your tips'),
				body: language.text(
					'Henter kampstatus, poeng og liga.',
					'Hentar kampstatus, poeng og liga.',
					'Fetching match status, points, and league.'
				),
				href: '/tips',
				label: language.text('Åpne kamptips', 'Opne kamptips', 'Open match tips')
			};
		}
		if (liveMatches.length > 0) {
			const first = liveMatches[0];
			return {
				tone: 'live',
				kicker: language.text('Live nå', 'Live no', 'Live now'),
				title: liveMatches.length === 1
					? language.text(
						`${resultTeams(first)} spilles nå`,
						`${resultTeams(first)} spelast no`,
						`${resultTeams(first)} is playing now`
					)
					: language.text(
						`${liveMatches.length} kamper pågår`,
						`${liveMatches.length} kampar pågår`,
						`${liveMatches.length} matches in progress`
					),
				body: liveMatches.length === 1
					? `${stageLabel(first)}${first.tvChannel ? ` · ${language.text('på TV', 'på TV', 'on TV')}` : ''}`
					: language.text(
						'Følg kampene i sanntid',
						'Følg kampane i sanntid',
						'Follow the matches in real time'
					),
				href: matchTipHref(first),
				label: liveMatches.length === 1
					? language.text('Se kamp', 'Sjå kamp', 'View match')
					: language.text('Se kamper', 'Sjå kampar', 'View matches'),
				match: first,
				matches: liveMatches
			};
		}
		if (missingMatchTips.length > 0) {
			const count = missingMatchTips.length;
			const match = missingMatchTips[0];
			return {
				tone: 'urgent',
				kicker: language.text('Neste steg', 'Neste steg', 'Next action'),
				title: matchTipsMissingText(count),
				body: count === 1
					? language.text('Dette tipset må leveres før avspark.', 'Dette tipset må leverast før avspark.', 'This tip must be submitted before kickoff.')
					: language.text('De siste kamptipsene må leveres før avspark.', 'Dei siste kamptipsa må leverast før avspark.', 'The remaining match tips must be submitted before kickoff.'),
				href: missingMatchTipHref(match),
				label: count === 1
					? language.text('Tipp kampen', 'Tipp kampen', 'Tip the match')
					: language.text('Gå til kamptips', 'Gå til kamptips', 'Go to match tips'),
				deadline: match.kickoff,
				deadlineLabel: language.text('Frist', 'Frist', 'Deadline'),
				match
			};
		}
		if (vmTipsMissing) {
			return {
				tone: 'forecast',
				kicker: language.text('Neste steg', 'Neste steg', 'Next action'),
				title: language.text('VM-tipset må leveres før avspark', 'VM-tipset må leverast før avspark', 'The World Cup tip must be submitted before kickoff'),
				body: language.text(
					'Sett grupper, beste treere og sluttspill før turneringen starter.',
					'Set grupper, beste trearar og sluttspel før turneringa startar.',
					'Set groups, best thirds, and knockout before the tournament starts.'
				),
				href: '/forecast',
				label: language.text('Åpne VM-tips', 'Opne VM-tips', 'Open World Cup tips'),
				deadline: fs.tournamentStart,
				deadlineLabel: language.text('Låses', 'Låsast', 'Locks')
			};
		}
		if (tournamentFinished) {
			const winnerCount = wonLeaguePlacements.length;
			const isGold = winnerCount > 0;
			return {
				tone: isGold ? 'gold' : 'done',
				kicker: language.text('Turneringen er over', 'Turneringa er over', 'Tournament over'),
				title: isGold
					? language.text(`Du vant ${winnerCount} ${winnerCount === 1 ? 'liga' : 'ligaer'} 🥇🏆`, `Du vann ${winnerCount} ${winnerCount === 1 ? 'liga' : 'ligaer'} 🥇🏆`, `You won ${winnerCount} ${winnerCount === 1 ? 'league' : 'leagues'} 🥇🏆`)
					: language.text('VM er over 🎊🏆', 'VM er over 🎊🏆', 'World Cup is over 🎊🏆'),
				body: isGold 
					? language.text('Sterk tipping! Se sluttresultatene dine under.', 'Fantastisk tipping! Sjå sluttresultata dine under.', 'Fantastic predictions! Your final standings are below.')
					: activeLeagueRow
					? language.text('Sluttresultatene dine i ligaene ligger klare under.', 'Sluttresultata dine i ligaene ligg klare under.', 'Your final league standings are ready below.')
					: language.text('Takk for at du spilte. Sluttresultatene dine ligger klare under.', 'Takk for at du spelte. Sluttresultata dine ligg klare under.', 'Thanks for playing. Your final standings are ready below.'),
				href: leagueHref,
				label: language.text('Se sluttresultat', 'Sjå sluttresultat', 'View standings')
			};
		}
		if (recentResultWithPoints) {
			const { match, points } = recentResultWithPoints;
			return {
				tone: 'result',
				kicker: language.text('Siste resultat', 'Siste resultat', 'Latest result'),
				title: '',
				resultPoints: points,
				body: tipText(match),
				href: matchTipHref(match),
				label: language.text('Se resultat', 'Sjå resultat', 'View result'),
				match
			};
		}
		return {
			tone: 'ready',
			kicker: language.text('Alt klart', 'Alt klart', 'All set'),
			title: nextMatch
				? language.text(`Klart. Neste kamp er ${resultTeams(nextMatch)}`, `Klart. Neste kamp er ${resultTeams(nextMatch)}`, `Ready. Next match is ${resultTeams(nextMatch)}`)
				: language.text('Klart', 'Klart', 'Ready'),
			body: nextMatch
				? kickoffLabel(nextMatch.kickoff)
				: language.text('Du er klar for turneringen.', 'Du er klar for turneringa.', 'You are ready for the tournament.'),
			href: nextMatch ? matchTipHref(nextMatch) : '/tips',
			label: language.text('Se kamper', 'Sjå kampar', 'View matches'),
			match: nextMatch
		};
	});
	let finalMatch = $derived(fs.knockout.find((m) => m.stage === 'FINAL'));
	let bronzeMatch = $derived(fs.knockout.find((m) => m.stage === '3RD'));
	let championId = $derived(finalMatch ? (fs.bracket[koKey(finalMatch)] ?? '') : '');
	let runnerUpId = $derived.by(() => {
		if (!finalMatch || !championId) return '';
		const [homeId, awayId] = fs.sides(finalMatch);
		if (championId === homeId) return awayId;
		if (championId === awayId) return homeId;
		return '';
	});
	let thirdId = $derived(bronzeMatch ? (fs.bracket[koKey(bronzeMatch)] ?? '') : '');
	let podium = $derived([
		{ place: 1, label: language.text('Vinner', 'Vinnar', 'Winner'), id: championId },
		{ place: 2, label: language.text('Andreplass', 'Andreplass', 'Runner-up'), id: runnerUpId },
		{ place: 3, label: language.text('Tredjeplass', 'Tredjeplass', 'Third place'), id: thirdId }
	]);
	let hasPodium = $derived(podium.some((p) => !!p.id));

	let realFinalMatch = $derived(tipsStore.matches.find((m) => m.stage === 'FINAL'));
	let realBronzeMatch = $derived(tipsStore.matches.find((m) => m.stage === '3RD'));
	let tournamentFinished = $derived(realFinalMatch && playedM(realFinalMatch));
	let realChampionId = $derived(tournamentFinished && realFinalMatch ? realFinalMatch.advancer : '');
	let realRunnerUpId = $derived.by(() => {
		if (!tournamentFinished || !realFinalMatch || !realChampionId) return '';
		return realChampionId === realFinalMatch.homeTeam ? realFinalMatch.awayTeam : realFinalMatch.homeTeam;
	});
	let realThirdId = $derived(tournamentFinished && realBronzeMatch ? realBronzeMatch.advancer : '');
	let realPodiumParams = $derived([
		{ place: 1, label: language.text('VM-gull', 'VM-gull', 'World Cup gold'), id: realChampionId },
		{ place: 2, label: language.text('VM-sølv', 'VM-sølv', 'World Cup silver'), id: realRunnerUpId },
		{ place: 3, label: language.text('VM-bronse', 'VM-bronse', 'World Cup bronze'), id: realThirdId }
	]);

	let forecastPulse = $derived.by(() => {
		const champion = championId ? teamDisplayName(teamAny(championId), language.text('Ukjent', 'Ukjent', 'Unknown')) : '';
		if (!fs.loaded) {
			return {
				kicker: language.text('VM-tips', 'VM-tips', 'World Cup tip'),
				title: language.text('Laster VM-tipset ditt', 'Lastar VM-tipset ditt', 'Loading your World Cup tip'),
				body: language.text('Vi sjekker grupper, sluttspill og pall.', 'Vi sjekkar grupper, sluttspel og pall.', 'We are checking groups, knockout, and podium.'),
				label: language.text('Åpne VM-tips', 'Opne VM-tips', 'Open World Cup tips'),
				tone: 'loading'
			};
		}
		if (vmTipsMissing) {
			return {
				kicker: language.text('VM-tips', 'VM-tips', 'World Cup tip'),
				title: language.text('VM-tipset må leveres før avspark', 'VM-tipset må leverast før avspark', 'The World Cup tip must be submitted before kickoff'),
				body: language.text('Fyll ut grupper, beste treere og sluttspill.', 'Fyll ut grupper, beste trearar og sluttspel.', 'Enter groups, best thirds, and knockout.'),
				label: language.text('Lever VM-tips', 'Lever VM-tips', 'Submit World Cup tip'),
				tone: 'urgent'
			};
		}
		if (tournamentFinished) {
			return {
				kicker: language.text('VM-tips', 'VM-tips', 'World Cup tip'),
				title: forecastPointsText(activeLeagueRow?.forecastPoints ?? 0),
				body: champion
					? language.text(`Du hadde ${champion} som vinner.`, `Du hadde ${champion} som vinnar.`, `You had ${champion} as winner.`)
					: language.text('Turneringen er ferdig.', 'Turneringa er ferdig.', 'The tournament is finished.'),
				label: language.text('Se VM-tips', 'Sjå VM-tips', 'View World Cup tips'),
				tone: 'done'
			};
		}
		if (!tournamentStarted) {
			return {
				kicker: language.text('VM-tips', 'VM-tips', 'World Cup tip'),
				title: champion
					? language.text(`${champion} er vinneren din`, `${champion} er vinnaren din`, `${champion} is your winner`)
					: language.text('VM-tipset ditt er klart', 'VM-tipset ditt er klart', 'Your World Cup tip is ready'),
				body: language.text('Pallen din låses når turneringen starter.', 'Pallen din blir låst når turneringa startar.', 'Your podium is locked in when the tournament starts.'),
				label: language.text('Se VM-tips', 'Sjå VM-tips', 'View World Cup tips'),
				tone: 'ready'
			};
		}
		const alive = teamStillAlive(championId);
		return {
			kicker: fs.groupStageDone
				? language.text('VM-tips ', 'VM-tips ', 'World Cup tip ')
				: language.text('VM-tips ', 'VM-tips ', 'World Cup tip '),
			title: champion
				? alive
					? language.text('Ditt tips er med', 'Ditt tips er med', 'Your guess is alive')
					: language.text('Vinneren din er ute', 'Vinnaren din er ute', 'Your winner is out')
				: language.text('Er i gang', 'Er i gang', 'In play'),
			body: champion
				? `${champion}${activeLeagueRow ? ` · ${forecastPointsText(activeLeagueRow.forecastPoints)}` : ''}`
				: language.text('Følg grupper og sluttspill etter hvert som resultatene kommer.', 'Følg grupper og sluttspel etter kvart som resultata kjem.', 'Follow groups and knockout as results come in.'),
			label: language.text('Se VM-tips', 'Sjå VM-tips', 'View World Cup tips'),
			tone: alive ? 'ready' : 'out'
		};
	});

	let activeGoldenBoot = $derived.by(() => {
		if (!activeLeague) return null;
		return goldenBoots[activeLeague.id] ?? null;
	});
	let myGoldenBootPlayerId = $derived(fs.loaded ? fs.goldenBootPlayer : null);
	let myGoldenBootPick = $derived.by(() => {
		if (!myGoldenBootPlayerId || !activeGoldenBoot) return null;
		return activeGoldenBoot.players.find((player) => player.id === myGoldenBootPlayerId) ?? null;
	});
	let myGoldenBootPickIsWinner = $derived.by(
		() => (myGoldenBootPick?.rank ?? 0) === 1 && (myGoldenBootPick?.goals ?? 0) > 0
	);
	let goldenBootLeaders = $derived.by(() => {
		const players = activeGoldenBoot?.players ?? [];
		return players
			.filter((player) => player.rank > 0)
			.sort(
				(a, b) =>
					a.rank - b.rank ||
					b.goals - a.goals ||
					b.assists - a.assists ||
					a.name.localeCompare(b.name, locale)
			)
			.slice(0, 3);
	});
	let myGoldenBootPickInLeaders = $derived.by(
		() => !!myGoldenBootPick && goldenBootLeaders.some((player) => player.id === myGoldenBootPick.id)
	);

	let miniLeaderboard = $derived.by(() => {
		const top3 = lb.slice(0, 3).map((r, i) => ({ ...r, rank: i + 1 }));
		const myIndex = lb.findIndex((r) => r.userId === auth.user?.id);
		if (myIndex > 2) {
			top3.push({ ...lb[myIndex], rank: myIndex + 1 });
		}
		return top3;
	});

	let finalLeaguePlacements = $derived.by(() => {
		return leagues
			.map((league): HomeLeagueResult | null => {
			const rows = leaderboards[league.id] ?? [];
				const myIndex = rows.findIndex((row) => row.userId === auth.user?.id);
				if (myIndex < 0) return null;
				const row = rows[myIndex];
				return {
					id: league.id,
					name: league.name,
					members: league.members,
					rank: myIndex + 1,
					total: row.total,
					medal: medalForRank(myIndex + 1),
					href: `/leagues/${league.id}`
				};
			})
			.filter((league): league is HomeLeagueResult => league !== null)
			.sort(
				(a, b) =>
					a.rank - b.rank ||
					b.total - a.total ||
					a.name.localeCompare(b.name, locale)
			);
	});

	let wonLeaguePlacements = $derived(
		finalLeaguePlacements.filter((league) => league.rank === 1)
	);
	let leagueFinishSummary = $derived.by(() => {
		if (wonLeaguePlacements.length === 1) {
			const [league] = wonLeaguePlacements;
			return {
				title: language.text(`Gratulerer! Du vant ${league.name}`, `Gratulerer! Du vann ${league.name}`, `Congratulations! You won ${league.name}`),
				body: language.text(`Du endte øverst med ${shortPoints(league.total)}.`, `Du enda øvst med ${shortPoints(league.total)}.`, `You finished first with ${shortPoints(league.total)}.`)
			};
		}
		if (wonLeaguePlacements.length > 1) {
			const leagueNames = formatLeagueList(
				wonLeaguePlacements.map((league) => league.name)
			);
			return {
				title: language.text(
					`Gratulerer! Du vant ${wonLeaguePlacements.length} ligaer`,
					`Gratulerer! Du vann ${wonLeaguePlacements.length} ligaer`,
					`Congratulations! You won ${wonLeaguePlacements.length} leagues`
				),
				body: language.text(`Du vant ${leagueNames}.`, `Du vann ${leagueNames}.`, `Wins in ${leagueNames}.`)
			};
		}
		const podiumPlacements = finalLeaguePlacements.filter(
			(league) => league.rank > 1 && league.rank <= 3
		);
		if (podiumPlacements.length > 0) {
			const leagueNames = formatLeagueList(
				podiumPlacements.map((league) => league.name)
			);
			return {
				title: language.text('Turneringen er ferdig', 'Turneringa er ferdig', 'Tournament finished'),
				body: language.text(`Du tok pallplass i ${leagueNames}.`, `Du tok pallplass i ${leagueNames}.`, `You reached the podium in ${leagueNames}.`)
			};
		}
		const bestLeague = finalLeaguePlacements[0];
		if (!bestLeague) {
			return {
				title: language.text('Turneringen er ferdig', 'Turneringa er ferdig', 'Tournament finished'),
				body: language.text('Ligaplasseringene dine vises her.', 'Ligaplaceringane dine kjem her.', 'Your league standings will show here.')
			};
		}
		return {
			title: language.text('Turneringen er ferdig', 'Turneringa er ferdig', 'Tournament finished'),
			body: language.text(`Beste plassering: #${bestLeague.rank} i ${bestLeague.name}.`, `Beste plassering: #${bestLeague.rank} i ${bestLeague.name}.`, `Best finish: #${bestLeague.rank} in ${bestLeague.name}.`)
		};
	});

	let heroLeagues = $derived.by(() => {
		if (!leaguesLoaded) return [];

		const withStats = leagues.map((lg) => {
			const lb = leaderboards[lg.id] ?? [];
			const myIndex = lb.findIndex((r) => r.userId === auth.user?.id);
			const meRow = myIndex >= 0 ? lb[myIndex] : null;
			const rankNum = myIndex >= 0 ? myIndex + 1 : 999999;
			const total = meRow ? meRow.total : -1;
			const loading = !leaderboardFetchedAt[lg.id];
			return { lg, rankNum, total, meRow, loading };
		});
		
		withStats.sort((a, b) => a.rankNum - b.rankNum || b.total - a.total);
		
		const topLeagues = [];
				if (activeLeague) {
			const activeIndex = withStats.findIndex(s => s.lg.id === activeLeague?.id);
			if (activeIndex >= 0) {
				topLeagues.push(withStats.splice(activeIndex, 1)[0]);
			}
		}
		
		while (topLeagues.length < 3 && withStats.length > 0) {
			topLeagues.push(withStats.shift()!);
		}

		topLeagues.sort((a, b) => a.rankNum - b.rankNum || b.total - a.total);
		return topLeagues;
	});

	// Per-match form strip for the "Poengtrend" card: one bar per recent match, height
	// = points won that match (gold = perfect tip, muted = zero). Shows form/streaks.
	let formBars = $derived.by(() => {
		const events = leagueProgress?.events ?? [];
		if (events.length === 0) return null;
		const sorted = [...events].sort(
			(a, b) => new Date(a.kickoff).getTime() - new Date(b.kickoff).getTime()
		);
		const recent = sorted.slice(-16);
		const scale = Math.max(6, ...recent.map((e) => e.points));
		return recent.map((e) => {
			const home = team(e.homeTeam)?.fifaCode ?? e.homeLabel;
			const away = team(e.awayTeam)?.fifaCode ?? e.awayLabel;
			return {
				matchId: e.matchId,
				points: e.points,
				exact: e.exact,
				height: e.points > 0 ? Math.max(0.16, e.points / scale) : 0.1,
				label: `${home} ${e.ftHome}–${e.ftAway} ${away} · ${e.points} p`
			};
		});
	});

	// Aggregate tipping stats for the trend card — all-time, from the league summary.
	let progressStats = $derived.by(() => {
		const s = leagueProgress?.summary;
		if (!s) return null;
		return {
			total: s.tipsPoints,
			last3: s.last5Points,
			exact: s.exactScores,
			avg: s.finishedMatches > 0 ? s.tipsPoints / s.finishedMatches : 0,
			hitRate: s.tippedFinished > 0 ? Math.round((s.matchesWithPoints / s.tippedFinished) * 100) : 0,
			best: s.bestPoints
		};
	});

</script>

{#if !auth.isAuthed}
	<PublicLanding />
{:else}
<header class="home-hero">
	<div class="hero-copy">
		<p class="kicker">{language.text('VM 2026 · 11. juni - 19. juli', 'VM 2026 · 11. juni - 19. juli', 'World Cup 2026 · 11 June - 19 July')}</p>
		<h1 class="hero-greeting">
			<span class="greet">{greeting()}</span>{#if firstName(auth.user?.name ?? '')}<span class="punct">,</span> <span class="name">{firstName(auth.user?.name ?? '')}</span>{/if}
		</h1>
	</div>
	<div class="hero-chips">
		{#if leaguesLoaded}
			{#if leaguesError}
				<span class="hero-chip error-pill">{language.text('Ligastatus mangler', 'Ligastatus manglar', 'League status missing')}</span>
			{/if}
			{#each heroLeagues as { lg, rankNum, meRow, loading } (lg.id)}
				{@const medal = rankNum === 1 ? '🥇' : rankNum === 2 ? '🥈' : rankNum === 3 ? '🥉' : `#${rankNum}`}
				<a href={`/leagues/${lg.id}`} class="hero-chip league-pill" class:loading={loading} class:mobile-hide={heroLeagues.length > 1 && lg.id !== activeLeague?.id}>
					<span>{lg.name}</span>
					{#if meRow && rankNum > 0 && rankNum !== 999999}
						<b>{medal}</b>
					{:else if loading}
						<i class="league-rank-skeleton" aria-hidden="true"></i>
					{/if}
				</a>
			{/each}
		{/if}
		{#if activeLeagueRow}
			<a href={pointsHref} class="hero-chip points-pill">
				<span>{language.text('Poeng', 'Poeng', 'Points')}</span>
				<b>{totalPoints}</b>
			</a>
		{/if}
	</div>
</header>

<div class="bento home-bento stagger">
	{#if featureBanner.visible}
		<div class="home-feature-banner">
			<a class="hfb-main" href={`${leagueHref}?view=matches`} onclick={() => featureBanner.dismiss()}>
				<span class="hfb-badge">{language.text('Nytt', 'Nytt', 'New')}</span>
				<span class="hfb-copy">
					<b>{language.text('Se kampene i ligaen', 'Sjå kampane i ligaen', 'See the league matches')}</b>
					<span class="muted">{language.text('Kamp for kamp: hva alle tippet og hvem som fikk poeng.', 'Kamp for kamp: kva alle tippa og kven som fekk poeng.', 'Match by match: what everyone tipped and who scored points.')}</span>
				</span>
				<ArrowUpRight size={18} class="hfb-go" />
			</a>
			<button class="hfb-x" onclick={() => featureBanner.dismiss()} aria-label={language.text('Lukk', 'Lukk', 'Dismiss')}><X size={16} /></button>
		</div>
	{/if}

	{#if homeIntro.ready && introCardOpen}
		<section class="card tile intro-card home-span-primary" aria-labelledby="home-intro-title">
			<div class="intro-head">
				<div class="intro-copy">
					<span class="kicker">{introCopy.kicker}</span>
					<h2 id="home-intro-title">{introCopy.title}</h2>
					<p class="muted">{introCopy.body}</p>
				</div>
				<button
					type="button"
					class="intro-dismiss"
					aria-label={introCopy.close}
					onclick={dismissIntroCard}
				>
					<X size={16} />
				</button>
			</div>

			<div class="intro-grid">
				<article class="intro-pill">
					<span class="intro-pill-icon leagues"><Trophy size={18} /></span>
					<div>
						<b>{introCopy.leaguesTitle}</b>
						<p>{introCopy.leaguesBody}</p>
					</div>
				</article>

				<article class="intro-pill">
					<span class="intro-pill-icon match-tips"><Volleyball size={18} /></span>
					<div>
						<b>{introCopy.matchTipsTitle}</b>
						<p>{introCopy.matchTipsBody}</p>
					</div>
				</article>

				<article class="intro-pill">
					<span class="intro-pill-icon worldcup"><Telescope size={18} /></span>
					<div>
						<b>{introCopy.worldCupTipsTitle}</b>
						<p>{introCopy.worldCupTipsBody}</p>
					</div>
				</article>
			</div>

			<div class="intro-actions">
				<div class="intro-links">
					<a class="btn" href="/leagues">{introCopy.primaryCta}</a>
					<a class="btn secondary" href="/tips" onclick={(event) => void followTipsLink(event, '/tips')}>
						{introCopy.secondaryCta}
					</a>
				</div>
			</div>
		</section>
	{/if}

	<PendingInvites homeTile />

	<!-- Main task panel: what matters right now. -->
	<section
		class="card tasks tile action-card home-span-primary"
		class:done={tipsStore.loaded && fs.loaded && missingTaskCount === 0}
		class:needs-tips={missingMatchTips.length > 0}
		class:live={nowHero.tone === 'live'}
		class:result={nowHero.tone === 'result'}
		class:tourney-over={nowHero.tone === 'done'}
		class:tourney-gold={nowHero.tone === 'gold'}
	>
		<div class="now-topline">
			<span class="kicker">{nowHero.kicker}</span>
			<span class="now-icon" class:plain-alert={nowHero.tone === 'urgent' || nowHero.tone === 'result'} class:urgent={nowHero.tone === 'urgent'} class:live={nowHero.tone === 'live'} class:result={nowHero.tone === 'result'} class:done={nowHero.tone === 'done'} class:gold={nowHero.tone === 'gold'}>
				{#if nowHero.tone === 'urgent'}<img class="football-mark" src="/icons/football-alert.svg" alt="" />
				{:else if nowHero.tone === 'forecast'}<Telescope size={18} />
				{:else if nowHero.tone === 'live'}<Radio size={18} />
				{:else if nowHero.tone === 'result'}<img class="football-mark" src="/icons/football-alert.svg" alt="" />
				{:else if nowHero.tone === 'done' || nowHero.tone === 'gold'}<Trophy size={18} />
				{:else}<CheckCircle2 size={18} />{/if}
			</span>
		</div>

		<div class="now-copy">
			{#if nowHero.tone === 'result' && typeof nowHero.resultPoints === 'number'}
				<h2>
					{lastMatchTitlePrefix(nowHero.resultPoints)}<span class="gold-points">{pointText(nowHero.resultPoints)}</span>{lastMatchTitleSuffix(nowHero.resultPoints)}
				</h2>
			{:else}
				<h2>{nowHero.title}</h2>
			{/if}
			<p class="muted">{nowHero.body}</p>
			{#if nowHero.deadline}
				<DeadlineCountdown deadline={nowHero.deadline} label={nowHero.deadlineLabel ?? ''} />
			{/if}
		</div>

		{#if nowHero.tone === 'live' && nowHero.matches && nowHero.matches.length > 0}
			<div class="now-live-matches">
				{#each nowHero.matches as m (m.id)}
					<div class="now-result-card">
						<LeagueMatchCard match={m} leagueId={activeLeague?.id ?? ''} avatars={homeAvatars} flat showTv />
					</div>
				{/each}
			</div>
		{:else if nowHero.tone === 'result' && nowHero.match}
			<div class="now-result-card">
				<LeagueMatchCard match={nowHero.match} leagueId={activeLeague?.id ?? ''} avatars={homeAvatars} flat />
			</div>
		{:else if nowHero.match}
			<div class="now-match">
				<div class="now-teams">
					<span>
						{#if team(nowHero.match.homeTeam)}
							<Flag iso2={team(nowHero.match.homeTeam)?.iso2 ?? ''} code={team(nowHero.match.homeTeam)?.fifaCode ?? ''} size={22} />
						{/if}
						<b>{teamLabel(nowHero.match, 'h')}</b>
					</span>
					<strong class="digits">
						{#if playedM(nowHero.match) || isLiveStatus(nowHero.match.status)}
							{scoreText(nowHero.match)}
						{:else}
							{language.text('mot', 'mot', 'vs')}
						{/if}
					</strong>
					<span class="away">
						<b>{teamLabel(nowHero.match, 'a')}</b>
						{#if team(nowHero.match.awayTeam)}
							<Flag iso2={team(nowHero.match.awayTeam)?.iso2 ?? ''} code={team(nowHero.match.awayTeam)?.fifaCode ?? ''} size={22} />
						{/if}
					</span>
				</div>
				<div class="now-meta">
					{#if isLiveStatus(nowHero.match.status)}
						<span class="now-live-pill"><span class="live-dot" aria-hidden="true"></span>{language.text('Live', 'Live', 'Live')}</span>
					{/if}
					<span>{stageLabel(nowHero.match)}</span>
					<span>{kickoffLabel(nowHero.match.kickoff)}</span>
					{#if nowHero.match.tvChannel}<TvLogo channel={nowHero.match.tvChannel} compact />{/if}
				</div>
			</div>
		{/if}

		{#if tipsStore.loaded && totalMatches > 0 && missingMatchTips.length > 0}
			<div class="tip-progress-panel" style={`--tip-progress: ${progressPct}%`}>
				<div class="tip-progress-info">
					<span><b>{submittedOpenMatchTipCount}</b> {language.text(`av ${openMatchTipCount} åpne kamper levert`, `av ${openMatchTipCount} opne kampar leverte`, `of ${openMatchTipCount} open matches submitted`)}</span>
					<span class="muted">{language.text(`${missingMatchTips.length} mangler`, `${missingMatchTips.length} manglar`, `${missingMatchTips.length} missing`)}</span>
				</div>
				<div class="tip-progress-track" aria-hidden="true">
					<span></span>
				</div>
			</div>
		{/if}

		<a class="action-link" href={nowHero.href} onclick={(event) => void followTipsLink(event, nowHero.href)}>
			<ListChecks size={16} />
			{nowHero.label}
		</a>
	</section>

	{#if !tournamentFinished && tournamentStarted && activeLeague && leagueProgress && leagueProgress.events.length > 0}
		<section class="card tile progress-card home-span-support">
			<div class="hd">
				<h3><img class="football-mark football-mark-inline" src="/icons/football-alert.svg" alt="" /> {language.text('Poengtrend', 'Poengtrend', 'Points trend')}</h3>
				<a class="hdlink" href={leagueHref}>{language.text('Liga', 'Liga', 'League')}</a>
			</div>

			{#if progressStats}
				<div class="trend-hero">
					<div class="trend-total">
						<span class="trend-label">{language.text('Kamppoeng', 'Kamppoeng', 'Match points')}</span>
						<b class="trend-num">{progressStats.total}<em>p</em></b>
					</div>
					<span class="trend-chip" class:flat={progressStats.last3 === 0}>
						{#if progressStats.last3 > 0}<TrendingUp size={13} />{:else}<Minus size={13} />{/if}
						{shortPoints(progressStats.last3, true)}
						<i>{language.text('siste 3', 'siste 3', 'last 3')}</i>
					</span>
				</div>

				{#if formBars}
					<div class="trend-form">
						<div class="form-bars">
							{#each formBars as bar (bar.matchId)}
								<span class="form-col">
									<span
										class="form-bar"
										class:zero={bar.points === 0}
										class:exact={bar.exact}
										style={`--h:${(bar.height * 100).toFixed(0)}%`}
									></span>
									<span class="form-bar-tip">{bar.label}</span>
								</span>
							{/each}
						</div>
						<span class="trend-cap">{language.text(`Siste ${formBars.length} kamper`, `Siste ${formBars.length} kampar`, `Last ${formBars.length} matches`)}</span>
					</div>
				{/if}

				<div class="trend-stats">
					<span><b>{progressStats.avg.toFixed(1)}<em>p</em></b><i>{language.text('Snitt', 'Snitt', 'Avg')}</i></span>
					<span><b>{progressStats.hitRate}<em>%</em></b><i>{language.text('Treff', 'Treff', 'Hit rate')}</i></span>
					<span><b>{progressStats.exact}</b><i>{language.text('Eksakte', 'Eksakte', 'Exact')}</i></span>
					<span><b>{progressStats.best}<em>p</em></b><i>{language.text('Beste', 'Beste', 'Best')}</i></span>
				</div>
			{/if}
		</section>
	{/if}

	{#if tournamentStarted && leaguesLoaded && activeLeague && activeLeagueRow && lb.length > 1}
		<section class="card tile standing-card home-span-support">
			<div class="hd">
				<h3><Crown size={15} style="margin-right:0.35rem;vertical-align:-2px;color:var(--gold)" /> {language.text('Ligatabell', 'Ligatabell', 'League table')}</h3>
				<div class="league-card-actions">
					{#if leagues.length > 1}
						<div class="league-select-shell">
							<select
								class="league-select"
								aria-label={language.text('Velg liga for ligatabell', 'Vel liga for ligatabell', 'Choose league for league table')}
								value={activeLeague.id}
								onchange={onLeagueSelect}
							>
								{#each leagues as leagueOption (leagueOption.id)}
									<option value={leagueOption.id}>{leagueOption.name}</option>
								{/each}
							</select>
							<ChevronDown size={15} />
						</div>
					{/if}
					<a class="hdlink" href={leagueHref}>{language.text('Tabell', 'Tabell', 'Table')}</a>
				</div>
			</div>

			<div class="standing-hero">
				<span class="rank-big">#{myRank}</span>
				<span class="standing-copy">
					<b>{shortPoints(totalPoints)}</b>
					<i>
						{#if myRank === 1}
							{tournamentFinished
								? language.text('Du vant ligaen', 'Du vann ligaen', 'You won the league')
								: language.text('Du leder ligaen', 'Du leiar ligaen', 'You lead the league')}
						{:else if personAbove && gapToAbove > 0}
							{language.text(`${shortPoints(gapToAbove)} bak ${personAbove.name}`, `${shortPoints(gapToAbove)} bak ${personAbove.name}`, `${shortPoints(gapToAbove)} behind ${personAbove.name}`)}
						{:else if personAbove}
							{language.text(`Lik med ${personAbove.name}`, `Lik med ${personAbove.name}`, `Level with ${personAbove.name}`)}
						{:else}
							{language.text('Du er på tabellen', 'Du er på tabellen', 'You are on the table')}
						{/if}
					</i>
				</span>
			</div>

			{#if personAbove}
				<div class="league-gaps">
				{#if personAbove}
					<span>
						<i>{language.text('Jakter', 'Jaktar', 'Chasing')}</i>
						<b>{personAbove.name}</b>
						<em>{gapToAbove > 0 ? language.text(`${shortPoints(gapToAbove)} bak`, `${shortPoints(gapToAbove)} bak`, `${shortPoints(gapToAbove)} behind`) : language.text('likt', 'likt', 'level')}</em>
					</span>
				{/if}
				</div>
			{/if}

			<div class="mini-lb">
				{#each miniLeaderboard as row (row.userId)}
					<a class="mini-lb-row" class:me={row.userId === auth.user?.id} href={leagueHref}>
						<span class="mini-rank">#{row.rank}</span>
						<span class="mini-name">
							<Avatar name={row.name} src={row.avatarUrl} size={24} />
							<b>{row.name}</b>
							{#if row.userId === auth.user?.id}<i>{language.text('deg', 'deg', 'you')}</i>{/if}
						</span>
						<strong>{shortPoints(row.total)}</strong>
					</a>
				{/each}
			</div>
		</section>
	{/if}

	{#if !tournamentFinished && chatLoaded && (chatItems.length > 0 || chatError)}
		<section class="card tile chat-preview-card home-span-support" class:has-unread={unreadChatItems.length > 0}>
			<div class="hd">
				<h3><MessageCircle size={15} style="margin-right:0.35rem;vertical-align:-2px;color:var(--accent)" /> {language.text('Siste fra liga-chatten', 'Siste frå liga-chatten', 'Latest from league chat')}</h3>
				<a class="hdlink" href="/leagues">{language.text('Ligaer', 'Ligaer', 'Leagues')}</a>
			</div>

			<div class="chat-preview-list">
				{#if chatError}
					<p class="muted chat-error">{language.text('Kunne ikke hente siste chat nå.', 'Kunne ikkje hente siste chat no.', 'Could not fetch the latest chat right now.')}</p>
				{/if}
				{#each chatItems as item (item.leagueId)}
					<a class="chat-preview" class:unread={item.unread > 0} href={`/leagues/${item.leagueId}#chat`}>
						<span class="chat-badge"><MessageCircle size={15} /></span>
						<span class="chat-main">
							<span class="chat-title">
								<b>{item.leagueName}</b>
								{#if item.unread > 0}<i>{unreadLabel(item.unread)}</i>{/if}
							</span>
							{#if item.message}
								<span class="chat-text">
									<strong>{item.message.userId === auth.user?.id ? language.text('Du', 'Du', 'You') : item.message.user.name}:</strong>
									{item.message.deleted ? language.text('Melding slettet', 'Melding sletta', 'Message deleted') : chatPreview(item.message.text)}
								</span>
							{:else}
								<span class="chat-text muted">{language.text('Ingen meldinger ennå', 'Ingen meldingar enno', 'No messages yet')}</span>
							{/if}
						</span>
						<span class="chat-jump">
							{#if item.message}{chatTimeLabel(item.message.created)}{:else}{language.text('Åpne', 'Opne', 'Open')}{/if}
							<ArrowUpRight size={14} />
						</span>
					</a>
				{/each}
			</div>
		</section>
	{/if}

	{#if !tournamentFinished && fs.loaded}
		<section class="card tile forecast-pulse-card home-span-support" class:urgent={forecastPulse.tone === 'urgent'} class:out={forecastPulse.tone === 'out'}>
			<div class="hd">
				<h3 style="flex:1"><Telescope size={15} style="margin-right:0.35rem;vertical-align:-2px;color:var(--gold)" /> {forecastPulse.kicker}: {forecastPulse.title}</h3>
				<a class="hdlink" href="/forecast">{forecastPulse.label}</a>
			</div>
			<p class="muted forecast-copy">{forecastPulse.body}</p>
			{#if hasPodium}
				<div class="forecast-mini-podium">
					{#each podium as pick (pick.place)}
						{#if pick.id}
							{@const pickedTeam = teamAny(pick.id)}
							<span class="place-{pick.place}">
								<i>{pick.place}</i>
								<b>
									<Flag iso2={pickedTeam?.iso2 ?? ''} code={pickedTeam?.fifaCode ?? ''} size={15} />
									{teamDisplayName(pickedTeam, language.text('Ukjent', 'Ukjent', 'Unknown'))}
								</b>
							</span>
						{/if}
					{/each}
				</div>
			{/if}
		</section>
	{/if}

	{#if false && hasPodium}
		<!-- Top 3 podium summary -->
		<section class="card champ tile podium-card home-span-support">
			<div class="hd">
				<h3><Crown size={15} style="margin-right:0.35rem;vertical-align:-2px;color:var(--gold)" /> Din pall</h3>
				<a class="hdlink" href="/forecast">Endre</a>
			</div>
			<div class="podium-list">
				{#each podium as pick (pick.place)}
					{#if pick.id}
						{@const pickedTeam = teamAny(pick.id)}
						<div class="podium-row place-{pick.place}">
							<span class="medal">{pick.place}</span>
							<span>
								<i>{pick.label}</i>
								<b>
									<Flag iso2={pickedTeam?.iso2 ?? ''} code={pickedTeam?.fifaCode ?? ''} size={16} />
									{teamDisplayName(pickedTeam, language.text('Ukjent', 'Ukjent', 'Unknown'))}
								</b>
							</span>
						</div>
					{/if}
				{/each}
			</div>
		</section>
	{/if}

	{#if nextMatchesPreview.length > 0}
		<section class="card tile next-card home-span-primary">
			<div class="hd">
				<h3><Clock size={15} style="margin-right:0.35rem;vertical-align:-2px;color:var(--accent)" /> {language.text('Kommende kamper', 'Komande kampar', 'Upcoming matches')}</h3>
			</div>

			<div class="ready-list">
				{#each nextMatchesPreview as match (match.id)}
					<a class="ready-item" class:missing={!tipsStore.tips[match.id] && teamsResolved(match) && !isLocked(match)} href={matchTipHref(match)} onclick={(event) => void followTipsLink(event, matchTipHref(match))}>
						<span class="ready-meta">
							<span>{kickoffLabel(match.kickoff)}</span>
							<span class="spacer"></span>
							{#if match.tvChannel}<TvLogo channel={match.tvChannel} compact />{/if}
							{#if tipsStore.tips[match.id]}
								<i class="ready-state ok">{language.text('Levert', 'Tipset', 'Submitted')}</i>
							{:else if teamsResolved(match) && !isLocked(match)}
								<i class="ready-state warn">{language.text('Mangler', 'Manglar', 'Missing')}</i>
							{/if}
						</span>
						<span class="ready-stage">{stageLabel(match)}</span>
						<div class="ready-teams">
							<span class="ready-team">
								{#if team(match.homeTeam)}
									<Flag iso2={team(match.homeTeam)?.iso2 ?? ''} code={team(match.homeTeam)?.fifaCode ?? ''} size={18} />
								{/if}
								<b>{teamLabel(match, 'h')}</b>
							</span>
							<span class="ready-vs">{language.text('mot', 'mot', 'vs')}</span>
							<span class="ready-team away">
								<b>{teamLabel(match, 'a')}</b>
								{#if team(match.awayTeam)}
									<Flag iso2={team(match.awayTeam)?.iso2 ?? ''} code={team(match.awayTeam)?.fifaCode ?? ''} size={18} />
								{/if}
							</span>
						</div>
					</a>
				{/each}
			</div>
		</section>
	{/if}

	<!-- Recent results / final league summary -->
	{#if (tournamentFinished && finalLeaguePlacements.length > 0) || recentResults.length > 0}
		<section class="card results tile home-span-support">
			<div class="hd">
				<h3>
					{#if tournamentFinished}
						<Trophy size={15} style="margin-right:0.35rem;vertical-align:-2px;color:var(--gold)" />
						{language.text('Sluttresultat', 'Sluttresultat', 'Final standings')}
					{:else}
						<ListChecks size={15} style="margin-right:0.35rem;vertical-align:-2px;color:var(--accent)" />
						{language.text('Siste resultat', 'Siste resultat', 'Latest results')}
					{/if}
				</h3>
				{#if tournamentFinished && finalLeaguePlacements.length > 0}
					<a class="hdlink" href="/leagues">{language.text('Alle ligaer', 'Alle ligaer', 'All leagues')}</a>
				{:else if activeLeague && recentResults.length > 0}
					<a class="hdlink" href={`/leagues/${activeLeague.id}?view=matches`}>{language.text('Alle kamper', 'Alle kampar', 'All matches')}</a>
				{/if}
			</div>
			{#if tournamentFinished && finalLeaguePlacements.length > 0}
				<div class="league-finish-summary">
					<p class="league-finish-title">{leagueFinishSummary.title}</p>
					<p class="muted league-finish-copy">{leagueFinishSummary.body}</p>
				</div>
				<div class="league-results-list" style="max-height: 250px; overflow-y: auto; margin-right:-0.5rem; padding-right:0.5rem;">
					{#each finalLeaguePlacements as result (result.id)}
						<a class="league-result-row" class:winner={result.rank === 1} href={result.href}>
							<span class="league-result-rank">{result.medal}</span>
							<span class="league-result-main">
								<b>{result.name}</b>
								<i>{rankSummary(result.rank, result.members)}</i>
							</span>
							<strong>{shortPoints(result.total)}</strong>
						</a>
					{/each}
				</div>
			{:else}
				<div class="result-cards">
					{#each recentResults as m (m.id)}
						<LeagueMatchCard match={m} leagueId={activeLeague?.id ?? ''} avatars={homeAvatars} flat />
					{/each}
				</div>
			{/if}
		</section>
	{/if}

	{#if tournamentFinished && (goldenBootLeaders.length > 0 || myGoldenBootPick)}
		<section class="card tile golden-boot-card home-span-support">
			<div class="hd">
				<h3>
					<Volleyball size={15} style="margin-right:0.35rem;vertical-align:-2px;color:var(--gold)" />
					{language.text('Toppscorer og min spiller', 'Toppscorar og min spelar', 'Top scorer and your player')}
				</h3>
			</div>

			{#if goldenBootLeaders.length > 0}
				<div class="golden-boot-section">
					<p class="golden-boot-section-title">{language.text('Toppscorertabell', 'Toppscorartabell', 'Top scorer table')}</p>
					<div class="golden-boot-list">
						{#each goldenBootLeaders as player (player.id)}
							{@const playerTeam = player.teamId ? teamAny(player.teamId) : null}
							<div class="golden-boot-row" class:leader={player.rank === 1}>
								<span class="golden-boot-rank">{medalForRank(player.rank)}</span>
								<span class="golden-boot-player">
									{#if player.photoUrl}
										<img class="golden-boot-photo" src={player.photoUrl} alt="" loading="lazy" />
									{:else}
										<span class="golden-boot-photo fallback">{initials(player.name)}</span>
									{/if}
									<span class="golden-boot-main">
										<b>
											{#if playerTeam}
												<Flag iso2={playerTeam.iso2 ?? ''} code={playerTeam.fifaCode ?? ''} size={16} />
											{/if}
											{player.name}
											{#if player.id === myGoldenBootPlayerId}
												<span class="golden-boot-tag">{language.text('(mitt valg)', '(mitt val)', '(my pick)')}</span>
											{/if}
										</b>
										<i>{player.teamName}</i>
									</span>
								</span>
								<strong class="golden-boot-goals">{player.goals} {language.text('mål', 'mål', 'goals')}</strong>
							</div>
						{/each}
					</div>
				</div>
			{/if}

			{#if myGoldenBootPick && !myGoldenBootPickInLeaders}
				{@const pickedTeam = myGoldenBootPick.teamId ? teamAny(myGoldenBootPick.teamId) : null}
				<div class="golden-boot-section golden-boot-pick">
					<p class="golden-boot-section-title">{language.text('Min spiller', 'Min spelar', 'My player')}</p>
					<div class="golden-boot-row" class:leader={myGoldenBootPickIsWinner}>
						<span class="golden-boot-rank">
							{#if myGoldenBootPick.rank > 0}
								{medalForRank(myGoldenBootPick.rank)}
							{:else}
								⚽
							{/if}
						</span>
						<span class="golden-boot-player">
							{#if myGoldenBootPick.photoUrl}
								<img class="golden-boot-photo" src={myGoldenBootPick.photoUrl} alt="" loading="lazy" />
							{:else}
								<span class="golden-boot-photo fallback">{initials(myGoldenBootPick.name)}</span>
							{/if}
							<span class="golden-boot-main">
								<b>
									{#if pickedTeam}
										<Flag iso2={pickedTeam.iso2 ?? ''} code={pickedTeam.fifaCode ?? ''} size={16} />
									{/if}
									{myGoldenBootPick.name}
								</b>
								<i>
									{#if myGoldenBootPickIsWinner}
										{language.text('Du valgte vinneren', 'Du valde vinnaren', 'You picked the winner')}
									{:else if myGoldenBootPick.rank > 0}
										{language.text(`Endte på #${myGoldenBootPick.rank}`, `Enda på #${myGoldenBootPick.rank}`, `Finished #${myGoldenBootPick.rank}`)}
									{:else}
										{language.text('Ingen plassering i sluttabellen', 'Ingen plassering i sluttabellen', 'No placing in the final table')}
									{/if}
								</i>
							</span>
						</span>
						<strong class="golden-boot-goals">{myGoldenBootPick.goals} {language.text('mål', 'mål', 'goals')}</strong>
					</div>
				</div>
			{/if}
		</section>
	{/if}

	{#if tournamentFinished}
		<!-- Legg til ekte turnerings-pallkort -->
		<section class="card champ tile podium-card home-span-support">
			<div class="hd">
				<h3><Crown size={15} style="margin-right:0.35rem;vertical-align:-2px;color:var(--gold)" /> {language.text('Verdensmester', 'Verdsmeister', 'World champion')}</h3>
			</div>
			<div class="podium-list">
				{#each realPodiumParams as pick (pick.place)}
					{#if pick.id}
						{@const pickedTeam = teamAny(pick.id)}
						<div class="podium-row place-{pick.place}">
							<span class="medal">
								{#if pick.place === 1}🥇
								{:else if pick.place === 2}🥈
								{:else if pick.place === 3}🥉{/if}
							</span>
							<span>
								<i>{pick.label}</i>
								<b>
									<Flag iso2={pickedTeam?.iso2 ?? ''} code={pickedTeam?.fifaCode ?? ''} size={16} />
									{teamDisplayName(pickedTeam, language.text('Ukjent', 'Ukjent', 'Unknown'))}
								</b>
							</span>
						</div>
					{/if}
				{/each}
			</div>
		</section>

	{/if}

</div>
	<p class="home-footer">{language.text('VM 2026 · 11. juni - 19. juli', 'VM 2026 · 11. juni - 19. juli', 'World Cup 2026 · 11 June - 19 July')}</p>
{/if}

<style>
	/* Date range sits in the hero on desktop; on mobile it moves to a footer at
	   the bottom (and "World Cup 26" shows in the top header instead). */
	.home-footer {
		display: none;
	}
	@media (max-width: 899px) {
		.home-hero .kicker {
			display: none;
		}
		.home-footer {
			display: block;
			text-align: center;
			margin: 1.5rem 0 0;
			font-size: 0.72rem;
			font-weight: 600;
			letter-spacing: 0.05em;
			text-transform: uppercase;
			color: var(--muted-2);
		}
	}
	.hero-bar {
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: 1rem;
		flex-wrap: wrap;
		margin: 0.25rem 0 1.1rem;
	}
	/* removed unused: .hero-bar h1, .hero-bar .sub */
	
	/* Progress bar */
	.progress-bar-container {
		margin: 1.5rem 0;
	}
	.progress-info {
		display: flex;
		justify-content: space-between;
		font-size: 0.85rem;
		margin-bottom: 0.4rem;
	}
	.progress-label {
		color: var(--muted);
		font-weight: 500;
	}
	.progress-pct {
		font-family: var(--font-mono);
		font-variant-numeric: tabular-nums;
		font-weight: 700;
	}
	.progress-track {
		height: 8px;
		background: var(--surface-3);
		border-radius: var(--radius-pill);
		overflow: hidden;
	}
	.progress-fill {
		height: 100%;
		background: var(--success);
		border-radius: var(--radius-pill);
		transition: width 0.5s cubic-bezier(0.4, 0, 0.2, 1);
	}

	.tasks:not(.done) {
		border-color: color-mix(in srgb, var(--success) 60%, var(--border));
	}
	.tasks.done {
		border-color: var(--success);
		background: color-mix(in srgb, var(--success) 4%, var(--surface));
	}
	.done-cheer {
		margin-top: 1rem;
		display: flex;
		justify-content: center;
		padding: 1rem;
	}

	.herorank {
		display: grid;
		gap: 0.15rem;
		padding: 0.7rem 1.05rem;
		background:
			linear-gradient(
				135deg,
				color-mix(in srgb, var(--accent) 18%, transparent),
				transparent 60%
			),
			var(--surface);
		border: 1px solid color-mix(in srgb, var(--accent) 28%, var(--border));
		border-radius: var(--radius);
		color: var(--text);
	}
	.rk-label {
		font-size: 0.68rem;
		font-weight: 700;
		letter-spacing: 0.18em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.rk-row {
		display: inline-flex;
		align-items: baseline;
		gap: 0.4rem;
		font-family: var(--font);
	}
	.rk-row b {
		font-family: var(--font-display);
		font-size: 1.5rem;
		color: var(--accent);
	}
	/* removed unused: .rk-row b.pt */
	.rk-row i {
		font-style: normal;
		color: var(--muted);
		font-size: 0.78rem;
	}
	/* removed unused: .rk-row .dot */

	.tile {
		display: flex;
		flex-direction: column;
		gap: 0.7rem;
		min-height: 160px;
		color: var(--text);
		transition: transform 0.18s ease, border-color 0.18s ease;
	}
	.tile .hd {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 0.5rem;
	}
	.tile .hd h3 {
		font-size: 1.15rem;
		margin: 0.1rem 0 0;
	}
	.tile .hdlink {
		font-size: 0.85rem;
		color: var(--accent);
		white-space: nowrap;
		font-weight: 600;
	}
	.league-card-actions {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		flex-wrap: wrap;
		gap: 0.45rem;
		min-width: 0;
	}
	.league-select-shell {
		position: relative;
		display: inline-flex;
		align-items: center;
		min-width: 0;
	}
	.league-select {
		appearance: none;
		width: auto;
		max-width: min(13rem, 46vw);
		min-height: 34px;
		padding: 0.4rem 1.95rem 0.4rem 0.72rem;
		border: 1px solid color-mix(in srgb, var(--accent) 18%, var(--border));
		border-radius: var(--radius-pill);
		background:
			linear-gradient(180deg, color-mix(in srgb, var(--bg) 38%, transparent), transparent),
			var(--surface-2);
		color: var(--text);
		box-shadow: 0 1px 4px rgba(9, 9, 11, 0.05);
		font: inherit;
		font-size: 0.78rem;
		font-weight: 800;
		cursor: pointer;
	}
	.league-select:focus-visible {
		outline: 2px solid color-mix(in srgb, var(--accent) 38%, transparent);
		outline-offset: 2px;
	}
	.league-select-shell :global(svg) {
		position: absolute;
		right: 0.65rem;
		pointer-events: none;
		color: var(--muted);
	}
	.league-name-pill {
		display: inline-flex;
		align-items: center;
		min-height: 34px;
		padding: 0.35rem 0.68rem;
		border-radius: var(--radius-pill);
		background: var(--surface-2);
		color: var(--muted);
		font-size: 0.78rem;
		font-weight: 800;
	}

	/* ===== Home task panel ===== */
	.tasks {
		gap: 1rem;
		background: var(--surface);
		border-color: var(--border);
	}
	.tasks.done {
		border-color: color-mix(in srgb, var(--success) 22%, var(--border));
	}
	.task-head {
		display: grid;
		gap: 0.35rem;
	}
	/* removed unused: .task-head h2, .task-head p */
	.tasks.done {
		border-color: var(--success);
		background: color-mix(in srgb, var(--success) 4%, var(--surface));
	}
	.done-cheer {
		margin-top: 1rem;
		display: flex;
		justify-content: center;
		padding: 1rem;
	}
	.progress-bar-container {
		margin: 1.5rem 0;
	}
	.progress-info {
		display: flex;
		justify-content: space-between;
		font-size: 0.85rem;
		margin-bottom: 0.4rem;
	}
	.progress-label {
		color: var(--muted);
		font-weight: 500;
	}
	.progress-pct {
		font-family: var(--font-mono);
		font-variant-numeric: tabular-nums;
		font-weight: 700;
	}
	.progress-track {
		height: 8px;
		background: var(--surface-3);
		border-radius: var(--radius-pill);
		overflow: hidden;
	}
	.progress-fill {
		height: 100%;
		background: var(--success);
		border-radius: var(--radius-pill);
		transition: width 0.5s cubic-bezier(0.4, 0, 0.2, 1);
	}

	.task-grid {
		display: flex;
		flex-direction: column;
		gap: 0;
		margin: 0.5rem 0 1rem;
		border-top: 1px solid var(--border);
	}
	.task-status {
		display: grid;
		grid-template-columns: auto 1fr;
		align-items: center;
		gap: 0.75rem;
		padding: 0.8rem 0;
		background: transparent;
		border: none;
		border-bottom: 1px solid var(--border);
		color: var(--text);
		transition: opacity 0.15s ease;
	}
	.task-status:hover {
		opacity: 0.8;
	}
	.task-status.warn {
		color: var(--warning);
	}
	.task-status.warn i {
		color: var(--warning);
	}
	.task-icon {
		display: grid;
		place-items: center;
		width: 34px;
		height: 34px;
		border-radius: 50%;
		background: color-mix(in srgb, var(--success) 12%, transparent);
		color: var(--success);
		border: none;
	}
	/* removed unused: .task-status.warn .task-icon */
	.task-status b,
	.task-status i {
		display: block;
	}
	.task-status b {
		font-weight: 650;
	}
	.task-status i {
		margin-top: 0.1rem;
		font-style: normal;
		color: var(--muted);
		font-size: 0.82rem;
		line-height: 1.25;
	}
	.missing-preview, .next-ok {
		display: grid;
		gap: 0.85rem;
		padding: 0;
		background: transparent;
		border: none;
	}
	.quick-head,
	.quick-title {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.7rem;
		flex-wrap: wrap;
	}
	/* removed unused: .quick-title h3 */
	.pill.missing {
		color: var(--warning);
		border: none;
		background: color-mix(in srgb, var(--warning) 12%, transparent);
	}
	.cd {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		padding: 0.3rem 0;
		background: transparent;
		border: none;
		font-family: var(--font-mono);
		font-size: 0.8rem;
		color: var(--muted);
	}

	.match-line {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
		align-items: center;
		gap: 0.85rem;
		padding: 0.8rem 0;
		background: transparent;
		border: none;
		border-top: 1px dashed var(--border);
		border-bottom: 1px dashed var(--border);
	}
	.team {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		min-width: 0;
		color: var(--text);
	}
	.team.r {
		justify-content: flex-end;
	}
	.team b {
		font-weight: 700;
		font-size: 1.05rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.vs {
		color: var(--muted);
		font-size: 0.78rem;
		font-weight: 650;
		text-transform: uppercase;
		letter-spacing: 0.08em;
	}
	.hp-actions {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		position: relative;
	}
	.btn.save {
		flex: 1;
		background: var(--text);
		color: var(--bg);
		font-weight: 700;
	}
	.btn.save:disabled {
		opacity: 0.6;
	}
	.more {
		color: var(--muted);
		font-weight: 600;
		white-space: nowrap;
	}
	.next-ok span {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		color: var(--muted);
		font-size: 0.8rem;
		font-weight: 650;
	}
	.next-ok b {
		font-size: 1rem;
	}
	.next-ok i {
		font-style: normal;
		color: var(--muted);
		font-size: 0.85rem;
	}
	@media (min-width: 640px) {
		.task-grid {
			flex-direction: row;
		}
		.task-status {
			border-bottom: none;
			border-right: 1px solid var(--border);
			padding: 0.5rem 1rem 0.5rem 0;
			flex: 1;
		}
		.task-status:last-child {
			border-right: none;
			padding-right: 0;
			padding-left: 1rem;
		}
	}
	@media (max-width: 720px) {
		.match-line {
			grid-template-columns: 1fr;
		}
		.team.r {
			justify-content: flex-start;
			flex-direction: row-reverse;
		}
		.vs {
			justify-content: center;
			text-align: center;
		}
		.hp-actions {
			flex-direction: column;
			align-items: stretch;
		}
		.more {
			text-align: center;
		}
	}

	/* removed unused: Standings (.st) CSS block */
	.ptslabel {
		font-size: 0.7rem;
		text-align: right;
		margin: 0;
		letter-spacing: 0.18em;
		text-transform: uppercase;
	}

	/* ===== Accuracy ===== */
	.acc-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.6rem;
	}
	.acc-num {
		display: inline-flex;
		align-items: baseline;
		gap: 0.2rem;
	}
	.acc-num b {
		font-family: var(--font-display);
		font-size: clamp(2.4rem, 5vw, 3.4rem);
		color: var(--accent);
		font-weight: 700;
	}
	.acc-num i {
		font-style: normal;
		font-size: 1.4rem;
		color: var(--muted);
		margin-left: 0.1rem;
	}
	.acc-donut {
		width: 70px;
		height: 70px;
		border-radius: 50%;
		display: grid;
		place-items: center;
		color: var(--accent);
		background:
			conic-gradient(
				var(--accent) calc(var(--p, 0) * 1%),
				var(--surface-2) 0
			);
		box-shadow: inset 0 0 0 10px var(--surface);
	}
	.bar {
		height: 8px;
		background: var(--surface-2);
		border-radius: 999px;
		overflow: hidden;
	}
	.bar span {
		display: block;
		height: 100%;
		background: linear-gradient(
			90deg,
			var(--accent),
			color-mix(in srgb, var(--accent) 60%, var(--accent-2))
		);
	}

	/* ===== Group predictor ===== */
	.gpl {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		gap: 0.45rem;
	}
	/* removed unused: .gpl list & children */

	/* ===== Champion footer ===== */
	.champ {
		display: grid;
		gap: 1rem;
		background: var(--surface);
	}
	@media (min-width: 900px) {
		.champ {
			grid-template-columns: 1fr auto;
			align-items: center;
		}
	}
	/* removed unused: .champ-text h2 */
	.kicker.gold {
		color: var(--gold);
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
	}
	/* removed unused: .champ-text p */
	.champ-picks {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.7rem;
		min-width: min(420px, 100%);
	}
	.pick {
		display: grid;
		justify-items: center;
		text-align: center;
		gap: 0.35rem;
		padding: 0.85rem 0.6rem;
		background: var(--surface-2);
		border: 1px dashed var(--border-strong);
		border-radius: var(--radius);
		color: var(--text);
		transition: border-color 0.15s ease, background 0.15s ease, transform 0.15s ease;
	}
	.pick.podium-pick {
		border: 1px solid var(--border);
	}
	.pick.podium-pick:hover {
		transform: none;
		border-color: var(--border);
		background: var(--surface-2);
	}
	/* removed unused: a.pick:hover */
	/* removed unused: .pick .lab */
	/* removed unused: .pick .ic and medal variants */

	/* removed unused: .pick .pl */
	.team-name {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.35rem;
		color: var(--text);
	}

	.pill.warn {
		color: var(--warning);
		border-color: color-mix(in srgb, var(--warning) 40%, var(--border));
	}

	/* ===== Modern compact home override ===== */
	.home-hero {
		display: grid;
		gap: 0.75rem;
		margin: 0.1rem 0 0.9rem;
	}
	.home-hero h1 {
		margin: 0.15rem 0 0;
		font-size: clamp(1.65rem, 6.5vw, 2.75rem);
		letter-spacing: -0.035em;
		line-height: 1.1;
	}
	.hero-greeting .greet {
		font-weight: 500;
		color: var(--muted);
		letter-spacing: -0.02em;
	}
	.hero-greeting .punct {
		color: var(--muted);
		font-weight: 500;
		opacity: 0.5;
	}
	.hero-greeting .name {
		font-weight: 800;
		background: linear-gradient(110deg, var(--text) 30%, var(--accent) 100%);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
		background-clip: text;
	}
	.hero-chips {
		display: flex;
		flex-wrap: wrap;
		gap: 0.45rem;
	}
	.hero-chip {
		display: inline-flex;
		align-items: baseline;
		gap: 0.35rem;
		min-height: 36px;
		padding: 0.42rem 0.65rem;
		border-radius: var(--radius-pill);
		background: color-mix(in srgb, var(--surface) 90%, transparent);
		box-shadow: 0 1px 4px rgba(9, 9, 11, 0.05);
		color: var(--text);
	}
	.hero-chip span,
	.hero-chip i {
		font-size: 0.72rem;
		font-style: normal;
		font-weight: 600;
		color: var(--muted);
	}
	.hero-chip b {
		font-family: var(--font-mono);
		font-size: 1.05rem;
		font-weight: 800;
		color: var(--accent);
	}
	.hero-chip.league-pill b {
		color: var(--text);
	}
	.hero-chip.league-pill.loading {
		border-color: color-mix(in srgb, var(--accent) 14%, var(--border));
	}
	.league-rank-skeleton {
		position: relative;
		display: inline-block;
		flex: 0 0 auto;
		width: 2.15rem;
		height: 1.05rem;
		border-radius: var(--radius-pill);
		overflow: hidden;
		background:
			linear-gradient(
				90deg,
				color-mix(in srgb, var(--muted) 14%, transparent) 0%,
				color-mix(in srgb, var(--accent) 18%, transparent) 44%,
				color-mix(in srgb, var(--muted) 14%, transparent) 88%
			);
		background-size: 220% 100%;
		box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--text) 7%, transparent);
		animation: leagueRankShimmer 1.35s ease-in-out infinite;
	}
	.league-rank-skeleton::after {
		content: '';
		position: absolute;
		inset: 0.3rem 0.48rem;
		border-radius: inherit;
		background: color-mix(in srgb, var(--text) 18%, transparent);
		opacity: 0.72;
		animation: leagueRankPulse 1.35s ease-in-out infinite;
	}
	@keyframes leagueRankShimmer {
		0% {
			background-position: 120% 0;
		}
		100% {
			background-position: -80% 0;
		}
	}
	@keyframes leagueRankPulse {
		0%,
		100% {
			opacity: 0.38;
			transform: scaleX(0.78);
		}
		50% {
			opacity: 0.82;
			transform: scaleX(1);
		}
	}
	.hero-chip.error-pill {
		border: 1px solid color-mix(in srgb, var(--warning) 42%, var(--border));
		background: color-mix(in srgb, var(--warning) 10%, var(--surface));
		color: var(--warning);
		font-size: 0.78rem;
		font-weight: 800;
	}
	.hero-chip.points-pill {
		align-items: center;
		gap: 0.5rem;
		padding: 0.28rem 0.38rem 0.28rem 0.72rem;
		border: 1px solid color-mix(in srgb, var(--text) 18%, var(--border));
		background: var(--text);
		color: var(--bg);
		box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--bg) 12%, transparent), 0 12px 22px -18px rgba(9, 9, 11, 0.55);
	}
	.hero-chip.points-pill span {
		color: color-mix(in srgb, var(--bg) 72%, transparent);
		font-size: 0.68rem;
		letter-spacing: 0.09em;
		text-transform: uppercase;
	}
	.hero-chip.points-pill b {
		display: grid;
		place-items: center;
		min-width: 2.15rem;
		height: 2rem;
		padding-inline: 0.42rem;
		border-radius: var(--radius-pill);
		background: var(--bg);
		color: var(--text);
		font-size: 1rem;
		box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--text) 14%, transparent);
	}
	.league-finish-summary {
		display: grid;
		gap: 0.3rem;
		margin-bottom: 0.85rem;
	}
	.league-finish-title {
		margin: 0;
		font-size: 1rem;
		font-weight: 800;
		line-height: 1.3;
	}
	.league-finish-copy {
		margin: 0;
	}
	.league-results-list {
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
	}
	.league-result-row {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		align-items: center;
		gap: 0.75rem;
		padding: 0.6rem 0.7rem;
		border-radius: 0.85rem;
		background: var(--surface-2);
		border: 1px solid color-mix(in srgb, var(--border) 70%, transparent);
		color: inherit;
		transition: transform 120ms ease, border-color 120ms ease, background 120ms ease;
	}
	.league-result-row:hover {
		transform: translateY(-1px);
		border-color: color-mix(in srgb, var(--accent) 28%, var(--border));
		background: color-mix(in srgb, var(--surface-2) 88%, var(--accent));
	}
	.league-result-row.winner {
		border-color: color-mix(in srgb, var(--gold) 34%, var(--border));
		background: color-mix(in srgb, var(--gold) 10%, var(--surface));
	}
	.league-result-rank {
		display: grid;
		place-items: center;
		min-width: 2rem;
		font-family: var(--font-mono);
		font-size: 1.05rem;
		font-weight: 700;
		color: var(--text);
	}
	.league-result-main {
		display: grid;
		gap: 0.15rem;
		min-width: 0;
	}
	.league-result-main b {
		overflow-wrap: anywhere;
	}
	.league-result-main i {
		font-style: normal;
		font-size: 0.78rem;
		color: var(--muted);
	}
	.league-result-row strong {
		font-family: var(--font-mono);
		font-size: 0.95rem;
		color: var(--accent);
	}
	.golden-boot-card {
		display: grid;
		gap: 0.85rem;
	}
	.golden-boot-section {
		display: grid;
		gap: 0.45rem;
	}
	.golden-boot-section-title {
		margin: 0;
		font-size: 0.72rem;
		font-weight: 800;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.golden-boot-list {
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
	}
	.golden-boot-row {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		align-items: center;
		gap: 0.75rem;
		padding: 0.7rem 0.8rem;
		border-radius: 0.95rem;
		background: var(--surface-2);
		border: 1px solid color-mix(in srgb, var(--border) 70%, transparent);
	}
	.golden-boot-row.leader {
		border-color: color-mix(in srgb, var(--gold) 34%, var(--border));
		background: color-mix(in srgb, var(--gold) 10%, var(--surface));
	}
	.golden-boot-rank {
		display: grid;
		place-items: center;
		min-width: 2rem;
		font-family: var(--font-mono);
		font-size: 1.05rem;
		font-weight: 700;
		color: var(--text);
	}
	.golden-boot-player {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr);
		align-items: center;
		gap: 0.7rem;
		min-width: 0;
	}
	.golden-boot-photo {
		display: inline-grid;
		place-items: center;
		width: 34px;
		height: 34px;
		border-radius: 50%;
		background: var(--surface);
		border: 1px solid var(--border);
		object-fit: cover;
		flex: none;
	}
	.golden-boot-photo.fallback {
		font-family: var(--font-display);
		font-size: 0.72rem;
		font-weight: 800;
		color: var(--muted);
	}
	.golden-boot-main {
		display: grid;
		gap: 0.15rem;
		min-width: 0;
	}
	.golden-boot-main b {
		display: inline-flex;
		align-items: center;
		gap: 0.38rem;
		flex-wrap: wrap;
		min-width: 0;
		overflow-wrap: anywhere;
	}
	.golden-boot-tag {
		display: inline-flex;
		align-items: center;
		padding: 0.12rem 0.45rem;
		border-radius: 999px;
		background: color-mix(in srgb, var(--accent) 14%, transparent);
		color: var(--accent);
		font-size: 0.68rem;
		font-weight: 800;
		letter-spacing: 0.01em;
		white-space: nowrap;
	}
	.golden-boot-main i {
		font-style: normal;
		font-size: 0.78rem;
		color: var(--muted);
	}
	.golden-boot-goals {
		font-family: var(--font-mono);
		font-size: 0.9rem;
		color: var(--accent);
		white-space: nowrap;
	}
	.golden-boot-pick {
		padding-top: 0.2rem;
		border-top: 1px solid color-mix(in srgb, var(--border) 55%, transparent);
	}
	.home-feature-banner {
		grid-column: 1 / -1;
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.6rem 0.75rem 0.6rem 0.85rem;
		background: color-mix(in srgb, var(--accent) 14%, var(--surface));
		border: 1px solid color-mix(in srgb, var(--accent) 38%, var(--border));
		border-radius: 16px;
	}
	.hfb-main {
		display: flex;
		align-items: center;
		gap: 0.7rem;
		flex: 1;
		min-width: 0;
		color: var(--text);
		text-decoration: none;
	}
	.hfb-badge {
		flex: none;
		font-size: 0.64rem;
		font-weight: 800;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--accent-fg);
		background: var(--accent);
		padding: 0.2rem 0.5rem;
		border-radius: var(--radius-pill);
	}
	.hfb-copy {
		display: flex;
		flex-direction: column;
		gap: 0.05rem;
		min-width: 0;
		flex: 1;
	}
	.hfb-copy b {
		font-weight: 700;
		font-size: 0.92rem;
	}
	.hfb-copy .muted {
		font-size: 0.8rem;
	}
	.home-feature-banner :global(.hfb-go) {
		flex: none;
		color: var(--accent);
	}
	.hfb-x {
		flex: none;
		background: none;
		border: none;
		color: var(--muted);
		cursor: pointer;
		display: inline-flex;
		padding: 0.25rem;
		border-radius: 8px;
	}
	.hfb-x:hover {
		color: var(--text);
		background: color-mix(in srgb, var(--text) 8%, transparent);
	}
	.home-bento {
		gap: 0.75rem;
		grid-auto-rows: auto;
	}
	.home-bento .card {
		border-color: transparent;
		border-radius: 20px;
		box-shadow: 0 10px 30px -24px rgba(9, 9, 11, 0.28), var(--shadow-tile);
		min-height: auto;
	}
	.home-bento .card + .card {
		margin-top: 0;
	}
	.home-bento .card:hover {
		border-color: transparent;
		transform: none;
	}
	:global(:root[data-theme='dark']) .home-bento .card {
		border-color: color-mix(in srgb, var(--border-strong) 44%, var(--border));
	}
	:global(:root[data-theme='dark']) .home-bento .card:hover {
		border-color: color-mix(in srgb, var(--border-strong) 50%, var(--accent) 18%);
	}
	@media (prefers-color-scheme: dark) {
		:global(:root:not([data-theme])) .home-bento .card {
			border-color: color-mix(in srgb, var(--border-strong) 44%, var(--border));
		}
		:global(:root:not([data-theme])) .home-bento .card:hover {
			border-color: color-mix(in srgb, var(--border-strong) 50%, var(--accent) 18%);
		}
	}
	.action-card {
		grid-row: auto !important;
		gap: 0.95rem;
		padding: clamp(1rem, 3vw, 1.35rem);
		background: var(--surface);
	}
	.intro-card {
		position: relative;
		isolation: isolate;
		overflow: clip;
		gap: 1rem;
		padding: clamp(1.05rem, 3vw, 1.35rem);
		background:
			radial-gradient(circle at top right, color-mix(in srgb, var(--gold) 18%, transparent), transparent 34%),
			linear-gradient(135deg, color-mix(in srgb, var(--accent) 10%, var(--surface)) 0%, var(--surface) 55%, color-mix(in srgb, var(--gold) 10%, var(--surface)) 100%);
		border-color: color-mix(in srgb, var(--accent) 18%, var(--border)) !important;
	}
	.intro-card::after {
		content: '';
		position: absolute;
		right: -2.5rem;
		bottom: -2.75rem;
		z-index: 0;
		width: 12rem;
		aspect-ratio: 1;
		pointer-events: none;
		border-radius: 50%;
		background: radial-gradient(circle, color-mix(in srgb, var(--gold) 28%, transparent), transparent 70%);
		opacity: 0.7;
	}
	.intro-card > * {
		position: relative;
		z-index: 1;
	}
	.intro-head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 0.85rem;
	}
	.intro-dismiss {
		display: inline-grid;
		place-items: center;
		width: 2rem;
		height: 2rem;
		padding: 0;
		border: 1px solid color-mix(in srgb, var(--border) 70%, transparent);
		border-radius: 999px;
		background: color-mix(in srgb, var(--surface) 88%, transparent);
		color: var(--muted);
		cursor: pointer;
		flex: none;
		transition: color 0.18s ease, border-color 0.18s ease, background 0.18s ease;
	}
	.intro-dismiss:hover {
		color: var(--text);
		background: var(--surface);
		border-color: color-mix(in srgb, var(--accent) 25%, var(--border));
	}
	.intro-copy {
		display: grid;
		gap: 0.42rem;
		min-width: 0;
	}
	.intro-copy h2 {
		margin: 0;
		font-size: clamp(1.3rem, 4.6vw, 1.95rem);
		letter-spacing: 0;
	}
	.intro-copy p {
		max-width: 56ch;
		margin: 0;
		line-height: 1.45;
	}
	.intro-grid {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 0.65rem;
	}
	.intro-pill {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr);
		align-items: start;
		gap: 0.72rem;
		padding: 0.9rem;
		border-radius: 16px;
		background: color-mix(in srgb, var(--surface) 74%, transparent);
		box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--border) 64%, transparent);
	}
	.intro-pill-icon {
		display: grid;
		place-items: center;
		width: 2.4rem;
		height: 2.4rem;
		border-radius: 0.9rem;
		background: color-mix(in srgb, var(--accent) 12%, transparent);
		color: var(--accent);
	}
	.intro-pill-icon.match-tips {
		background: color-mix(in srgb, var(--warning) 12%, transparent);
		color: var(--warning);
	}
	.intro-pill-icon.worldcup {
		background: color-mix(in srgb, var(--gold) 15%, transparent);
		color: var(--gold);
	}
	.intro-pill b {
		display: block;
		font-size: 0.92rem;
		font-weight: 800;
		line-height: 1.15;
	}
	.intro-pill p {
		margin: 0.24rem 0 0;
		font-size: 0.82rem;
		line-height: 1.42;
		color: var(--muted);
	}
	.intro-actions {
		display: flex;
		align-items: center;
		justify-content: flex-start;
		gap: 0.85rem;
		flex-wrap: wrap;
	}
	.intro-links {
		display: flex;
		gap: 0.65rem;
		flex-wrap: wrap;
	}
	.action-topline,
	.action-copy,
	.tip-progress-panel,
	.match-mini {
		position: relative;
		z-index: 1;
	}
	.action-topline {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
	}
	.action-icon {
		display: grid;
		place-items: center;
		width: 40px;
		height: 40px;
		border-radius: 14px;
		background: color-mix(in srgb, var(--warning) 12%, transparent);
		color: var(--warning);
	}
	.action-icon.ok {
		background: color-mix(in srgb, var(--success) 14%, transparent);
		color: var(--success);
	}
	.action-icon.forecast {
		background: color-mix(in srgb, var(--accent) 12%, transparent);
		color: var(--accent);
	}
	.action-icon.plain-alert {
		background: transparent;
		border-radius: 0;
	}
	.urgency-chip {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		padding: 0.28rem 0.62rem;
		border-radius: var(--radius-pill);
		background: color-mix(in srgb, var(--warning) 10%, var(--surface-2));
		color: var(--warning);
		font-size: 0.78rem;
		font-weight: 700;
		letter-spacing: 0.01em;
	}
	.next-match-hint {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		flex-wrap: wrap;
		margin: 0.3rem 0 0;
		font-size: 0.84rem;
		line-height: 1.4;
	}
	.football-mark {
		width: 28px;
		height: 28px;
		display: block;
	}
	.football-mark-inline {
		width: 18px;
		height: 18px;
		display: inline-block;
		vertical-align: -3px;
		margin-right: 0.35rem;
	}
	/* removed unused: .action-copy h2, .action-copy p */
	.progress-compact {
		display: grid;
		grid-template-columns: 1fr auto;
		align-items: end;
		gap: 0.35rem 0.75rem;
	}
	.progress-compact span,
	.progress-compact b {
		font-family: var(--font-mono);
		font-weight: 800;
		font-variant-numeric: tabular-nums;
	}
	.progress-compact i {
		display: block;
		font-style: normal;
		font-size: 0.75rem;
		color: var(--muted);
		margin-top: 0.05rem;
	}
	/* removed unused: .progress-compact .progress-track */
	.tip-progress-panel {
		display: grid;
		gap: 0.4rem;
		padding: 0.35rem 0;
	}
	.tip-progress-info {
		display: flex;
		align-items: center;
		justify-content: space-between;
		font-size: 0.82rem;
		font-weight: 700;
	}
	.tip-progress-info b {
		font-family: var(--font-mono);
		font-weight: 850;
		color: var(--warning);
	}
	.tip-progress-info .muted {
		color: var(--muted);
		font-weight: 650;
	}
	.tip-progress-track {
		height: 6px;
		border-radius: var(--radius-pill);
		background: color-mix(in srgb, var(--surface-3) 50%, transparent);
		overflow: hidden;
	}
	.tip-progress-track span {
		display: block;
		width: var(--tip-progress, 0%);
		height: 100%;
		border-radius: inherit;
		background: var(--warning);
	}
	.match-mini {
		display: grid;
		gap: 0.5rem;
		padding: 0.65rem 0;
	}
	.match-meta {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		flex-wrap: wrap;
		font-size: 0.82rem;
		font-weight: 600;
		color: var(--muted);
	}
	.match-teams {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
		align-items: center;
		gap: 0.5rem;
	}
	.match-team {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		min-width: 0;
		justify-self: start;
	}
	.match-team.away {
		justify-self: end;
		flex-direction: row-reverse;
	}
	.match-team b {
		min-width: 0;
		font-weight: 750;
		line-height: 1.18;
		overflow-wrap: anywhere;
	}
	.match-vs {
		font-size: 0.65rem;
		font-weight: 800;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--muted);
		text-align: center;
	}
	.action-link {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.45rem;
		min-height: 44px;
		padding: 0.7rem 1rem;
		border-radius: var(--radius-pill);
		background: var(--text);
		color: var(--bg);
		font-weight: 800;
		width: 100%;
	}
	.done-link {
		background: var(--surface-2);
		color: var(--text);
		font-weight: 650;
	}
	.standing-card,
	.last-match-card {
		gap: 0.85rem;
	}
	.standing-card .hd {
		flex-wrap: wrap;
	}
	.standing-hero {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr);
		align-items: center;
		gap: 0.8rem;
		padding: 0.82rem 0.9rem;
		border-radius: 16px;
		background: color-mix(in srgb, var(--accent) 8%, var(--surface-2));
	}
	.rank-big {
		font-family: var(--font-display);
		font-size: clamp(2.1rem, 7vw, 3rem);
		font-weight: 850;
		line-height: 0.95;
		color: var(--accent);
	}
	.standing-copy {
		display: grid;
		gap: 0.08rem;
		min-width: 0;
	}
	.standing-copy b {
		font-size: 1.08rem;
		font-weight: 820;
	}
	.standing-copy i {
		font-style: normal;
		font-size: 0.82rem;
		line-height: 1.3;
		color: var(--muted);
		overflow-wrap: anywhere;
	}
	.mini-lb {
		display: grid;
		gap: 0.12rem;
	}
	.mini-lb-row {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		align-items: center;
		gap: 0.62rem;
		padding: 0.5rem 0;
		border-bottom: 1px solid color-mix(in srgb, var(--border) 55%, transparent);
		color: var(--text);
	}
	.mini-lb-row:last-child {
		border-bottom: none;
	}
	.mini-lb-row.me {
		color: var(--accent);
	}
	.mini-rank {
		display: inline-grid;
		place-items: center;
		min-width: 2.1rem;
		min-height: 1.75rem;
		padding: 0 0.35rem;
		border-radius: 10px;
		background: var(--surface-2);
		font-family: var(--font-mono);
		font-size: 0.78rem;
		font-weight: 850;
		color: var(--muted);
	}
	.mini-lb-row.me .mini-rank {
		background: color-mix(in srgb, var(--accent) 13%, var(--surface-2));
		color: var(--accent);
	}
	.mini-name {
		display: flex;
		align-items: center;
		gap: 0.38rem;
		min-width: 0;
	}
	.mini-name b {
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-size: 0.9rem;
	}
	.mini-name i {
		font-style: normal;
		font-size: 0.68rem;
		font-weight: 850;
		color: var(--accent);
	}
	.mini-lb-row strong {
		font-size: 0.9rem;
		font-weight: 850;
		font-family: var(--font-mono);
		white-space: nowrap;
	}
	.lm-score {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
		align-items: center;
		gap: 0.52rem;
		padding: 0.82rem 0.9rem;
		border-radius: 16px;
		background: var(--surface-2);
	}
	.lm-team {
		display: inline-flex;
		align-items: center;
		gap: 0.42rem;
		min-width: 0;
	}
	.lm-team.away {
		justify-content: flex-end;
		flex-direction: row-reverse;
	}
	.lm-team b {
		min-width: 0;
		font-size: 0.92rem;
		font-weight: 780;
		line-height: 1.15;
		overflow-wrap: anywhere;
	}
	.lm-scoreline {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 3.4rem;
		padding: 0.3rem 0.55rem;
		border-radius: 10px;
		background: var(--surface);
		font-weight: 850;
	}
	.lm-feedback {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.6rem;
		padding: 0.65rem 0.75rem;
		border-radius: 14px;
		background: color-mix(in srgb, var(--surface-2) 85%, transparent);
		font-size: 0.88rem;
		font-weight: 700;
	}
	.lm-feedback.plus {
		background: color-mix(in srgb, var(--success) 12%, var(--surface-2));
		color: var(--success);
	}
	.lm-feedback.zero {
		color: var(--muted);
	}
	.lm-feedback b {
		font-family: var(--font-mono);
		white-space: nowrap;
	}
	.lm-missing {
		margin: 0;
		font-size: 0.88rem;
	}
	.ready-list {
		display: grid;
		gap: 0.55rem;
	}
	.ready-item {
		display: grid;
		gap: 0.35rem;
		padding: 0.8rem 0.9rem;
		border-radius: 16px;
		background: color-mix(in srgb, var(--surface-2) 82%, transparent);
		color: var(--text);
	}
	.ready-meta,
	.ready-stage {
		font-size: 0.75rem;
		font-weight: 650;
		color: var(--muted);
	}
	.ready-meta {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		flex-wrap: wrap;
	}
	.ready-meta :global(.tv-logo.compact),
	.match-meta :global(.tv-logo.compact) {
		width: 60px;
		height: 19px;
	}
	.ready-stage {
		text-transform: uppercase;
		letter-spacing: 0.08em;
	}
	.ready-teams {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
		align-items: center;
		gap: 0.45rem;
	}
	.ready-team {
		display: inline-flex;
		align-items: center;
		gap: 0.42rem;
		min-width: 0;
	}
	.ready-team.away {
		justify-content: flex-end;
	}
	.ready-team b {
		min-width: 0;
		font-size: 0.92rem;
		font-weight: 760;
		overflow-wrap: anywhere;
	}
	.ready-vs {
		font-size: 0.68rem;
		font-weight: 800;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--muted);
	}

	.chat-preview-card {
		gap: 0.85rem;
	}
	.chat-kicker {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
	}
	.chat-preview-list {
		display: grid;
		gap: 0.38rem;
	}
	.chat-error {
		margin: 0;
		padding: 0.65rem 0;
	}
	.chat-preview {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		align-items: center;
		gap: 0.62rem;
		padding: 0.58rem 0;
		border-bottom: 1px solid color-mix(in srgb, var(--border) 55%, transparent);
		color: var(--text);
	}
	.chat-preview:last-child {
		border-bottom: none;
	}
	.chat-preview.unread .chat-title b {
		font-weight: 850;
	}
	.chat-badge {
		display: grid;
		place-items: center;
		width: 32px;
		height: 32px;
		border-radius: 10px;
		background: var(--surface-2);
		color: var(--muted);
	}
	.chat-preview.unread .chat-badge {
		background: color-mix(in srgb, var(--accent) 12%, var(--surface-2));
		color: var(--accent);
	}
	.chat-main,
	.chat-title,
	.chat-text,
	.chat-jump {
		min-width: 0;
	}
	.chat-main {
		display: grid;
		gap: 0.1rem;
	}
	.chat-title {
		display: flex;
		align-items: center;
		gap: 0.45rem;
	}
	.chat-title b,
	.chat-text {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.chat-title b {
		font-size: 0.9rem;
	}
	.chat-title i {
		display: inline-flex;
		align-items: center;
		min-height: 20px;
		padding: 0.12rem 0.38rem;
		border-radius: var(--radius-pill);
		background: var(--accent);
		color: var(--bg);
		font-style: normal;
		font-size: 0.68rem;
		font-weight: 850;
	}
	.chat-text {
		display: block;
		font-size: 0.82rem;
		color: var(--muted);
	}
	.chat-text strong {
		color: var(--text);
	}
	.chat-jump {
		display: inline-flex;
		align-items: center;
		justify-content: flex-end;
		gap: 0.2rem;
		color: var(--muted);
		font-size: 0.75rem;
		font-weight: 750;
		white-space: nowrap;
	}
	.chat-preview:hover .chat-jump {
		color: var(--accent);
	}
	.chat-preview-card.home-span-support .chat-preview {
		grid-template-columns: auto minmax(0, 1fr);
		align-items: start;
	}
	.chat-preview-card.home-span-support .chat-jump {
		grid-column: 2;
		justify-content: flex-start;
		font-size: 0.72rem;
	}

	.standing-card .hd,
	.progress-card .hd,
	.results .hd,
	.golden-boot-card .hd,
	.podium-card .hd {
		align-items: center;
	}
	.progress-card {
		display: grid;
		gap: 0.95rem;
	}
	/* Hero: big total + trend chip */
	.trend-hero {
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: 0.75rem;
	}
	.trend-total {
		display: grid;
		gap: 0.2rem;
	}
	.trend-label {
		font-size: 0.62rem;
		font-weight: 700;
		letter-spacing: 0.16em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.trend-num {
		font-family: var(--font-display);
		font-size: clamp(2.1rem, 7vw, 2.7rem);
		line-height: 0.9;
		color: var(--text);
	}
	.trend-num em {
		font-style: normal;
		font-size: 0.46em;
		font-weight: 700;
		color: var(--muted);
		margin-left: 0.18rem;
	}
	.trend-chip {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		padding: 0.34rem 0.62rem;
		border-radius: 999px;
		background: color-mix(in srgb, var(--success) 15%, transparent);
		color: var(--success);
		font-weight: 800;
		font-size: 0.84rem;
		white-space: nowrap;
	}
	.trend-chip.flat {
		background: color-mix(in srgb, var(--muted) 16%, transparent);
		color: var(--muted);
	}
	.trend-chip i {
		font-style: normal;
		font-weight: 600;
		font-size: 0.72rem;
		opacity: 0.85;
	}
	/* Form strip + caption */
	.trend-form {
		display: grid;
		gap: 0.4rem;
	}
	.trend-cap {
		font-size: 0.6rem;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--muted);
	}
	/* Stats row */
	.trend-stats {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 0.45rem;
		padding-top: 0.85rem;
		border-top: 1px solid var(--border);
	}
	.trend-stats span {
		display: grid;
		gap: 0.22rem;
		justify-items: start;
	}
	.trend-stats b {
		font-family: var(--font-display);
		font-size: 1.2rem;
		line-height: 1;
		color: var(--accent);
	}
	.trend-stats b em {
		font-style: normal;
		font-size: 0.6em;
		color: var(--muted);
		margin-left: 0.06rem;
	}
	.trend-stats i {
		font-style: normal;
		font-size: 0.56rem;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--muted);
	}
	/* removed unused: .st li variants and related rules */

	.podium-card {
		gap: 0.85rem;
		grid-template-columns: 1fr !important;
		align-items: stretch !important;
		align-content: start;
		align-self: start;
	}
	.podium-list {
		display: grid;
		gap: 0.12rem;
		align-content: start;
	}
	.podium-row {
		display: grid;
		grid-template-columns: auto minmax(7.5rem, 0.65fr) minmax(0, 1fr);
		align-items: center;
		gap: 0.7rem;
		padding: 0.55rem 0;
		border-bottom: 1px solid color-mix(in srgb, var(--border) 55%, transparent);
	}
	.podium-row:last-child {
		border-bottom: none;
	}
	.medal {
		display: grid;
		place-items: center;
		width: 32px;
		height: 32px;
		border-radius: 50%;
		font-family: var(--font-mono);
		font-weight: 900;
		background: color-mix(in srgb, var(--surface-3) 70%, transparent);
		color: var(--muted);
	}
	.place-1 .medal {
		background: color-mix(in srgb, var(--gold) 18%, transparent);
		color: var(--gold);
	}
	.place-2 .medal {
		background: rgba(161, 161, 170, 0.14);
		color: #71717a;
	}
	.place-3 .medal {
		background: rgba(217, 119, 6, 0.13);
		color: #b45309;
	}
	.podium-row i,
	.podium-row b {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		min-width: 0;
	}
	.podium-row > span:last-child {
		display: contents;
		min-width: 0;
	}
	.podium-row i {
		font-style: normal;
		font-size: 0.72rem;
		font-weight: 700;
		color: var(--muted);
		text-transform: uppercase;
		letter-spacing: 0.08em;
	}
	.podium-row b {
		justify-self: end;
		margin-top: 0;
		font-size: 0.94rem;
		text-align: right;
		overflow-wrap: anywhere;
	}
	.podium-row b :global(.flag) {
		flex: 0 0 auto;
	}
	@media (max-width: 899px) {
		.hero-chip.league-pill.mobile-hide {
			display: none !important;
		}
		.hero-chip.points-pill {
			margin-left: auto;
		}
		.podium-row {
			grid-template-columns: auto 1fr;
		}
		.podium-row > span:last-child {
			display: block;
		}
		.podium-row b {
			justify-self: start;
			margin-top: 0.08rem;
			text-align: left;
		}
	}

	@media (max-width: 639px) {
		.home-bento {
			grid-template-columns: 1fr;
			gap: 0.55rem;
		}
		.home-bento .card {
			padding: 1rem;
			border-radius: 18px;
			box-shadow: 0 1px 6px rgba(9, 9, 11, 0.06);
		}
		/* Mobile card order */
		.action-card       { order: 1; }
		.progress-card     { order: 2; }
		.standing-card     { order: 2; }
		.last-match-card   { order: 3; }
		.next-card         { order: 4; }
		.results           { order: 5; }
		.chat-preview-card { order: 6; }
		.golden-boot-card  { order: 7; }
		.podium-card       { order: 7; }
		.match-mini {
			padding: 0.5rem 0;
		}
		.golden-boot-row {
			grid-template-columns: auto minmax(0, 1fr);
			align-items: start;
			row-gap: 0.45rem;
		}
		.golden-boot-goals {
			grid-column: 2;
			justify-self: start;
			font-size: 0.82rem;
		}
		.golden-boot-photo {
			width: 30px;
			height: 30px;
		}
		.golden-boot-main b {
			gap: 0.3rem;
		}
		.golden-boot-tag {
			font-size: 0.64rem;
		}
		.ready-item {
			padding: 0.75rem 0.8rem;
		}
		.ready-teams {
			grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
			gap: 0.4rem;
		}
		.ready-vs {
			display: inline;
			font-size: 0.62rem;
		}
		.ready-team.away {
			justify-content: flex-end;
		}
		.match-teams {
			grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
			gap: 0.4rem;
		}
		.match-vs {
			font-size: 0.62rem;
		}
		.match-team.away {
			justify-self: end;
			flex-direction: row-reverse;
		}
		.podium-card {
			padding-bottom: 0.75rem;
		}
	}
	@media (min-width: 640px) {
		.home-span-primary,
		.home-span-support {
			grid-column: span 4 !important;
		}
	}
	@media (min-width: 900px) {
		.home-bento {
			grid-auto-flow: row dense;
		}
		.home-hero {
			grid-template-columns: 1fr auto;
			align-items: end;
			margin-bottom: 1rem;
		}
		.hero-chips {
			justify-content: flex-end;
		}
		.home-span-primary { grid-column: span 6 !important; }
		.home-span-support { grid-column: span 3 !important; }
	}
	@media (min-width: 1200px) {
		.home-span-primary { grid-column: span 8 !important; }
		.home-span-support { grid-column: span 4 !important; }
	}
	@media (min-width: 1400px) {
		.home-span-primary { grid-column: span 8 !important; }
		.home-span-support { grid-column: span 4 !important; }
	}

	/* ===== Dynamic tournament dashboard ===== */
	.now-topline {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
	}
	.now-icon {
		display: grid;
		place-items: center;
		width: 42px;
		height: 42px;
		border-radius: 14px;
		background: color-mix(in srgb, var(--success) 13%, var(--surface-2));
		color: var(--success);
		flex: none;
	}
	.now-icon.urgent {
		background: color-mix(in srgb, var(--warning) 14%, var(--surface-2));
		color: var(--warning);
	}
	.now-icon.live {
		background: color-mix(in srgb, var(--live) 15%, var(--surface-2));
		color: var(--live);
	}
	.now-icon.result {
		background: color-mix(in srgb, var(--accent) 15%, var(--surface-2));
		color: var(--accent);
	}
	.now-icon.plain-alert {
		width: auto;
		height: auto;
		padding: 0;
		background: transparent;
		border: 0;
		border-radius: 0;
		box-shadow: none;
	}
	.now-copy h2 {
		font-size: clamp(1.45rem, 5vw, 2.15rem);
		letter-spacing: 0;
	}
	.now-copy p {
		margin: 0.4rem 0 0;
		max-width: 50ch;
		line-height: 1.45;
	}
	.now-match {
		display: grid;
		gap: 0.5rem;
		padding: 0.8rem 0.9rem;
		border-radius: 16px;
		background: color-mix(in srgb, var(--surface-2) 82%, transparent);
	}
	.now-result-card {
		padding: 0.15rem 0.9rem;
		border-radius: 16px;
		background: color-mix(in srgb, var(--surface-2) 82%, transparent);
	}
	.now-live-matches {
		display: grid;
		gap: 0.7rem;
	}
	.now-teams {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
		align-items: center;
		gap: 0.6rem;
	}
	.now-teams span {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		min-width: 0;
	}
	.now-teams span.away {
		justify-content: flex-end;
	}
	.now-teams b {
		min-width: 0;
		font-weight: 800;
		overflow-wrap: anywhere;
	}
	.now-teams strong {
		display: inline-flex;
		justify-content: center;
		min-width: 3.25rem;
		padding: 0.28rem 0.55rem;
		border-radius: 10px;
		background: var(--surface);
		font-weight: 850;
		color: var(--text);
	}
	.now-meta {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex-wrap: wrap;
		color: var(--muted);
		font-size: 0.78rem;
		font-weight: 650;
	}
	.now-live-pill {
		display: inline-flex;
		align-items: center;
		gap: 0.32rem;
		padding: 0.2rem 0.55rem;
		border-radius: 999px;
		background: color-mix(in srgb, var(--live) 14%, transparent);
		border: 1px solid color-mix(in srgb, var(--live) 40%, transparent);
		color: var(--live);
		font-size: 0.7rem;
		font-weight: 800;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}
	.live-dot {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		background: var(--live);
		animation: nowLivePulse 1.8s ease-out infinite;
	}
	@keyframes nowLivePulse {
		0% {
			box-shadow: 0 0 0 0 color-mix(in srgb, var(--live) 55%, transparent);
		}
		70% {
			box-shadow: 0 0 0 6px color-mix(in srgb, var(--live) 0%, transparent);
		}
		100% {
			box-shadow: 0 0 0 0 color-mix(in srgb, var(--live) 0%, transparent);
		}
	}
	/* Points-trend form strip */
	.form-bars {
		display: flex;
		align-items: stretch;
		gap: 3px;
		height: 48px;
		margin: 0;
		padding-bottom: 3px;
		border-bottom: 1px solid var(--border);
	}
	.form-col {
		position: relative;
		flex: 1;
		min-width: 4px;
		display: flex;
		align-items: flex-end;
	}
	.form-bar {
		width: 100%;
		height: var(--h, 10%);
		border-radius: 3px 3px 0 0;
		background: linear-gradient(
			180deg,
			var(--accent),
			color-mix(in srgb, var(--accent) 55%, transparent)
		);
		transition: filter 0.12s ease;
	}
	.form-col:hover .form-bar {
		filter: brightness(1.15);
	}
	.form-bar-tip {
		position: absolute;
		bottom: calc(100% + 6px);
		left: 50%;
		transform: translateX(-50%) translateY(3px);
		padding: 0.32rem 0.5rem;
		border-radius: 8px;
		background: var(--surface-3);
		color: var(--text);
		border: 1px solid var(--border);
		box-shadow: 0 10px 24px -12px rgba(0, 0, 0, 0.7);
		font-size: 0.68rem;
		font-weight: 700;
		white-space: nowrap;
		opacity: 0;
		pointer-events: none;
		transition: opacity 0.12s ease, transform 0.12s ease;
		z-index: 6;
	}
	.form-col:hover .form-bar-tip {
		opacity: 1;
		transform: translateX(-50%) translateY(0);
	}
	.form-bar.zero {
		background: color-mix(in srgb, var(--muted) 26%, transparent);
	}
	.form-bar.exact {
		background: linear-gradient(
			180deg,
			var(--gold),
			color-mix(in srgb, var(--gold) 50%, transparent)
		);
		box-shadow: 0 0 8px -2px color-mix(in srgb, var(--gold) 70%, transparent);
	}
	@media (prefers-reduced-motion: reduce) {
		.live-dot,
		.league-rank-skeleton,
		.league-rank-skeleton::after {
			animation: none;
		}
	}
	.now-meta :global(.tv-logo.compact) {
		width: 60px;
		height: 19px;
	}
	.action-card.live {
		border-color: color-mix(in srgb, var(--live) 28%, var(--border));
	}
	.action-card.result {
		border-color: color-mix(in srgb, var(--accent) 24%, var(--border));
	}
	
	.action-card.tourney-gold {
		border-color: color-mix(in srgb, var(--gold) 60%, var(--border));
		background: radial-gradient(circle at top right, color-mix(in srgb, var(--gold) 20%, var(--surface)) 0%, var(--surface) 120%);
		position: relative;
		overflow: hidden;
	}
	.action-card.tourney-gold::after {
		content: '🎊';
		position: absolute;
		right: 1.5rem;
		top: -0.5rem;
		font-size: 6rem;
		opacity: 0.15;
		transform: rotate(15deg);
		pointer-events: none;
	}
	.now-icon.gold {
		background: color-mix(in srgb, var(--gold) 15%, transparent);
		color: var(--gold);
	}

	.action-card.tourney-over {
		border-color: color-mix(in srgb, var(--gold) 40%, var(--border));
		background: linear-gradient(135deg, var(--surface) 0%, color-mix(in srgb, var(--gold) 6%, var(--surface)) 100%);
	}
	.now-icon.done {
		background: color-mix(in srgb, var(--gold) 15%, transparent);
		color: var(--gold);
	}
	.gold-points {
		color: var(--gold);
		font-weight: 900;
	}
	.since-card {
		gap: 0.85rem;
	}
	.since-grid {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 0.55rem;
	}
	.since-item {
		display: grid;
		gap: 0.15rem;
		padding: 0.7rem 0.75rem;
		border-radius: 14px;
		background: var(--surface-2);
		min-width: 0;
	}
	.since-item span {
		color: var(--muted);
		font-size: 0.72rem;
		font-weight: 750;
	}
	.since-item b {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		font-family: var(--font-mono);
		font-size: 0.92rem;
		color: var(--text);
		white-space: nowrap;
	}
	.since-item.good b {
		color: var(--success);
	}
	.since-result {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
		padding: 0.65rem 0;
		border-top: 1px solid var(--border);
		color: var(--text);
		font-weight: 700;
	}
	.league-gaps {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 0.5rem;
	}
	.league-gaps span {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		align-items: baseline;
		gap: 0.35rem;
		padding: 0.6rem 0.7rem;
		border-radius: 14px;
		background: var(--surface-2);
		min-width: 0;
	}
	.league-gaps span:only-child {
		grid-column: 1 / -1;
	}
	.league-gaps i,
	.league-gaps em {
		font-style: normal;
		font-size: 0.72rem;
		color: var(--muted);
		font-weight: 700;
		white-space: nowrap;
	}
	.league-gaps b {
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-size: 0.88rem;
	}
	.forecast-pulse-card {
		gap: 0.8rem;
	}
	.forecast-pulse-card.urgent {
		border-color: color-mix(in srgb, var(--warning) 28%, var(--border)) !important;
	}
	.forecast-pulse-card.out {
		border-color: color-mix(in srgb, var(--danger) 22%, var(--border)) !important;
	}
	.forecast-copy {
		margin: 0;
		line-height: 1.42;
	}
	.forecast-mini-podium {
		position: relative;
		display: grid;
		gap: 0.2rem;
	}
	.forecast-mini-podium span {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr);
		align-items: center;
		gap: 0.55rem;
		padding: 0.45rem 0;
		border-top: 1px solid color-mix(in srgb, var(--border) 60%, transparent);
	}
	.forecast-mini-podium i {
		display: grid;
		place-items: center;
		width: 26px;
		height: 26px;
		border-radius: 50%;
		background: color-mix(in srgb, var(--gold) 14%, var(--surface-2));
		color: var(--gold);
		font-style: normal;
		font-family: var(--font-mono);
		font-weight: 900;
	}
	.forecast-mini-podium b {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		min-width: 0;
		font-size: 0.9rem;
		overflow-wrap: anywhere;
	}
	.ready-item.missing {
		background: color-mix(in srgb, var(--warning) 9%, var(--surface-2));
		box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--warning) 22%, transparent);
	}
	.ready-state {
		display: inline-flex;
		align-items: center;
		min-height: 20px;
		padding: 0.12rem 0.42rem;
		border-radius: var(--radius-pill);
		font-style: normal;
		font-size: 0.68rem;
		font-weight: 850;
	}
	.ready-state.ok {
		color: var(--success);
		background: color-mix(in srgb, var(--success) 11%, transparent);
	}
	.ready-state.warn {
		color: var(--warning);
		background: color-mix(in srgb, var(--warning) 12%, transparent);
	}
	.chat-preview-card.has-unread {
		border-color: color-mix(in srgb, var(--accent) 24%, var(--border)) !important;
	}

	@media (max-width: 639px) {
		.intro-head {
			align-items: stretch;
		}
		.intro-grid {
			grid-template-columns: 1fr;
		}
		.intro-actions {
			align-items: flex-start;
		}
		.chat-preview-card.has-unread { order: 3; }
		.progress-card     { order: 2; }
		.standing-card     { order: 4; }
		.next-card         { order: 5; }
		.results           { order: 6; }
		.forecast-pulse-card { order: 7; }
		.chat-preview-card:not(.has-unread) { order: 8; }
		.golden-boot-card  { order: 9; }
		.podium-card       { order: 9; }
		.league-gaps {
			grid-template-columns: 1fr;
		}
	}
	@media (min-width: 900px) {
		.chat-preview-card.has-unread {
			order: 3;
		}
		.forecast-pulse-card {
			order: 6;
		}
	}

	:global(:root[data-theme='worldcup']) .home-hero {
		position: relative;
	}
	:global(:root[data-theme='worldcup']) .home-hero::before {
		content: none;
	}
	:global(:root[data-theme='worldcup']) .home-hero .kicker,
	:global(:root[data-theme='worldcup']) .now-topline .kicker {
		color: var(--gold);
	}
	:global(:root[data-theme='worldcup']) .hero-greeting .greet {
		color: color-mix(in srgb, var(--text) 82%, var(--muted));
	}
	:global(:root[data-theme='worldcup']) .hero-greeting .name {
		background: linear-gradient(110deg, #f3dfae 8%, var(--gold) 44%, var(--accent) 100%);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
		background-clip: text;
	}
	:global(:root[data-theme='worldcup']) .hero-chip {
		border: 1px solid color-mix(in srgb, var(--gold) 18%, var(--border));
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.045), transparent),
			color-mix(in srgb, var(--surface) 86%, transparent);
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.05), 0 10px 24px -20px rgba(0, 0, 0, 0.86);
	}
	:global(:root[data-theme='worldcup']) .hero-chip b {
		color: var(--gold);
	}
	:global(:root[data-theme='worldcup']) .hero-chip.points-pill {
		border-color: rgba(232, 197, 116, 0.58);
		background: linear-gradient(180deg, #f1d78f 0%, #d8b86c 100%);
		color: #071019;
		box-shadow: 0 14px 30px -22px rgba(232, 197, 116, 0.68);
	}
	:global(:root[data-theme='worldcup']) .hero-chip.points-pill span {
		color: rgba(7, 16, 25, 0.72);
	}
	:global(:root[data-theme='worldcup']) .hero-chip.points-pill b {
		background: rgba(7, 16, 25, 0.92);
		color: #f7e8bd;
		box-shadow: inset 0 0 0 1px rgba(247, 232, 189, 0.22);
	}

	:global(:root[data-theme='worldcup']) .home-bento .card {
		background:
			radial-gradient(circle at 18% 0%, rgba(217, 187, 114, 0.075), transparent 31%),
			linear-gradient(135deg, rgba(217, 187, 114, 0.045), transparent 38%),
			linear-gradient(180deg, rgba(16, 39, 47, 0.94), rgba(8, 20, 29, 0.98)),
			var(--surface);
		border-color: color-mix(in srgb, var(--gold) 9%, var(--border));
		box-shadow:
			inset 0 1px 0 rgba(246, 225, 176, 0.08),
			inset 0 0 0 1px rgba(217, 187, 114, 0.025),
			0 18px 44px -36px rgba(217, 187, 114, 0.24),
			0 20px 48px -34px rgba(0, 0, 0, 0.9);
	}
	:global(:root[data-theme='worldcup']) .home-bento .card:hover {
		border-color: color-mix(in srgb, var(--accent) 24%, var(--border));
	}
	:global(:root[data-theme='worldcup']) .home-bento .card::before {
		background:
			linear-gradient(118deg, rgba(246, 225, 176, 0.075) 0%, rgba(217, 187, 114, 0.03) 28%, transparent 52%),
			radial-gradient(circle at 16% 4%, rgba(143, 197, 143, 0.09), transparent 26%);
		opacity: 0.58;
		mask-image: linear-gradient(135deg, rgba(0, 0, 0, 0.9), transparent 78%);
	}
	:global(:root[data-theme='worldcup']) .home-bento .action-card.card {
		background:
			linear-gradient(90deg, rgba(7, 16, 25, 0.9), rgba(7, 16, 25, 0.42) 58%, rgba(7, 16, 25, 0.86)),
			radial-gradient(circle at 30% 7%, rgba(248, 222, 152, 0.2), transparent 33%),
			linear-gradient(180deg, rgba(15, 45, 37, 0.35), rgba(8, 20, 29, 0.82)),
			url('/theme/field-clean.png') center 48% / cover no-repeat,
			var(--surface);
		background-blend-mode: normal, screen, multiply, normal, normal;
	}
	:global(:root[data-theme='worldcup']) .home-bento .action-card.card::before {
		background:
			linear-gradient(120deg, rgba(246, 225, 176, 0.12), rgba(217, 187, 114, 0.035) 30%, transparent 58%),
			linear-gradient(90deg, rgba(143, 197, 143, 0.05), transparent 48%, rgba(217, 187, 114, 0.035));
		opacity: 0.64;
		mask-image: linear-gradient(135deg, rgba(0, 0, 0, 0.92), transparent 88%);
	}
	:global(:root[data-theme='worldcup']) .intro-card {
		background:
			linear-gradient(90deg, rgba(7, 16, 25, 0.88), rgba(7, 16, 25, 0.38) 56%, rgba(7, 16, 25, 0.82)),
			radial-gradient(circle at 18% 0%, rgba(248, 222, 152, 0.18), transparent 34%),
			linear-gradient(135deg, rgba(15, 45, 37, 0.5), rgba(8, 20, 29, 0.94)),
			url('/theme/field-clean.png') center 48% / cover no-repeat,
			var(--surface);
		background-blend-mode: normal, screen, multiply, normal, normal;
		border-color: color-mix(in srgb, var(--gold) 24%, var(--border)) !important;
	}
	:global(:root[data-theme='worldcup']) .intro-pill {
		background: color-mix(in srgb, var(--surface) 78%, rgba(7, 16, 25, 0.14));
		box-shadow: inset 0 0 0 1px rgba(232, 197, 116, 0.08);
	}
	:global(:root[data-theme='worldcup']) .intro-dismiss {
		background: rgba(7, 16, 25, 0.74);
		border-color: rgba(232, 197, 116, 0.18);
	}
	:global(:root[data-theme='worldcup']) .forecast-pulse-card::after {
		content: '';
		position: absolute;
		right: -2.1rem;
		bottom: -2.65rem;
		z-index: 0;
		width: min(14.5rem, 52%);
		aspect-ratio: 1;
		pointer-events: none;
		background: url('/theme/pokal.png') right bottom / contain no-repeat;
		filter: brightness(0.74) contrast(0.96) saturate(0.78);
		mix-blend-mode: normal;
		opacity: 0.23;
		mask-image: radial-gradient(circle at 66% 64%, black 0 42%, rgba(0, 0, 0, 0.5) 56%, transparent 74%);
	}
	:global(:root[data-theme='worldcup']) .tile .hd h3 {
		color: color-mix(in srgb, var(--text) 94%, var(--gold));
		font-weight: 800;
	}
	:global(:root[data-theme='worldcup']) .tile .hd h3 :global(svg),
	:global(:root[data-theme='worldcup']) .now-icon :global(svg) {
		filter: drop-shadow(0 0 10px rgba(232, 197, 116, 0.18));
	}
	:global(:root[data-theme='worldcup']) .hdlink {
		color: var(--gold);
	}

	:global(:root[data-theme='worldcup']) .now-match,
	:global(:root[data-theme='worldcup']) .standing-hero,
	:global(:root[data-theme='worldcup']) .league-gaps span,
	:global(:root[data-theme='worldcup']) .lm-score {
		border: 1px solid color-mix(in srgb, var(--gold) 14%, var(--border));
		background:
			linear-gradient(135deg, rgba(143, 197, 143, 0.08), transparent 58%),
			color-mix(in srgb, var(--surface-2) 82%, transparent);
	}
	:global(:root[data-theme='worldcup']) .now-icon,
	:global(:root[data-theme='worldcup']) .action-icon,
	:global(:root[data-theme='worldcup']) .chat-badge,
	:global(:root[data-theme='worldcup']) .mini-rank,
	:global(:root[data-theme='worldcup']) .ready-state,
	:global(:root[data-theme='worldcup']) .forecast-mini-podium i,
	:global(:root[data-theme='worldcup']) .medal {
		border-radius: 999px;
		border: 1px solid color-mix(in srgb, var(--gold) 16%, transparent);
		background: color-mix(in srgb, var(--surface-3) 70%, transparent);
	}
	:global(:root[data-theme='worldcup']) .now-icon,
	:global(:root[data-theme='worldcup']) .action-icon.ok,
	:global(:root[data-theme='worldcup']) .action-icon.forecast {
		color: var(--gold);
		background: color-mix(in srgb, var(--gold) 11%, var(--surface-2));
	}
	:global(:root[data-theme='worldcup']) .now-icon.plain-alert {
		background: transparent;
		border: 0;
		box-shadow: none;
	}
	:global(:root[data-theme='worldcup']) .action-link {
		background: linear-gradient(180deg, #f1d78f 0%, #d8b86c 100%);
		color: #071019;
		box-shadow: 0 16px 32px -24px rgba(232, 197, 116, 0.74);
	}
	:global(:root[data-theme='worldcup']) .now-teams strong {
		background: rgba(5, 13, 20, 0.72);
		border: 1px solid color-mix(in srgb, var(--gold) 13%, var(--border));
		border-radius: 999px;
	}

	:global(:root[data-theme='worldcup']) .ready-item {
		position: relative;
		overflow: hidden;
		border: 1px solid color-mix(in srgb, var(--gold) 12%, var(--border));
		background:
			linear-gradient(90deg, rgba(143, 197, 143, 0.1), transparent 18%),
			color-mix(in srgb, var(--surface-2) 82%, transparent);
	}
	:global(:root[data-theme='worldcup']) .ready-item::before {
		content: '';
		position: absolute;
		inset: 0 auto 0 0;
		width: 3px;
		background: linear-gradient(180deg, var(--accent), var(--gold));
		opacity: 0.68;
	}
	:global(:root[data-theme='worldcup']) .ready-item > * {
		position: relative;
		z-index: 1;
	}
	:global(:root[data-theme='worldcup']) .ready-stage {
		color: color-mix(in srgb, var(--gold) 68%, var(--muted));
	}
	:global(:root[data-theme='worldcup']) .ready-team :global(.flag),
	:global(:root[data-theme='worldcup']) .now-teams :global(.flag),
	:global(:root[data-theme='worldcup']) .forecast-mini-podium :global(.flag),
	:global(:root[data-theme='worldcup']) .podium-row :global(.flag) {
		border-color: rgba(232, 197, 116, 0.28);
		box-shadow: 0 0 0 2px rgba(232, 197, 116, 0.08), 0 0 14px rgba(143, 197, 143, 0.1);
	}
	:global(:root[data-theme='worldcup']) .mini-lb-row,
	:global(:root[data-theme='worldcup']) .chat-preview,
	:global(:root[data-theme='worldcup']) .podium-row {
		border-bottom-color: color-mix(in srgb, var(--gold) 12%, var(--border));
	}
	:global(:root[data-theme='worldcup']) .chat-title i,
	:global(:root[data-theme='worldcup']) .ready-state.ok {
		background: color-mix(in srgb, var(--accent) 16%, transparent);
		color: var(--accent);
	}
	:global(:root[data-theme='worldcup']) .ready-state.warn {
		background: color-mix(in srgb, var(--gold) 14%, transparent);
		color: var(--gold);
	}
</style>
