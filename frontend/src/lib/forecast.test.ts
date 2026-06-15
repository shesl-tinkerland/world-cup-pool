// Tests the pure standings/derivation logic of ForecastStore against
// hand-crafted match results. No network — the pb module is mocked.

import { describe, it, expect, vi } from 'vitest';

vi.mock('./pb', () => ({ pb: { send: vi.fn(), collection: () => ({}) } }));
vi.mock('./auth.svelte', () => ({ auth: { user: { id: 'u1' } } }));

import { ForecastStore, koKey } from './forecast.svelte';

function seed(store: ForecastStore) {
	// 4 teams in group A.
	store.groups = [{ letter: 'A', teams: ['t1', 't2', 't3', 't4'] }];
	// Three round-robin matches, all finished.
	store.results = [
		mk('A', 't1', 't2', 2, 0),
		mk('A', 't3', 't4', 1, 1),
		mk('A', 't1', 't3', 3, 0),
		mk('A', 't2', 't4', 0, 2),
		mk('A', 't1', 't4', 1, 0),
		mk('A', 't2', 't3', 2, 2)
	];
}

function mk(letter: string, home: string, away: string, h: number, a: number) {
	return {
		stage: 'group',
		groupLetter: letter,
		num: 0,
		homeTeam: home,
		awayTeam: away,
		ftHome: h,
		ftAway: a,
		advancer: '',
		finished: true
	};
}

describe('ForecastStore.actualOrder', () => {
	it('ranks the group by points then GD then GF', () => {
		const store = new ForecastStore();
		seed(store);
		const order = store.actualOrder('A');
		// t1: 3W = 9pts, gd +6. t4: 1W 1D 1L = 4pts, gd +1.
		// t3: 2D 1L = 2pts, gd -3. t2: 1D 2L = 1pt, gd -4.
		expect(order).toEqual(['t1', 't4', 't3', 't2']);
	});

	it('returns null while the group is incomplete', () => {
		const store = new ForecastStore();
		store.groups = [{ letter: 'A', teams: ['t1', 't2', 't3', 't4'] }];
		store.results = [mk('A', 't1', 't2', 1, 0)];
		expect(store.actualOrder('A')).toBeNull();
	});
});

describe('ForecastStore.groupStageDone', () => {
	it('is false until every group match is finished', () => {
		const store = new ForecastStore();
		store.results = [{ ...mk('A', 't1', 't2', 1, 0), finished: false }];
		expect(store.groupStageDone).toBe(false);
		store.results = [mk('A', 't1', 't2', 1, 0)];
		expect(store.groupStageDone).toBe(true);
	});
});

describe('koKey', () => {
	it('returns the match number when present, else the stage label', () => {
		expect(koKey({ num: 49, stage: 'R16' })).toBe('49');
		expect(koKey({ num: 0, stage: 'FINAL' })).toBe('FINAL');
	});
});

describe('ForecastStore.move + toggleThird', () => {
	it('move swaps adjacent teams in a group', () => {
		const store = new ForecastStore();
		store.groupOrder = { A: ['t1', 't2', 't3', 't4'] };
		store.move('A', 0, 1);
		expect(store.groupOrder['A']).toEqual(['t2', 't1', 't3', 't4']);
	});

	it('move clamps at the boundaries', () => {
		const store = new ForecastStore();
		store.groupOrder = { A: ['t1', 't2'] };
		store.move('A', 0, -1);
		expect(store.groupOrder['A']).toEqual(['t1', 't2']);
	});

	it('toggleThird respects the 8-team cap', () => {
		const store = new ForecastStore();
		store.groupOrder = {
			A: ['', '', 'a3', ''],
			B: ['', '', 'b3', ''],
			C: ['', '', 'c3', ''],
			D: ['', '', 'd3', ''],
			E: ['', '', 'e3', ''],
			F: ['', '', 'f3', ''],
			G: ['', '', 'g3', ''],
			H: ['', '', 'h3', ''],
			I: ['', '', 'i3', '']
		};
		for (const l of ['A', 'B', 'C', 'D', 'E', 'F', 'G', 'H'])
			store.toggleThird(l);
		expect(store.chosenThirdLetters.length).toBe(8);
		store.toggleThird('I');
		expect(store.chosenThirdLetters.length).toBe(8);
		expect(store.chosenThirdLetters).not.toContain('I');
	});
});
