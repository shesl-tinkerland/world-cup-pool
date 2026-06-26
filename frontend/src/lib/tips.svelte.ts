import { pb } from './pb';
import { auth } from './auth.svelte';
import { serverClock } from './serverclock.svelte';
import { applyRealtimeMatchBatch } from './tipsRealtime';

export interface Team {
	id: string;
	name: string;
	iso2: string;
	fifaCode: string;
}

export interface Match {
	id: string;
	stage: string; // group | R32 | R16 | QF | SF | 3RD | FINAL
	groupLetter: string;
	roundLabel: string;
	num: number;
	kickoff: string;
	tvChannel: string;
	status: string;
	homeTeam: string;
	awayTeam: string;
	homeLabel: string;
	awayLabel: string;
	ftHome: number;
	ftAway: number;
	etHome: number;
	etAway: number;
	penHome: number;
	penAway: number;
	advancer: string;
	finalizedAt: string;
}

export interface Tip {
	id?: string;
	match: string;
	ftHome: number;
	ftAway: number;
	etHome: number;
	etAway: number;
	penWinner: string;
	advancer: string;
}

export interface MatchOdds {
	matchId: string;
	pHome: number;
	pDraw: number;
	pAway: number;
	homeOdds: number;
	drawOdds: number;
	awayOdds: number;
}

export interface LiveEvent {
	id: string;
	match: string;
	providerKey: string;
	created: string;
	elapsed: number;
	extra: number;
	type: 'Goal' | 'Card' | 'subst' | 'Var' | string;
	detail: string;
	player: string;
	assist: string;
	team: string;
	teamId: string;
	comments: string;
}

export interface FriendTip {
	userId: string;
	name: string;
	isMe: boolean;
	hasTip?: boolean;
	ftHome: number;
	ftAway: number;
	etHome: number;
	etAway: number;
	penWinner: string;
	advancer: string;
	points: number; // -1 = points not available yet, or no tip when hasTip=false
}

class TipsStore {
	teams = $state<Record<string, Team>>({});
	matches = $state<Match[]>([]);
	tips = $state<Record<string, Tip>>({}); // keyed by matchId
	scores = $state<Record<string, number>>({}); // matchId -> points (default cfg)
	tournamentGroups = $state<Record<string, string[]>>({}); // letter -> teamIds
	odds = $state<Record<string, MatchOdds>>({}); // keyed by matchId
	oddsSource = $state<'odds_api' | 'rankings' | 'none'>('none');
	loaded = $state(false);
	liveMatchIds = $state<Set<string>>(new Set());
	liveEvents = $state<Record<string, LiveEvent[]>>({});
	scoreRevision = $state(0);
	private loadPromise: Promise<void> | null = null;
	private _matchUnsub: (() => void) | null = null;
	private _tipUnsub: (() => void) | null = null;
	private _eventUnsub: (() => void) | null = null;
	private _matchScoreUnsub: (() => void) | null = null;
	private _forecastScoreUnsub: (() => void) | null = null;
	private scoreRefreshTimer: ReturnType<typeof setTimeout> | null = null;
	private pendingMatchUpdates = new Map<string, Match>();
	private pendingMatchTimer: ReturnType<typeof setTimeout> | null = null;

	private setLiveMatchState(matchId: string, live: boolean) {
		const next = new Set(this.liveMatchIds);
		if (live) next.add(matchId);
		else next.delete(matchId);
		this.liveMatchIds = next;
		if (!live && this.liveEvents[matchId]) {
			const events = { ...this.liveEvents };
			delete events[matchId];
			this.liveEvents = events;
		}
	}

	async load() {
		if (this.loaded) return;
		if (this.loadPromise) return this.loadPromise;
		this.loadPromise = this.loadInner().finally(() => {
			this.loadPromise = null;
		});
		return this.loadPromise;
	}

	async refresh() {
		if (this.loadPromise) return this.loadPromise;
		this.loadPromise = this.loadInner().finally(() => {
			this.loadPromise = null;
		});
		return this.loadPromise;
	}

