import { untrack } from 'svelte';
import { api, type ChatOverviewItem } from '$lib/api';
import { leagueInvitations } from '$lib/leagueInvitations.svelte';
import { pb } from '$lib/pb';

type RecordEvent = {
	action: string;
	record: Record<string, unknown>;
};

type Unsubscribe = () => void;

class LeagueBadges {
	chatItems = $state<ChatOverviewItem[]>([]);
	chatUnreadCount = $state(0);
	loading = $state(false);
	loaded = $state(false);
	realtimeConnected = $state(false);
	// League whose chat is on screen right now. Its unread is pinned to 0 so
	// badges and toasts never fire for the conversation the user is already
	// reading — the server-side read mark races the overview fetch.
	activeChatLeagueId = $state('');

	private timer: ReturnType<typeof setInterval> | null = null;
	private refreshTimer: ReturnType<typeof setTimeout> | null = null;
	private messageUnsubscribe: Unsubscribe | null = null;
	private consumers = 0;

	private readonly onVisibilityChange = () => {
		if (document.visibilityState !== 'visible') return;
		this.refreshNow();
	};
	private readonly onOnline = () => this.refreshNow();

	get totalCount() {
		return this.chatUnreadCount + leagueInvitations.pendingCount;
	}

	get firstUnreadChat() {
		return this.chatItems.find((item) => item.unread > 0) ?? null;
	}

	get activityHref() {
		const firstUnread = this.firstUnreadChat;
		return firstUnread ? `/leagues/${firstUnread.leagueId}#chat` : '/leagues';
	}

	unreadForLeague(leagueId: string) {
		return this.chatItems.find((item) => item.leagueId === leagueId)?.unread ?? 0;
	}

	markLeagueChatRead(leagueId: string) {
		if (!leagueId) return;
		// untrack: callers run inside component effects. Reading chatItems here
		// must not register a dependency — the rewrite below replaces the array,
		// which would re-trigger the calling effect forever (depth exceeded).
		untrack(() => {
			this.applyItems(
				this.chatItems.map((item) => (item.leagueId === leagueId ? { ...item, unread: 0 } : item))
			);
		});
	}

	// The league chat card registers itself while mounted so the store knows
	// which conversation is being read.
	setActiveChatLeague(leagueId: string) {
		if (!leagueId) return;
		this.activeChatLeagueId = leagueId;
		this.markLeagueChatRead(leagueId);
	}

	clearActiveChatLeague(leagueId: string) {
		if (this.activeChatLeagueId === leagueId) this.activeChatLeagueId = '';
	}

	start() {
		this.consumers += 1;
		if (this.consumers > 1) return;
		void this.load(true);
		void this.connect();
		// Poll as a realtime fallback; the tick also retries a failed subscribe.
		this.timer = setInterval(() => this.refreshNow(), 45_000);
		document.addEventListener('visibilitychange', this.onVisibilityChange);
		window.addEventListener('online', this.onOnline);
	}

	stop() {
		this.consumers = Math.max(0, this.consumers - 1);
		if (this.consumers > 0) return;
		if (this.timer) clearInterval(this.timer);
		if (this.refreshTimer) clearTimeout(this.refreshTimer);
		this.timer = null;
		this.refreshTimer = null;
		document.removeEventListener('visibilitychange', this.onVisibilityChange);
		window.removeEventListener('online', this.onOnline);
		this.disconnect();
	}

	clear() {
		this.stop();
		this.consumers = 0;
		this.chatItems = [];
		this.chatUnreadCount = 0;
		this.loading = false;
		this.loaded = false;
		this.realtimeConnected = false;
		this.activeChatLeagueId = '';
		leagueInvitations.clear();
	}

	private refreshNow() {
		if (!this.realtimeConnected) void this.connect();
		void this.load(true);
	}

	private async connect() {
		if (this.messageUnsubscribe) return;
		try {
			this.messageUnsubscribe = await pb.collection('league_messages').subscribe('*', (event: RecordEvent) => {
				if (event.action === 'create' || event.action === 'update' || event.action === 'delete') {
					this.queueLoad();
				}
			});
			this.realtimeConnected = true;
		} catch {
			this.realtimeConnected = false;
		}
	}

	private disconnect() {
		this.messageUnsubscribe?.();
		this.messageUnsubscribe = null;
		this.realtimeConnected = false;
	}

	private queueLoad() {
		if (this.refreshTimer) clearTimeout(this.refreshTimer);
		this.refreshTimer = setTimeout(() => {
			this.refreshTimer = null;
			void this.load(true);
		}, 220);
	}

	private applyItems(items: ChatOverviewItem[]) {
		const active = this.activeChatLeagueId;
		this.chatItems = active
			? items.map((item) => (item.leagueId === active ? { ...item, unread: 0 } : item))
			: items;
		this.chatUnreadCount = this.chatItems.reduce(
			(sum, item) => sum + Math.max(0, item.unread),
			0
		);
	}

	async load(force = false) {
		void leagueInvitations.load(force);
		if (this.loading || (this.loaded && !force)) return;
		this.loading = true;
		try {
			const overview = await api.chatOverview();
			this.applyItems(overview.items);
			this.loaded = true;
		} catch {
			// Keep the last good counts. Zeroing on a failed poll would make the
			// next successful one look like a burst of new activity and fire
			// toasts for old messages after a network blip.
		} finally {
			this.loading = false;
		}
	}
}

export const leagueBadges = new LeagueBadges();
