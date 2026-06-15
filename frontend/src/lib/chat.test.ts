import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockState = vi.hoisted(() => ({
	send: vi.fn(),
	collection: vi.fn(),
	unsubscribe: vi.fn(),
	subscriptions: {} as Record<string, (event: unknown) => void>
}));

vi.mock('./pb', () => ({
	pb: {
		send: mockState.send,
		collection: mockState.collection.mockImplementation((name: string) => ({
			subscribe: vi.fn(async (topic: string, callback: (event: unknown) => void) => {
				mockState.subscriptions[`${name}:${topic}`] = callback;
				return mockState.unsubscribe;
			})
		}))
	}
}));

import { leagueChat, type ChatMessage } from './chat.svelte';

const message = (over: Partial<ChatMessage> = {}): ChatMessage => ({
	id: 'm1',
	leagueId: 'l1',
	userId: 'u1',
	user: { id: 'u1', name: 'Ada', avatarUrl: null },
	text: 'Hei liga',
	created: '2026-05-22T10:00:00Z',
	updated: '2026-05-22T10:00:00Z',
	deleted: false,
	reactions: [],
	...over
});

describe('leagueChat', () => {
	beforeEach(() => {
		vi.useRealTimers();
		leagueChat.disconnect();
		mockState.send.mockReset();
		mockState.collection.mockClear();
		mockState.unsubscribe.mockReset();
		mockState.subscriptions = {};
	});

	it('loads messages and subscribes to realtime collections', async () => {
		mockState.send.mockResolvedValueOnce({ messages: [message()] });

		await leagueChat.load('l1');

		expect(mockState.send).toHaveBeenCalledWith('/api/leagues/l1/chat', { method: 'GET' });
		expect(leagueChat.messages).toHaveLength(1);
		expect(mockState.subscriptions['league_messages:*']).toBeTypeOf('function');
		expect(mockState.subscriptions['league_message_reactions:*']).toBeTypeOf('function');
		expect(leagueChat.connected).toBe(true);
	});

	it('sends trimmed text and adds the returned message', async () => {
		mockState.send
			.mockResolvedValueOnce({ messages: [] })
			.mockResolvedValueOnce({ message: message({ id: 'm2', text: 'Ny melding' }) });

		await leagueChat.load('l1');
		await leagueChat.send('  Ny melding  ');

		expect(mockState.send).toHaveBeenLastCalledWith('/api/leagues/l1/chat/messages', {
			method: 'POST',
			body: { text: 'Ny melding' }
		});
		expect(leagueChat.messages[0].id).toBe('m2');
	});

	it('soft-deletes and restores messages through the custom endpoints', async () => {
		mockState.send
			.mockResolvedValueOnce({ messages: [message()] })
			.mockResolvedValueOnce({ message: message({ deleted: true, text: '' }) })
			.mockResolvedValueOnce({ message: message({ deleted: false, text: 'Hei liga' }) });

		await leagueChat.load('l1');
		const deleted = await leagueChat.delete('m1');
		const restored = await leagueChat.restore('m1');

		expect(mockState.send).toHaveBeenNthCalledWith(2, '/api/leagues/l1/chat/messages/m1', {
			method: 'DELETE'
		});
		expect(mockState.send).toHaveBeenNthCalledWith(3, '/api/leagues/l1/chat/messages/m1/restore', {
			method: 'POST'
		});
		expect(deleted?.deleted).toBe(true);
		expect(restored?.deleted).toBe(false);
		expect(leagueChat.messages[0].text).toBe('Hei liga');
	});

	it('groups reaction summaries and marks my reaction', () => {
		const summaries = leagueChat.reactionSummary(
			message({
				reactions: [
					{ id: 'r1', emoji: '👍', userId: 'u1', user: { id: 'u1', name: 'Ada', avatarUrl: null } },
					{ id: 'r2', emoji: '👍', userId: 'u2', user: { id: 'u2', name: 'Bo', avatarUrl: null } },
					{ id: 'r3', emoji: '🔥', userId: 'u3', user: { id: 'u3', name: 'Cy', avatarUrl: null } }
				]
			}),
			'u1'
		);

		expect(summaries[0]).toMatchObject({ emoji: '👍', count: 2, mine: true });
		expect(summaries[1]).toMatchObject({ emoji: '🔥', count: 1, mine: false });
	});

	it('refreshes when a realtime message event belongs to the active league', async () => {
		vi.useFakeTimers();
		mockState.send
			.mockResolvedValueOnce({ messages: [message()] })
			.mockResolvedValueOnce({ messages: [message(), message({ id: 'm2', text: 'Live' })] });

		await leagueChat.load('l1');
		mockState.subscriptions['league_messages:*']({ record: { league: 'l1' } });
		await vi.advanceTimersByTimeAsync(200);

		expect(mockState.send).toHaveBeenCalledTimes(2);
		expect(leagueChat.messages).toHaveLength(2);
	});
});