	private async loadInner() {
		const wasLoaded = this.loaded;
		const previousFinalized = new Map(
			this.matches.map((match) => [match.id, match.finalizedAt])
		);
		const [teams, matches, mine, tgroups, , scoresResult] = await Promise.all([
			pb.collection('teams').getFullList({ sort: 'name' }),
			pb.collection('matches').getFullList({ sort: 'kickoff' }),
			pb
				.collection('tips')
				.getFullList({ filter: `user = "${auth.user?.id}"` }),
			pb.collection('tournament_groups').getFullList({ sort: 'letter' }),
			serverClock.refresh(),
			pb
				.send('/api/tips/scores', { method: 'GET' })
				.catch(() => null)
		]);
		const gmap: Record<string, string[]> = {};
		for (const g of tgroups) gmap[g.letter] = g.teams ?? [];
		this.tournamentGroups = gmap;
		const tmap: Record<string, Team> = {};
		for (const t of teams)
			tmap[t.id] = {
				id: t.id,
				name: t.name,
				iso2: t.iso2,
				fifaCode: t.fifaCode
			};
		this.teams = tmap;
		const nextMatches = matches as unknown as Match[];
		this.matches = nextMatches;
		this.liveMatchIds = new Set(
			nextMatches.filter((m) => isLiveStatus(m.status)).map((m) => m.id)
		);
		const tip: Record<string, Tip> = {};
		for (const r of mine)
			tip[r.match] = {
				id: r.id,
				match: r.match,
				ftHome: r.ftHome,
				ftAway: r.ftAway,
				etHome: r.etHome,
				etAway: r.etAway,
				penWinner: r.penWinner,
				advancer: r.advancer
			};
		this.tips = tip;
		if (scoresResult) {
			const scores = scoresResult.scores ?? {};
			const scoresChanged = !sameNumberRecord(this.scores, scores);
			this.scores = scores;
			const finalizedChanged = nextMatches.some(
				(match) => previousFinalized.get(match.id) !== match.finalizedAt
			);
			if (wasLoaded && (scoresChanged || finalizedChanged)) {
				this.scoreRevision += 1;
			}
		}
		this.loaded = true;

		// Odds are non-critical — load in background, silently skip on failure.
		pb.send('/api/odds', { method: 'GET' })
			.then((r) => {
				this.oddsSource = r.source ?? 'none';
				const map: Record<string, MatchOdds> = {};
				for (const o of r.odds ?? []) map[o.matchId] = o;
				this.odds = map;
			})
			.catch(() => {});
	}

	team(id: string): Team | undefined {
		return this.teams[id];
	}

	/** Save (create or update) a tip; throws with the server message on a
	 *  rule/validation failure so the UI can show it. */
	async save(t: Tip): Promise<void> {
		const user = auth.user?.id;
		if (!user) throw new Error('You must be signed in to save tips.');
		const data = {
			user,
			match: t.match,
			ftHome: t.ftHome,
			ftAway: t.ftAway,
			etHome: t.etHome,
			etAway: t.etAway,
			penWinner: t.penWinner || ''
		};
		let rec;
		if (t.id) {
			rec = await pb.collection('tips').update(t.id, data);
		} else {
			try {
				rec = await pb.collection('tips').create(data);
			} catch (createError) {
				try {
					const existing = await pb
						.collection('tips')
						.getFirstListItem(`user = "${user}" && match = "${t.match}"`);
					rec = await pb.collection('tips').update(existing.id, data);
				} catch {
					throw createError;
				}
			}
		}
		this.tips[t.match] = {
			id: rec.id,
			match: rec.match,
			ftHome: rec.ftHome,
			ftAway: rec.ftAway,
			etHome: rec.etHome,
			etAway: rec.etAway,
			penWinner: rec.penWinner,
			advancer: rec.advancer
		};
	}

	async friends(matchId: string, leagueId = ''): Promise<FriendTip[]> {
		const qs = leagueId ? `?leagueId=${encodeURIComponent(leagueId)}` : '';
		const r = await pb.send(`/api/tips/others/${matchId}${qs}`, {
			method: 'GET'
		});
		return r.tips ?? [];
	}

	private scheduleScoreRefresh() {
		if (this.scoreRefreshTimer) return;
		this.scoreRefreshTimer = setTimeout(() => {
			this.scoreRefreshTimer = null;
			this.scoreRevision += 1;
			void this.refreshScores();
		}, 300);
	}

	private queueMatchUpdate(match: Match) {
		this.pendingMatchUpdates.set(match.id, match);
		if (this.pendingMatchTimer) {
			clearTimeout(this.pendingMatchTimer);
		}
		this.pendingMatchTimer = setTimeout(() => {
			this.pendingMatchTimer = null;
			void this.flushMatchUpdates();
		}, 250);
	}

