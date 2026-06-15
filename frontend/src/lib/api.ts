import { pb } from './pb';

// Calls our custom Go endpoints. pb.send attaches the auth token and resolves
// relative to the SDK base URL (same origin).
async function post<T>(path: string, body: unknown): Promise<T> {
	return pb.send(path, { method: 'POST', body });
}
async function get<T>(path: string): Promise<T> {
	return pb.send(path, { method: 'GET' });
}
async function put<T>(path: string, body: unknown): Promise<T> {
	return pb.send(path, { method: 'PUT', body });
}

export interface LeagueSummary {
	id: string;
	name: string;
	inviteCode: string;
	role: string;
	private?: boolean;
	members: number;
}

export interface LeagueInviteUser {
	id: string;
	name: string;
	email?: string;
	avatarUrl: string | null;
}

export interface LeagueInvite {
	id: string;
	leagueId: string;
	leagueName: string;
	invitedUser: LeagueInviteUser;
	invitedBy: LeagueInviteUser;
	status: 'pending' | 'accepted' | 'declined';
	created: string;
	updated: string;
	actedAt?: string;
}

export interface LeaderboardRow {
	userId: string;
	name: string;
	avatarUrl: string | null;
	total: number;
	tipsPoints: number;
	forecastPoints: number;
	predicted: number;
	exactScores: number;
	correctWinners: number;
	gdDeviation: number;
	forecast?: Record<string, number>;
	rankDelta: number; // +N = moved up N spots since last matchday, 0 = no change or no data
}

export interface GoldenBootPlayer {
	id: string;
	name: string;
	teamId: string;
	teamName: string;
	photoUrl?: string;
	goals: number;
	assists: number;
	rank: number;
	eligible: boolean;
	seeded: boolean;
	syncedAt?: string;
}

export interface GoldenBootPickUser {
	id: string;
	name: string;
	avatarUrl: string | null;
}

export interface GoldenBootLeaguePlayer extends GoldenBootPlayer {
	picks: GoldenBootPickUser[];
}

export interface GoldenBootLeagueTable {
	players: GoldenBootLeaguePlayer[];
	updatedAt?: string;
}

export interface GoldenBootSearchResult {
	key: string;
	id?: string;
	providerId: number;
	name: string;
	teamId: string;
	teamName: string;
	photoUrl?: string;
	goals: number;
	assists: number;
	rank: number;
	eligible: boolean;
	existing: boolean;
}

export interface ChatOverviewUser {
	id: string;
	name: string;
	avatarUrl: string | null;
}

export interface ChatOverviewMessage {
	id: string;
	leagueId: string;
	userId: string;
	user: ChatOverviewUser;
	text: string;
	created: string;
	updated: string;
	editedAt?: string;
	deleted: boolean;
	deletedBy?: string;
	deletedAt?: string;
	origText?: string;
}

export interface ChatOverviewItem {
	leagueId: string;
	leagueName: string;
	message: ChatOverviewMessage | null;
	unread: number;
	lastReadAt?: string;
}

export interface LeagueProgressEvent {
	matchId: string;
	kickoff: string;
	stage: string;
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
	points: number;
	totalAfter: number;
	tipped: boolean;
	exact: boolean;
	correctWinner: boolean;
	correctTotalGoals: boolean;
	correctGoalDiff: boolean;
}

export interface LeagueProgress {
	league: { id: string; name: string };
	summary: {
		tipsPoints: number;
		last5Points: number;
		finishedMatches: number;
		tippedFinished: number;
		exactScores: number;
		matchesWithPoints: number;
		bestPoints: number;
	};
	events: LeagueProgressEvent[];
}

export interface PlayerStatsHitRate {
	count: number;
	total: number;
	pct: number;
}

export interface PlayerStatsLargestMiss {
	matchId: string;
	kickoff: string;
	stage: string;
	homeTeam: string;
	awayTeam: string;
	homeLabel: string;
	awayLabel: string;
	tipHome: number;
	tipAway: number;
	actualHome: number;
	actualAway: number;
	gdDev: number;
}

export interface PlayerStats {
	tipsPredicted: number;
	tipsScored: number;
	hitRate: PlayerStatsHitRate;
	longestStreak: number;
	currentStreak: number;
	largestMiss?: PlayerStatsLargestMiss;
}

export interface CrowdOutcome {
	count: number;
	pct: number;
}

export interface CrowdDistribution {
	locked: boolean;
	total?: number;
	isKO?: boolean;
	outcomes?: {
		home: CrowdOutcome;
		draw: CrowdOutcome;
		away: CrowdOutcome;
	};
}

export interface DevTopscorer {
	id: string;
	name: string;
	goals: number;
}

// Notification preferences. Maps an event key to per-channel opt-in flags,
// e.g. { pre_kickoff_reminder: { email: true } }. Everything is opt-in.
export type NotifyChannel = 'email' | 'push';
export type NotifyPrefs = Record<string, Partial<Record<NotifyChannel, boolean>>>;
export interface NotifyEvent {
	key: string;
	channels: NotifyChannel[];
}
export interface NotifyPrefsResponse {
	events: NotifyEvent[];
	prefs: NotifyPrefs;
	promptSeen: boolean;
	pushSubscribed: boolean;
}

