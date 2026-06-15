import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('./pb', () => {
	const send = vi.fn();
	return { pb: { send, __send: send } };
});

import { api } from './api';
import { pb } from './pb';

const sendMock = (pb as unknown as { send: ReturnType<typeof vi.fn> }).send;

describe('api', () => {
	beforeEach(() => {
		sendMock.mockReset();
	});

	it('myLeagues calls the right endpoint and returns rows', async () => {
		sendMock.mockResolvedValueOnce({ leagues: [{ id: 'a', name: 'X' }] });
		const r = await api.myLeagues();
		expect(sendMock).toHaveBeenCalledWith('/api/leagues/mine', { method: 'GET' });
		expect(r.leagues[0].name).toBe('X');
	});

	it('chatOverview calls the overview endpoint', async () => {
		sendMock.mockResolvedValueOnce({ items: [{ leagueId: 'L1', leagueName: 'Liga', unread: 2, message: null }] });
		const r = await api.chatOverview();
		expect(sendMock).toHaveBeenCalledWith('/api/chat/overview', { method: 'GET' });
		expect(r.items[0].unread).toBe(2);
	});

	it('createLeague posts the name', async () => {
		sendMock.mockResolvedValueOnce({ id: 'L1', name: 'Kompisar', inviteCode: 'ABCD' });
		const r = await api.createLeague('Kompisar');
		expect(sendMock).toHaveBeenCalledWith('/api/leagues/create', {
			method: 'POST',
			body: { name: 'Kompisar' }
		});
		expect(r.inviteCode).toBe('ABCD');
	});

	it('joinLeague posts the code', async () => {
		sendMock.mockResolvedValueOnce({ id: 'L2', name: 'NB' });
		await api.joinLeague('XYZ');
		expect(sendMock).toHaveBeenCalledWith('/api/leagues/join', {
			method: 'POST',
			body: { code: 'XYZ' }
		});
	});

	it('invitePreview URL-encodes the code', async () => {
		sendMock.mockResolvedValueOnce({ id: 'L3', name: 'Test' });
		await api.invitePreview('a b/c');
		expect(sendMock).toHaveBeenCalledWith('/api/invite/a%20b%2Fc', { method: 'GET' });
	});

	it('renameLeague posts the new name', async () => {
		sendMock.mockResolvedValueOnce({ id: 'L1', name: 'Ny liga' });
		await api.renameLeague('L1', 'Ny liga');
		expect(sendMock).toHaveBeenCalledWith('/api/leagues/L1/rename', {
			method: 'POST',
			body: { name: 'Ny liga' }
		});
	});

	it('regenerateLeagueCode posts to the regenerate endpoint', async () => {
		sendMock.mockResolvedValueOnce({ inviteCode: 'NEW123' });
		await api.regenerateLeagueCode('L1');
		expect(sendMock).toHaveBeenCalledWith('/api/leagues/L1/code/regenerate', {
			method: 'POST',
			body: {}
		});
	});

	it('setLeagueCodePrivacy posts the private flag', async () => {
		sendMock.mockResolvedValueOnce({ private: true });
		await api.setLeagueCodePrivacy('L1', true);
		expect(sendMock).toHaveBeenCalledWith('/api/leagues/L1/code/visibility', {
			method: 'POST',
			body: { private: true }
		});
	});

	it('removeLeagueMember posts the user id', async () => {
		sendMock.mockResolvedValueOnce({ ok: true });
		await api.removeLeagueMember('L1', 'u2');
		expect(sendMock).toHaveBeenCalledWith('/api/leagues/L1/members/remove', {
			method: 'POST',
			body: { userId: 'u2' }
		});
	});

	it('leaderboard returns rows with totals', async () => {
		sendMock.mockResolvedValueOnce({
			league: { id: 'L', name: 'X' },
			rows: [
				{ userId: 'u1', name: 'A', avatarUrl: '/api/files/users/u1/avatar.png', total: 42, tipsPoints: 30, forecastPoints: 12 },
				{ userId: 'u2', name: 'B', avatarUrl: null, total: 8, tipsPoints: 4, forecastPoints: 4 }
			]
		});
		const r = await api.leaderboard('L');
		expect(r.rows).toHaveLength(2);
		expect(r.rows[0].total).toBe(42);
		expect(r.rows[0].avatarUrl).toBe('/api/files/users/u1/avatar.png');
		expect(r.rows[1].avatarUrl).toBeNull();
	});
});
