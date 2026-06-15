import { pb } from './pb';
import { runtimeText } from './runtimeLanguage';

export const CHAT_EMOJIS = ['👍', '🔥', '😂', '❤️', '👏', '😮', '😢', '⚽'];

export interface ChatUser {
	id: string;
	name: string;
	avatarUrl: string | null;
}

export interface ChatReaction {
	id: string;
	emoji: string;
	userId: string;
	user: ChatUser;
}

export interface ChatMessage {
	id: string;
	leagueId: string;
	userId: string;
	user: ChatUser;
	text: string;
	created: string;
	updated: string;
	editedAt?: string;
	deleted: boolean;
	deletedBy?: string;
	deletedAt?: string;
	origText?: string;
	reactions: ChatReaction[];
}

export interface ReactionSummary {
	emoji: string;
	count: number;
	mine: boolean;
	users: string[];
}

type Unsubscribe = () => void;

type RecordEvent = {
	action: string;
	record: Record<string, unknown>;
};

function chatError(nb: string, nn: string, en: string) {
	return runtimeText(nb, nn, en);
}

class LeagueChatStore {
	messages = $state<ChatMessage[]>([]);
	loaded = $state(false);
	loading = $state(false);
	sending = $state(false);
	connected = $state(false);
	error = $state('');

	private leagueId = '';
	private messageUnsubscribe: Unsubscribe | null = null;
	private reactionUnsubscribe: Unsubscribe | null = null;
	private refreshTimer: ReturnType<typeof setTimeout> | null = null;

	async load(leagueId: string) {
		if (!leagueId) return;
		if (this.leagueId !== leagueId) {
			this.disconnect();
			this.leagueId = leagueId;
			this.messages = [];
			this.loaded = false;
		}

		this.loading = true;
		this.error = '';
		try {
			const data = await pb.send<{ messages: ChatMessage[] }>(
				`/api/leagues/${encodeURIComponent(leagueId)}/chat`,
				{ method: 'GET' }
			);
			this.messages = data.messages;
			this.loaded = true;
			await this.connect(leagueId);
		} catch {
			this.error = chatError('Kunne ikke laste chatten.', 'Kunne ikkje laste chatten.', 'Could not load the chat.');
		} finally {
			this.loading = false;
		}
	}

	async send(text: string) {
		const trimmed = text.trim();
		if (!this.leagueId || !trimmed) return;
		this.sending = true;
		this.error = '';
		try {
			const data = await pb.send<{ message: ChatMessage }>(
				`/api/leagues/${encodeURIComponent(this.leagueId)}/chat/messages`,
				{ method: 'POST', body: { text: trimmed } }
			);
			this.upsertMessage(data.message);
		} catch {
			this.error = chatError('Kunne ikke sende meldingen.', 'Kunne ikkje sende meldinga.', 'Could not send the message.');
			throw new Error(this.error);
		} finally {
			this.sending = false;
		}
	}

	async edit(messageId: string, text: string) {
		const trimmed = text.trim();
		if (!this.leagueId || !messageId || !trimmed) return;
		this.error = '';
		try {
			const data = await pb.send<{ message: ChatMessage }>(
				`/api/leagues/${encodeURIComponent(this.leagueId)}/chat/messages/${encodeURIComponent(messageId)}`,
				{ method: 'PATCH', body: { text: trimmed } }
			);
			this.upsertMessage(data.message);
		} catch {
			this.error = chatError('Kunne ikke lagre endringen.', 'Kunne ikkje lagre endringa.', 'Could not save the edit.');
			throw new Error(this.error);
		}
	}

	async delete(messageId: string): Promise<ChatMessage | null> {
		if (!this.leagueId || !messageId) return null;
		this.error = '';
		try {
			const data = await pb.send<{ message: ChatMessage }>(
				`/api/leagues/${encodeURIComponent(this.leagueId)}/chat/messages/${encodeURIComponent(messageId)}`,
				{ method: 'DELETE' }
			);
			this.upsertMessage(data.message);
			return data.message;
		} catch {
			this.error = chatError('Kunne ikke slette meldingen.', 'Kunne ikkje slette meldinga.', 'Could not delete the message.');
			throw new Error(this.error);
		}
	}

	async restore(messageId: string): Promise<ChatMessage | null> {
		if (!this.leagueId || !messageId) return null;
		this.error = '';
		try {
			const data = await pb.send<{ message: ChatMessage }>(
				`/api/leagues/${encodeURIComponent(this.leagueId)}/chat/messages/${encodeURIComponent(messageId)}/restore`,
				{ method: 'POST' }
			);
			this.upsertMessage(data.message);
			return data.message;
		} catch {
			this.error = chatError('Kunne ikke angre slettingen.', 'Kunne ikkje angre slettinga.', 'Could not undo the delete.');
			throw new Error(this.error);
		}
	}

	async toggleReaction(messageId: string, emoji: string) {
		if (!this.leagueId || !messageId || !emoji.trim()) return;
		this.error = '';
		try {
			await pb.send(
				`/api/leagues/${encodeURIComponent(this.leagueId)}/chat/messages/${encodeURIComponent(messageId)}/reactions`,
				{ method: 'POST', body: { emoji } }
			);
			this.queueRefresh();
		} catch {
			this.error = chatError('Kunne ikke oppdatere reaksjonen.', 'Kunne ikkje oppdatere reaksjonen.', 'Could not update the reaction.');
			throw new Error(this.error);
		}
	}

	reactionSummary(message: ChatMessage, currentUserId?: string): ReactionSummary[] {
		const grouped = new Map<string, ReactionSummary>();
		for (const reaction of message.reactions) {
			const row = grouped.get(reaction.emoji) ?? {
				emoji: reaction.emoji,
				count: 0,
				mine: false,
				users: []
			};
			row.count += 1;
			row.mine = row.mine || reaction.userId === currentUserId;
			row.users.push(reaction.user.name);
			grouped.set(reaction.emoji, row);
		}
		return [...grouped.values()].sort((a, b) => b.count - a.count || a.emoji.localeCompare(b.emoji));
	}

	disconnect() {
		this.messageUnsubscribe?.();
		this.reactionUnsubscribe?.();
		this.messageUnsubscribe = null;
		this.reactionUnsubscribe = null;
		this.connected = false;
		if (this.refreshTimer) {
			clearTimeout(this.refreshTimer);
			this.refreshTimer = null;
		}
	}

	private async connect(leagueId: string) {
		if (this.messageUnsubscribe && this.reactionUnsubscribe) return;
		this.disconnect();
		try {
			this.messageUnsubscribe = await pb.collection('league_messages').subscribe('*', (event: RecordEvent) => {
				if (event.record.league === leagueId) {
					this.queueRefresh();
				}
			});
			this.reactionUnsubscribe = await pb.collection('league_message_reactions').subscribe('*', () => {
				this.queueRefresh();
			});
			this.connected = true;
		} catch {
			this.connected = false;
		}
	}

	private queueRefresh() {
		if (!this.leagueId) return;
		if (this.refreshTimer) clearTimeout(this.refreshTimer);
		this.refreshTimer = setTimeout(() => {
			void this.load(this.leagueId);
		}, 180);
	}

	private upsertMessage(message: ChatMessage) {
		const next = this.messages.filter((m) => m.id !== message.id);
		next.push(message);
		next.sort((a, b) => new Date(a.created).getTime() - new Date(b.created).getTime());
		this.messages = next;
	}
}

export const leagueChat = new LeagueChatStore();
