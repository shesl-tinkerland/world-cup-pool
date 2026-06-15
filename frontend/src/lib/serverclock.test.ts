import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockState = vi.hoisted(() => ({
	send: vi.fn()
}));

vi.mock('./pb', () => ({
	pb: {
		send: mockState.send
	}
}));

import { serverClock } from './serverclock.svelte';

describe('serverClock', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-06-09T10:00:00Z'));
		mockState.send.mockReset();
		serverClock.stopAutoRefresh();
		serverClock.offset = 0;
		serverClock.dev = false;
		serverClock.simulated = false;
		serverClock.simTime = null;
		serverClock.loaded = false;
	});

	it('keeps simulated time fixed after refresh', async () => {
		mockState.send.mockResolvedValueOnce({
			now: Date.parse('2026-06-11T20:00:00Z'),
			dev: true,
			simulated: true,
			simTime: '2026-06-11T20:00:00Z'
		});

		await serverClock.refresh();

		expect(serverClock.now()).toBe(Date.parse('2026-06-11T20:00:00Z'));
		vi.advanceTimersByTime(90_000);
		expect(serverClock.now()).toBe(Date.parse('2026-06-11T20:00:00Z'));
	});

	it('continues advancing when using real time offset', async () => {
		const base = Date.parse('2026-06-09T10:00:00Z');
		mockState.send.mockResolvedValueOnce({
			now: base + 5_000,
			dev: true,
			simulated: false,
			simTime: null
		});

		await serverClock.refresh();

		expect(serverClock.now()).toBe(base + 5_000);
		vi.advanceTimersByTime(60_000);
		expect(serverClock.now()).toBe(base + 65_000);
	});
});