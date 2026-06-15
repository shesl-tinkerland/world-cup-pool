import { browser } from '$app/environment';
import { api, type LeagueSummary } from './api';
import { auth } from './auth.svelte';

const STORAGE_PREFIX = 'friend-tips-league-v1';
const GLOBAL_INVITE_CODE = 'GLOBAL';

class FriendTipsLeagueStore {
	leagues = $state<LeagueSummary[]>([]);
	selectedId = $state('');
	loaded = $state(false);
	busy = $state(false);
	error = $state(false);
	private userId = '';
	private loadPromise: Promise<void> | null = null;

	get selectedLeague() {
		return this.leagues.find((league) => league.id === this.selectedId) ?? null;
	}

	async load() {
		const currentUserId = auth.user?.id ?? '';
		if (!auth.isAuthed || !currentUserId) {
			this.reset();
			return;
		}
		if (this.loaded && this.userId === currentUserId) return;
		if (this.loadPromise && this.userId === currentUserId) return this.loadPromise;

		this.userId = currentUserId;
		this.busy = true;
		this.error = false;
		const promise = api
			.myLeagues()
			.then((response) => {
				if (this.userId !== currentUserId) return;
				this.leagues = response.leagues;
				this.selectedId = this.defaultLeagueId(response.leagues);
				this.loaded = true;
			})
			.catch(() => {
				if (this.userId !== currentUserId) return;
				this.leagues = [];
				this.selectedId = '';
				this.error = true;
				this.loaded = true;
			})
			.finally(() => {
				if (this.loadPromise !== promise) return;
				this.busy = false;
				this.loadPromise = null;
			});
		this.loadPromise = promise;
		return this.loadPromise;
	}

	select(leagueId: string) {
		if (!this.leagues.some((league) => league.id === leagueId)) return;
		this.selectedId = leagueId;
		this.remember(leagueId);
	}

	private reset() {
		this.leagues = [];
		this.selectedId = '';
		this.loaded = false;
		this.busy = false;
		this.error = false;
		this.userId = '';
		this.loadPromise = null;
	}

	private defaultLeagueId(leagues: LeagueSummary[]) {
		const stored = this.readStored();
		if (stored && leagues.some((league) => league.id === stored)) return stored;
		return (
			leagues.find((league) => league.inviteCode !== GLOBAL_INVITE_CODE)?.id ??
			leagues[0]?.id ??
			''
		);
	}

	private storageKey() {
		return this.userId ? `${STORAGE_PREFIX}:${this.userId}` : '';
	}

	private readStored() {
		if (!browser) return '';
		const key = this.storageKey();
		if (!key) return '';
		try {
			return localStorage.getItem(key) ?? '';
		} catch {
			return '';
		}
	}

	private remember(leagueId: string) {
		if (!browser) return;
		const key = this.storageKey();
		if (!key) return;
		try {
			localStorage.setItem(key, leagueId);
		} catch {
			/* localStorage can be unavailable in private mode */
		}
	}
}

export const friendTipsLeague = new FriendTipsLeagueStore();
