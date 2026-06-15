import { beforeEach, describe, expect, it } from 'vitest';
import type { LeagueSummary } from './api';
import type { Match, Team } from './tips.svelte';
import { normalizeSearchText, searchApp } from './search';

const teams: Record<string, Team> = {
	norway: { id: 'norway', name: 'Norge', iso2: 'NO', fifaCode: 'NOR' },
	brazil: { id: 'brazil', name: 'Brasil', iso2: 'BR', fifaCode: 'BRA' },
	southKorea: { id: 'southKorea', name: 'Sør-Korea', iso2: 'KR', fifaCode: 'KOR' },
	spain: { id: 'spain', name: 'Spania', iso2: 'ES', fifaCode: 'ESP' }
};

function match(partial: Partial<Match>): Match {
	return {
		id: 'm1',
		stage: 'group',
		groupLetter: 'A',
		roundLabel: 'Runde 1',
		num: 1,
		kickoff: '2026-06-12T18:00:00Z',
		tvChannel: 'NRK1',
		status: 'scheduled',
		homeTeam: 'norway',
		awayTeam: 'brazil',
		homeLabel: '',
		awayLabel: '',
		ftHome: 0,
		ftAway: 0,
		etHome: 0,
		etAway: 0,
		penHome: 0,
		penAway: 0,
		advancer: '',
		finalizedAt: '',
		...partial
	};
}

const leagues: LeagueSummary[] = [
	{ id: 'l1', name: 'Familieligaen', inviteCode: 'FAM123', role: 'owner', members: 8 },
	{ id: 'l2', name: 'Jobb VM', inviteCode: 'JOBB', role: 'member', members: 12 }
];

describe('normalizeSearchText', () => {
	beforeEach(() => {
		localStorage.clear();
		document.documentElement.lang = 'en';
	});

	it('normalizes case, accents and Norwegian letters', () => {
		expect(normalizeSearchText('  SØR-Koréa  ')).toBe('sor korea');
		expect(normalizeSearchText('Æ Å Ø')).toBe('ae a o');
	});
});

describe('searchApp', () => {
	beforeEach(() => {
		localStorage.clear();
		document.documentElement.lang = 'en';
	});

	it('returns empty groups for empty queries', () => {
		expect(searchApp('', { matches: [], teams, leagues })).toEqual({
			matches: [],
			teams: [],
			groups: [],
			leagues: []
		});
	});

	it('matches fixtures by team names and builds a tips link', () => {
		const results = searchApp('brasil', {
			matches: [match({ id: 'm-brasil' })],
			teams,
			leagues: []
		});

		expect(results.matches[0]).toMatchObject({
			id: 'm-brasil',
			title: 'Norway - Brazil',
			href: '/tips?match=m-brasil'
		});
	});

	it('matches teams with accent-insensitive queries', () => {
		const results = searchApp('sor korea', { matches: [], teams, leagues: [] });

		expect(results.teams[0]).toMatchObject({
			id: 'southKorea',
			title: 'South Korea',
			href: '/tips?team=southKorea'
		});
	});

	it('matches user leagues and links to the league page', () => {
		const results = searchApp('familie', { matches: [], teams: {}, leagues });

		expect(results.leagues[0]).toMatchObject({
			id: 'l1',
			title: 'Familieligaen',
			href: '/leagues/l1'
		});
	});

	it('matches groups and links to the group section on tips', () => {
		const results = searchApp('gruppe a', {
			matches: [
				match({ id: 'm-a1', homeTeam: 'norway', awayTeam: 'brazil', groupLetter: 'A' }),
				match({ id: 'm-a2', homeTeam: 'southKorea', awayTeam: 'spain', groupLetter: 'A' })
			],
			teams,
			leagues: []
		});

		expect(results.groups[0]).toMatchObject({
			id: 'A',
			title: 'Group A',
			href: '/tips?group=A'
		});
	});

	it('limits each result group', () => {
		const manyMatches = Array.from({ length: 6 }, (_, index) =>
			match({ id: `m${index}`, num: index + 1 })
		);

		const results = searchApp('norge', { matches: manyMatches, teams, leagues }, 3);

		expect(results.matches).toHaveLength(3);
	});
});