	private async flushMatchUpdates() {
		if (this.pendingMatchUpdates.size === 0) return;
		const updates = [...this.pendingMatchUpdates.values()];
		this.pendingMatchUpdates.clear();
		if (serverClock.dev) {
			await serverClock.refresh();
		}
		const next = applyRealtimeMatchBatch(this.matches, this.liveMatchIds, updates);
		this.matches = next.matches;
		this.liveMatchIds = next.liveMatchIds;
		if (next.finalizedChanged) this.scheduleScoreRefresh();
	}

	async subscribe() {
		if (this._matchUnsub) return;
		this._matchUnsub = await pb.collection('matches').subscribe('*', (data) => {
			const match = data.record as unknown as Match;
			this.queueMatchUpdate(match);
		});

		try {
			this._tipUnsub = await pb.collection('tips').subscribe('*', (data) => {
				const record = data.record as Record<string, unknown>;
				const owner = stringField(record.user);
				if (owner && owner !== auth.user?.id) return;
				const tip = this.toTip(record);
				if (!tip) return;
				if (data.action === 'delete') {
					this.removeTip(tip);
				} else {
					this.upsertTip(tip);
				}
				this.scheduleScoreRefresh();
			});
		} catch {
			// Older deployed backends may not expose tips realtime.
		}

		// Seed liveMatchIds from current matches state.
		this.liveMatchIds = new Set(this.matches.filter((m) => isLiveStatus(m.status)).map((m) => m.id));

		// Subscribe to events BEFORE loading snapshot — any events that arrive
		// during the HTTP fetch are caught by the subscription and merged via
		// upsertLiveEvent, closing the snapshot/subscription gap.
		try {
			this._eventUnsub = await pb.collection('match_events').subscribe('*', (data) => {
				const record = data.record as Record<string, unknown>;
				const event = this.toLiveEvent(record);
				if (!event) return;
				if (data.action === 'delete') {
					this.removeLiveEvent(event);
					return;
				}
				this.upsertLiveEvent(event);
			});
		} catch {
			// Older deployed backends may not have match_events during rollout.
		}

		try {
			this._matchScoreUnsub = await pb.collection('match_scores').subscribe('*', () => {
				this.scheduleScoreRefresh();
			});
		} catch {
			// Older deployed backends may not expose score collections realtime.
		}

		try {
			this._forecastScoreUnsub = await pb.collection('forecast_scores').subscribe('*', () => {
				this.scheduleScoreRefresh();
			});
		} catch {
			// Older deployed backends may not expose score collections realtime.
		}

		await this.loadLiveEvents();
	}

	unsubscribe() {
		if (this._matchUnsub) {
			this._matchUnsub();
			this._matchUnsub = null;
		}
		if (this._tipUnsub) {
			this._tipUnsub();
			this._tipUnsub = null;
		}
		if (this._eventUnsub) {
			this._eventUnsub();
			this._eventUnsub = null;
		}
		if (this._matchScoreUnsub) {
			this._matchScoreUnsub();
			this._matchScoreUnsub = null;
		}
		if (this._forecastScoreUnsub) {
			this._forecastScoreUnsub();
			this._forecastScoreUnsub = null;
		}
		if (this.pendingMatchTimer) {
			clearTimeout(this.pendingMatchTimer);
			this.pendingMatchTimer = null;
		}
		this.pendingMatchUpdates.clear();
		if (this.scoreRefreshTimer) {
			clearTimeout(this.scoreRefreshTimer);
			this.scoreRefreshTimer = null;
		}
		this.liveEvents = {};
	}

	private async refreshScores() {
		try {
			const r = await pb.send('/api/tips/scores', { method: 'GET' });
			this.scores = r.scores ?? {};
		} catch {
			/* ignore */
		}
	}

	private async loadLiveEvents() {
		try {
			const r = await pb.send('/api/live/events', { method: 'GET' });
			// Merge snapshot into current state via upsert — preserves any realtime
			// events that arrived between subscription setup and this HTTP response.
			for (const [, rawEvents] of Object.entries(r.events ?? {})) {
				if (!Array.isArray(rawEvents)) continue;
				for (const raw of rawEvents) {
					const event = this.toLiveEvent(raw as Record<string, unknown>);
					if (event) this.upsertLiveEvent(event);
				}
			}
		} catch {
			/* ignore — realtime subscription still works */
		}
	}

