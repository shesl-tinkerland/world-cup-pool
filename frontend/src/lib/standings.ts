import type { Match, Tip } from './tips.svelte';

export type StandRow = {
	id: string;
	pts: number;
	gf: number;
	ga: number;
	pld: number;
};

function cmp(a: StandRow, b: StandRow): number {
	return (
		b.pts - a.pts ||
		b.gf - b.ga - (a.gf - a.ga) ||
		b.gf - a.gf ||
		a.id.localeCompare(b.id)
	);
}

export function groupTable(matches: Match[], tips: Record<string, Tip>): StandRow[] {
	const table: Record<string, StandRow> = {};
	const ensure = (id: string) =>
		(table[id] ||= { id, pts: 0, gf: 0, ga: 0, pld: 0 });

	for (const match of matches) {
		if (match.homeTeam) ensure(match.homeTeam);
		if (match.awayTeam) ensure(match.awayTeam);
	}

	for (const match of matches) {
		if (!match.homeTeam || !match.awayTeam) continue;
		const played = match.status === 'finished' || !!match.finalizedAt;
		let homeGoals: number;
		let awayGoals: number;
		if (played) {
			homeGoals = match.ftHome;
			awayGoals = match.ftAway;
		} else {
			const tip = tips[match.id];
			if (!tip) continue;
			homeGoals = tip.ftHome;
			awayGoals = tip.ftAway;
		}
		const home = ensure(match.homeTeam);
		const away = ensure(match.awayTeam);
		home.pld++;
		away.pld++;
		home.gf += homeGoals;
		home.ga += awayGoals;
		away.gf += awayGoals;
		away.ga += homeGoals;
		if (homeGoals > awayGoals) home.pts += 3;
		else if (awayGoals > homeGoals) away.pts += 3;
		else {
			home.pts++;
			away.pts++;
		}
	}

	return Object.values(table).sort(cmp);
}

export function groupComplete(matches: Match[], tips: Record<string, Tip>): boolean {
	return (
		matches.length > 0 &&
		matches.every(
			(match) =>
				!match.homeTeam ||
				!match.awayTeam ||
				match.status === 'finished' ||
				!!match.finalizedAt ||
				!!tips[match.id]
		)
	);
}

export function bestThirds(
	groups: Match[][],
	tips: Record<string, Tip>,
	count = 8
): Set<string> {
	const thirds: StandRow[] = [];
	for (const group of groups) {
		if (!groupComplete(group, tips)) return new Set<string>();
		const rows = groupTable(group, tips);
		if (rows.length >= 3) thirds.push(rows[2]);
	}
	thirds.sort(cmp);
	return new Set(thirds.slice(0, count).map((row) => row.id));
}