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

	it('keeps only one red card per player (Canada 6:0 Qatar)', () => {
		// Real data: Al Amin is shown sent off at 32' and again at 33'; Madibo gets a
		// red 51', a second yellow 52', then a red 53' — each is one dismissal.
		const raw: LiveEvent[] = [
			ev({ elapsed: 32, type: 'Card', detail: 'Red Card', team: 'Qatar', player: 'H. Al Amin' }),
			ev({ elapsed: 33, type: 'Card', detail: 'Red Card', team: 'Qatar', player: 'H. Al Amin' }),
			ev({ elapsed: 51, type: 'Card', detail: 'Red Card', team: 'Qatar', player: 'A. O. Madibo' }),
			ev({ elapsed: 52, type: 'Card', detail: 'Yellow Card', team: 'Qatar', player: 'A. O. Madibo' }),
			ev({ elapsed: 53, type: 'Card', detail: 'Red Card', team: 'Qatar', player: 'A. O. Madibo' })
		];

		const reds = decisiveEvents(raw).filter((e) => e.detail === 'Red Card');
		expect(reds.length).toBe(2);
		expect(reds.filter((e) => e.player.includes('Al Amin')).length).toBe(1);
		expect(reds.filter((e) => e.player.includes('Madibo')).length).toBe(1);
	});

	it('trims a goal doubled across half-time when the score is known (Canada 6:0)', () => {
		// "J. David" 45+3' and "Jonathan David" 48' are the same goal across the break;
		// same-minute dedup can't see it, but the 6:0 score reveals the phantom.
		const raw: LiveEvent[] = [
			ev({ elapsed: 16, team: 'Canada', player: 'C. Larin' }),
			ev({ elapsed: 29, team: 'Canada', player: 'J. David' }),
			ev({ elapsed: 45, extra: 3, team: 'Canada', player: 'J. David' }),
			ev({ elapsed: 48, team: 'Canada', player: 'Jonathan David' }),
			ev({ elapsed: 64, team: 'Canada', player: 'N. Saliba' }),
			ev({ elapsed: 75, team: 'Canada', player: '' }),
			ev({ elapsed: 75, team: 'Canada', player: 'M. Al Mannai', detail: 'Own Goal' }),
			ev({ elapsed: 90, extra: 2, team: 'Canada', player: 'J. David' }),
			ev({ elapsed: 90, extra: 2, team: 'Canada', player: 'J. David' })
		];

		// Same-minute dedup alone still leaves 7 goals for a 6:0 match.
		expect(decisiveEvents(raw).length).toBe(7);

		const out = decisiveEvents(raw, 6);
		expect(out.length).toBe(6);
		// The 45+3' duplicate is dropped; the fuller "Jonathan David" 48' is kept.
		expect(out.some((e) => e.elapsed === 45 && e.extra === 3)).toBe(false);
		expect(out.some((e) => e.elapsed === 48 && e.player === 'Jonathan David')).toBe(true);
		// Real David goals at 29' and 90+2' survive.
		expect(out.some((e) => e.elapsed === 29)).toBe(true);
		expect(out.some((e) => e.elapsed === 90 && e.extra === 2)).toBe(true);
	});

	it('never trims a real same-player brace when the count already matches', () => {
		// Balogun scores at 45+5' and 50' — distinct goals. A correct score must not
		// merge them.
		const raw: LiveEvent[] = [
			ev({ elapsed: 45, extra: 5, team: 'USA', player: 'F. Balogun' }),
			ev({ elapsed: 50, team: 'USA', player: 'Folarin Balogun' })
		];
		expect(decisiveEvents(raw, 2).length).toBe(2);
	});
});