	// Fetch the stored events for a single match on demand. Unlike loadLiveEvents
	// (which only covers currently-live matches), this works for finished matches
	// too — their match_events rows persist after full-time — so TipCard can show
	// a post-match goals/red-cards summary. Returns [] on any failure or for a
	// match that never had events captured.
	async loadMatchEvents(matchId: string): Promise<LiveEvent[]> {
		if (!matchId) return [];
		try {
			const r = await pb.send(`/api/matches/${encodeURIComponent(matchId)}/events`, {
				method: 'GET'
			});
			const raw = Array.isArray(r.events) ? r.events : [];
			const events: LiveEvent[] = [];
			for (const item of raw) {
				const event = this.toLiveEvent(item as Record<string, unknown>);
				if (event) events.push(event);
			}
			return events.sort(sortLiveEvents);
		} catch {
			return [];
		}
	}

	private toLiveEvent(record: Record<string, unknown>): LiveEvent | null {
		const match = stringField(record.match);
		if (!match) return null;
		return {
			id: stringField(record.id),
			match,
			providerKey: stringField(record.providerKey),
			created: stringField(record.created),
			elapsed: numberField(record.elapsed),
			extra: numberField(record.extra),
			type: stringField(record.type),
			detail: stringField(record.detail),
			player: stringField(record.player),
			assist: stringField(record.assist),
			team: stringField(record.team),
			teamId: stringField(record.teamId),
			comments: stringField(record.comments)
		};
	}

	private toTip(record: Record<string, unknown>): Tip | null {
		const match = stringField(record.match);
		if (!match) return null;
		return {
			id: stringField(record.id),
			match,
			ftHome: numberField(record.ftHome),
			ftAway: numberField(record.ftAway),
			etHome: numberField(record.etHome),
			etAway: numberField(record.etAway),
			penWinner: stringField(record.penWinner),
			advancer: stringField(record.advancer)
		};
	}

	private upsertTip(tip: Tip) {
		this.tips = {
			...this.tips,
			[tip.match]: tip
		};
	}

	private removeTip(tip: Tip) {
		const next = { ...this.tips };
		if (next[tip.match]?.id === tip.id || !tip.id) {
			delete next[tip.match];
		} else {
			for (const [matchId, existing] of Object.entries(next)) {
				if (existing.id === tip.id) {
					delete next[matchId];
					break;
				}
			}
		}
		this.tips = next;
	}

	private upsertLiveEvent(event: LiveEvent) {
		const current = this.liveEvents[event.match] ?? [];
		const idx = current.findIndex(
			(existing) =>
				(!!event.id && existing.id === event.id) ||
				(!!event.providerKey && existing.providerKey === event.providerKey)
		);
		const next = idx >= 0 ? current.map((item, i) => (i === idx ? event : item)) : [...current, event];
		this.liveEvents = {
			...this.liveEvents,
			[event.match]: next.sort(sortLiveEvents)
		};
	}

	private removeLiveEvent(event: LiveEvent) {
		const current = this.liveEvents[event.match] ?? [];
		const next = current.filter(
			(existing) =>
				!(!!event.id && existing.id === event.id) &&
				!(!!event.providerKey && existing.providerKey === event.providerKey)
		);
		this.liveEvents = {
			...this.liveEvents,
			[event.match]: next
		};
	}
}

export const tipsStore = new TipsStore();

function stringField(value: unknown): string {
	return typeof value === 'string' ? value : '';
}

function numberField(value: unknown): number {
	return typeof value === 'number' && Number.isFinite(value) ? value : 0;
}

function sameNumberRecord(a: Record<string, number>, b: Record<string, number>): boolean {
	const aKeys = Object.keys(a);
	const bKeys = Object.keys(b);
	if (aKeys.length !== bKeys.length) return false;
	for (const key of aKeys) {
		if (a[key] !== b[key]) return false;
	}
	return true;
}

function sortLiveEvents(a: LiveEvent, b: LiveEvent): number {
	return a.elapsed - b.elapsed || a.extra - b.extra || a.type.localeCompare(b.type) || a.player.localeCompare(b.player);
}

export function isLiveStatus(status: string): boolean {
	return ['live', '1H', '2H', 'HT', 'ET', 'BT', 'P', 'LIVE', 'INT'].includes(status);
}

export function isLocked(m: Match): boolean {
	return serverClock.now() >= new Date(m.kickoff).getTime();
}
export function teamsResolved(m: Match): boolean {
	return !!m.homeTeam && !!m.awayTeam;
}
