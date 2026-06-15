import { describe, expect, it } from 'vitest'

import { applyRealtimeMatchBatch } from './tipsRealtime'
import type { Match } from './tips.svelte'

function match(overrides: Partial<Match> = {}): Match {
	return {
		id: 'm1',
		stage: 'group',
		groupLetter: 'A',
		roundLabel: 'Round 1',
		num: 1,
		kickoff: '2026-06-11T20:00:00Z',
		tvChannel: '',
		status: 'scheduled',
		homeTeam: 't1',
		awayTeam: 't2',
		homeLabel: 'Home',
		awayLabel: 'Away',
		ftHome: 0,
		ftAway: 0,
		etHome: 0,
		etAway: 0,
		penHome: 0,
		penAway: 0,
		advancer: '',
		finalizedAt: '',
		...overrides
	}
}

describe('applyRealtimeMatchBatch', () => {
	it('keeps only the latest update per match and recalculates live ids once', () => {
		const current = [match()]
		const next = applyRealtimeMatchBatch(current, new Set<string>(), [
			match({ id: 'm1', status: 'live' }),
			match({ id: 'm1', status: 'finished', finalizedAt: '2026-06-11T22:05:00Z', ftHome: 2, ftAway: 1 })
		])

		expect(next.matches).toHaveLength(1)
		expect(next.matches[0]).toMatchObject({ status: 'finished', ftHome: 2, ftAway: 1 })
		expect([...next.liveMatchIds]).toEqual([])
		expect(next.finalizedChanged).toBe(true)
	})

	it('sorts newly inserted matches by kickoff time', () => {
		const current = [match({ id: 'late', kickoff: '2026-06-12T20:00:00Z' })]
		const next = applyRealtimeMatchBatch(current, new Set<string>(), [
			match({ id: 'early', kickoff: '2026-06-11T20:00:00Z', status: 'live' })
		])

		expect(next.matches.map((item) => item.id)).toEqual(['early', 'late'])
		expect([...next.liveMatchIds]).toEqual(['early'])
	})
})