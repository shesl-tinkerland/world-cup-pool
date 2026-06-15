import { describe, it, expect } from 'vitest';
import { decisiveEvents } from './liveEvents';
import type { LiveEvent } from './tips.svelte';

function ev(p: Partial<LiveEvent>): LiveEvent {
	return {
		id: Math.random().toString(36).slice(2),
		match: 'm1',
		providerKey: Math.random().toString(36).slice(2),
		created: '',
		elapsed: 0,
		extra: 0,
		type: 'Goal',
		detail: 'Normal Goal',
		player: '',
		assist: '',
		team: '',
		teamId: '',
		comments: '',
		...p
	};
}

describe('decisiveEvents', () => {
	it('collapses provider duplicates and drops missed penalties (USA 4:1 Paraguay)', () => {
		// The raw feed for the screenshot: 8 "Goal" rows for a 4:1 match — the 31'
		// and 45+5' goals are doubled by assist/name backfill, and 73' is a missed
		// penalty filed under type "Goal".
		const raw: LiveEvent[] = [
			ev({ elapsed: 7, team: 'Paraguay', player: 'Damian Bobadilla' }),
			ev({ elapsed: 28, team: 'USA', player: 'F. Balogun' }),
			ev({ elapsed: 31, team: 'USA', player: 'F. Balogun' }),
			ev({ elapsed: 31, team: 'USA', player: 'Folarin Balogun', assist: 'Weah' }),
			ev({ elapsed: 45, extra: 5, team: 'USA', player: 'F. Balogun' }),
			ev({ elapsed: 45, extra: 5, team: 'USA', player: 'F. Balogun', assist: 'Pulisic' }),
			ev({ elapsed: 50, team: 'USA', player: 'Folarin Balogun' }),
			ev({ elapsed: 73, team: 'USA', player: 'Maurício Magalhães Prado', detail: 'Missed Penalty' })
		];

		const out = decisiveEvents(raw);

		// 5 real goals: 4 USA + 1 Paraguay.
		expect(out.length).toBe(5);
		expect(out.filter((e) => e.team === 'USA').length).toBe(4);
		expect(out.filter((e) => e.team === 'Paraguay').length).toBe(1);
		// No missed penalty survives.
		expect(out.some((e) => e.detail === 'Missed Penalty')).toBe(false);
		// The 31' duplicate keeps the fuller, corrected name.
		const at31 = out.find((e) => e.elapsed === 31);
		expect(at31?.player).toBe('Folarin Balogun');
		// Ordered by minute.
		expect(out.map((e) => e.elapsed)).toEqual([7, 28, 31, 45, 50]);
	});

	it('drops a goal that VAR disallowed as a separate event (South Korea 2:1 Czechia)', () => {
		// Real data: Souček's 77' goal stays "Normal Goal" while the offside reversal
		// arrives as a separate VAR event at the same moment. The summary must show 3.
		const raw: LiveEvent[] = [
			ev({ elapsed: 59, team: 'Czech Republic', player: 'L. Krejci' }),
			ev({ elapsed: 64, type: 'subst', detail: 'Substitution 1', team: 'Czech Republic', player: 'P. Sulc' }),
			ev({ elapsed: 67, team: 'South Korea', player: 'Hwang In-Beom' }),
			ev({ elapsed: 77, team: 'Czech Republic', player: 'T. Soucek' }),
			ev({ elapsed: 77, type: 'Var', detail: 'Goal Disallowed - offside', team: 'Czech Republic', player: 'T. Soucek' }),
			ev({ elapsed: 80, team: 'South Korea', player: 'Oh Hyeon-Gyu' })
		];

		const out = decisiveEvents(raw);

		expect(out.length).toBe(3);
		expect(out.filter((e) => e.team === 'South Korea').length).toBe(2);
		expect(out.filter((e) => e.team === 'Czech Republic').length).toBe(1);
		expect(out.some((e) => e.player === 'T. Soucek')).toBe(false);
		expect(out.map((e) => e.elapsed)).toEqual([59, 67, 80]);
	});

	it('keeps red cards but drops yellow cards, subs and VAR', () => {
		const raw: LiveEvent[] = [
			ev({ elapsed: 20, type: 'Card', detail: 'Yellow Card', team: 'USA', player: 'A' }),
			ev({ elapsed: 60, type: 'Card', detail: 'Red Card', team: 'USA', player: 'B' }),
			ev({ elapsed: 65, type: 'subst', detail: 'Substitution', team: 'USA', player: 'C' }),
			ev({ elapsed: 70, type: 'Var', detail: 'Goal Disallowed', team: 'USA', player: 'D' })
		];

		const out = decisiveEvents(raw);

		expect(out.length).toBe(1);
		expect(out[0].detail).toBe('Red Card');
	});
});