export interface PushSubscriptionBody {
	endpoint: string;
	keys: { p256dh: string; auth: string };
}

export const api = {
	notifyPrefs: () => get<NotifyPrefsResponse>('/api/account/notify-prefs'),
	updateNotifyPrefs: (prefs: NotifyPrefs) =>
		put<{ prefs: NotifyPrefs }>('/api/account/notify-prefs', prefs),
	markNotifyPromptSeen: () =>
		post<{ ok: boolean }>('/api/account/notify-prompt-seen', {}),
	// Web Push
	pushVapidKey: () => get<{ publicKey: string }>('/api/push/vapid-public-key'),
	pushSubscribe: (sub: PushSubscriptionBody) =>
		post<{ ok: boolean }>('/api/push/subscribe', sub),
	pushUnsubscribe: (endpoint: string) =>
		post<{ ok: boolean }>('/api/push/unsubscribe', { endpoint }),
	// Sends a real push to the caller's registered devices (self-test).
	pushTest: () => post<{ sent: boolean; devices: number }>('/api/push/test', {}),
	// Admin-only: render (preview) or send a test of a notification email.
	notifyPreview: (event: string, lang?: string) =>
		post<{ subject: string; html: string }>('/api/notifications/preview', { event, lang }),
	notifyTest: (event: string, to?: string, lang?: string) =>
		post<{ sent: boolean; to: string }>('/api/notifications/test', { event, to, lang }),
	createLeague: (name: string) =>
		post<{ id: string; name: string; inviteCode: string }>(
			'/api/leagues/create',
			{ name }
		),
	joinLeague: (code: string) =>
		post<{ id: string; name: string; already?: boolean }>(
			'/api/leagues/join',
			{ code }
		),
	deleteLeague: (id: string) => pb.send(`/api/leagues/${id}`, { method: 'DELETE' }),
	renameLeague: (id: string, name: string) =>
		post<{ id: string; name: string }>(`/api/leagues/${id}/rename`, { name }),
	regenerateLeagueCode: (id: string) =>
		post<{ inviteCode: string }>(`/api/leagues/${id}/code/regenerate`, {}),
	setLeagueCodePrivacy: (id: string, isPrivate: boolean) =>
		post<{ private: boolean }>(`/api/leagues/${id}/code/visibility`, {
			private: isPrivate
		}),
	removeLeagueMember: (id: string, userId: string) =>
		post<{ ok: boolean }>(`/api/leagues/${id}/members/remove`, { userId }),
	// Public — resolves an invite code to a league name for the /join page.
	invitePreview: (code: string) =>
		get<{ id: string; name: string }>(
			`/api/invite/${encodeURIComponent(code)}`
		),
	myLeagues: () => get<{ leagues: LeagueSummary[] }>('/api/leagues/mine'),
	inviteCandidates: (leagueId: string, query: string) =>
		get<{ users: LeagueInviteUser[] }>(
			`/api/leagues/${leagueId}/invite-candidates?q=${encodeURIComponent(query)}`
		),
	leagueInvites: (leagueId: string) =>
		get<{ invites: LeagueInvite[] }>(`/api/leagues/${leagueId}/invites`),
	createLeagueInvite: (leagueId: string, userId: string) =>
		post<{ invite: LeagueInvite }>(`/api/leagues/${leagueId}/invites`, { userId }),
	myLeagueInvitations: () =>
		get<{ invites: LeagueInvite[] }>('/api/leagues/invitations'),
	acceptLeagueInvitation: (inviteId: string) =>
		post<{ league: { id: string; name: string } }>(
			`/api/leagues/invitations/${inviteId}/accept`,
			{}
		),
	declineLeagueInvitation: (inviteId: string) =>
		post<void>(`/api/leagues/invitations/${inviteId}/decline`, {}),
	chatOverview: () => get<{ items: ChatOverviewItem[] }>('/api/chat/overview'),
	leaderboard: (id: string) =>
		get<{
			league: { id: string; name: string };
			rows: LeaderboardRow[];
			scoring?: Record<string, unknown>;
			goldenBoot?: GoldenBootLeagueTable;
		}>(`/api/leagues/${id}/leaderboard`),
	searchGoldenBootPlayers: (query: string) =>
		get<{ players: GoldenBootSearchResult[]; apiAvailable: boolean }>(
			`/api/forecast/topscorers/search?q=${encodeURIComponent(query)}`
		),
	ensureGoldenBootPlayer: (player: GoldenBootSearchResult) =>
		post<{ player: GoldenBootPlayer }>('/api/forecast/topscorers/ensure', player),
	leagueProgress: (id: string) =>
		get<LeagueProgress>(`/api/leagues/${id}/progress`),
	playerStats: () => get<PlayerStats>('/api/player/me/stats'),
	matchCrowd: (matchId: string) =>
		get<CrowdDistribution>(`/api/tips/crowd/${matchId}`),
	devTopscorers: () =>
		get<{ players: DevTopscorer[] }>('/api/dev/topscorers'),
	devSetTopscorers: (players: Record<string, number>) =>
		post<{ status: string }>('/api/dev/topscorers', { players }),
};
