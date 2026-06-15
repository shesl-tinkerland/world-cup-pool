import type { LeagueSummary } from './api';
import { readRuntimeLocale, runtimeText } from './runtimeLanguage';
import type { Match, Team } from './tips.svelte';
import { teamDisplayName } from './teamNames';

export type SearchGroup = 'matches' | 'teams' | 'groups' | 'leagues';

export interface SearchResult {
	id: string;
	group: SearchGroup;
	title: string;
	subtitle: string;
	href: string;
	keywords: string;
}

export interface SearchSources {
	matches: Match[];
	teams: Record<string, Team>;
	leagues: LeagueSummary[];
}

export interface SearchResults {
	matches: SearchResult[];
	teams: SearchResult[];
	groups: SearchResult[];
	leagues: SearchResult[];
}

const emptyResults = (): SearchResults => ({ matches: [], teams: [], groups: [], leagues: [] });

export function normalizeSearchText(value: string): string {
	return value
		.toLocaleLowerCase(readRuntimeLocale())
		.normalize('NFD')
		.replace(/[\u0300-\u036f]/g, '')
		.replace(/æ/g, 'ae')
		.replace(/ø/g, 'o')
		.replace(/å/g, 'a')
		.replace(/[-–—:]/g, ' ')
		.replace(/[^a-z0-9\s]/g, ' ')
		.replace(/\s+/g, ' ')
		.trim();
}

function teamName(teams: Record<string, Team>, id: string, fallback: string): string {
	return teamDisplayName(teams[id], fallback || runtimeText('Ukjent lag', 'Ukjent lag', 'Unknown team'));
}

function teamSearchKeywords(teams: Record<string, Team>, id: string, fallback: string): string[] {
	const team = teams[id];
	const display = teamName(teams, id, fallback);
	return [display, team?.name, team?.fifaCode, team?.iso2, fallback].filter(
		(value): value is string => Boolean(value)
	);
}

function matchStageLabel(match: Match): string {
	if (match.stage === 'group') {
		return `${runtimeText('Gruppe', 'Gruppe', 'Group')} ${match.groupLetter} · ${match.roundLabel}`;
	}
	return match.roundLabel || match.stage;
}

function matchTimeLabel(iso: string): string {
	return new Date(iso).toLocaleString(readRuntimeLocale(), {
		weekday: 'short',
		day: 'numeric',
		month: 'short',
		hour: '2-digit',
		minute: '2-digit'
	});
}

function localeSort(a: string, b: string): number {
	return a.localeCompare(b, readRuntimeLocale());
}

function buildGroupResults(matches: Match[], teams: Record<string, Team>): SearchResult[] {
	const grouped: Record<string, { matches: Match[]; teamNames: Set<string> }> = {};
	const groupedKeywords: Record<string, Set<string>> = {};
	for (const match of matches) {
		if (match.stage !== 'group' || !match.groupLetter) continue;
		const bucket = (grouped[match.groupLetter] ||= {
			matches: [],
			teamNames: new Set<string>()
		});
		const keywordBucket = (groupedKeywords[match.groupLetter] ||= new Set<string>());
		bucket.matches.push(match);
		bucket.teamNames.add(teamName(teams, match.homeTeam, match.homeLabel));
		bucket.teamNames.add(teamName(teams, match.awayTeam, match.awayLabel));
		for (const keyword of teamSearchKeywords(teams, match.homeTeam, match.homeLabel)) {
			keywordBucket.add(keyword);
		}
		for (const keyword of teamSearchKeywords(teams, match.awayTeam, match.awayLabel)) {
			keywordBucket.add(keyword);
		}
	}

	return Object.keys(grouped)
		.sort()
		.map((letter) => {
			const group = grouped[letter];
			const keywordBucket = groupedKeywords[letter] || new Set<string>();
			const teamsInGroup = Array.from(group.teamNames)
				.filter(Boolean)
				.sort(localeSort);
			const teamSummary = teamsInGroup.slice(0, 4).join(', ');
			const label = runtimeText('Gruppe', 'Gruppe', 'Group');
			const matchLabel = runtimeText(
				group.matches.length === 1 ? 'kamp' : 'kamper',
				group.matches.length === 1 ? 'kamp' : 'kampar',
				group.matches.length === 1 ? 'match' : 'matches'
			);
			return {
				id: letter,
				group: 'groups' as const,
				title: `${label} ${letter}`,
				subtitle: `${group.matches.length} ${matchLabel}${
					teamSummary ? ` · ${teamSummary}` : ''
				}`,
				href: `/tips?group=${encodeURIComponent(letter)}`,
				keywords: [`group ${letter}`, `gruppe ${letter}`, letter, ...keywordBucket].join(' ')
			};
		});
}

export function buildSearchIndex({ matches, teams, leagues }: SearchSources): SearchResults {
	return {
		matches: matches.map((match) => {
			const home = teamName(teams, match.homeTeam, match.homeLabel);
			const away = teamName(teams, match.awayTeam, match.awayLabel);
			const homeKeywords = teamSearchKeywords(teams, match.homeTeam, match.homeLabel);
			const awayKeywords = teamSearchKeywords(teams, match.awayTeam, match.awayLabel);
			const stage = matchStageLabel(match);
			return {
				id: match.id,
				group: 'matches',
				title: `${home} - ${away}`,
				subtitle: `${stage} · ${matchTimeLabel(match.kickoff)}`,
				href: `/tips?match=${encodeURIComponent(match.id)}`,
				keywords: [...homeKeywords, ...awayKeywords, stage, match.tvChannel, match.num, match.status]
					.filter(Boolean)
					.join(' ')
			};
		}),
		teams: Object.values(teams)
			.sort((a, b) => teamDisplayName(a).localeCompare(teamDisplayName(b), readRuntimeLocale()))
			.map((team) => ({
				id: team.id,
				group: 'teams',
				title: teamDisplayName(team),
				subtitle: team.fifaCode
					? `${runtimeText('Lag', 'Lag', 'Team')} · ${team.fifaCode}`
					: runtimeText('Lag', 'Lag', 'Team'),
				href: `/tips?team=${encodeURIComponent(team.id)}`,
				keywords: [team.name, team.fifaCode, team.iso2].filter(Boolean).join(' ')
			})),
		groups: buildGroupResults(matches, teams),
		leagues: leagues.map((league) => ({
			id: league.id,
			group: 'leagues',
			title: league.name,
			subtitle: `${league.members} ${
				runtimeText(
					league.members === 1 ? 'medlem' : 'medlemmer',
					league.members === 1 ? 'medlem' : 'medlemer',
					league.members === 1 ? 'member' : 'members'
				)
			} · ${
				league.inviteCode === 'GLOBAL'
					? 'Global'
					: league.role === 'owner'
						? runtimeText('Eier', 'Eigar', 'Owner')
						: runtimeText('Medlem', 'Medlem', 'Member')
			}`,
			href: `/leagues/${encodeURIComponent(league.id)}`,
			keywords: [league.name, league.inviteCode, league.role].filter(Boolean).join(' ')
		}))
	};
}

function scoreResult(result: SearchResult, query: string): number {
	const title = normalizeSearchText(result.title);
	const haystack = normalizeSearchText(`${result.title} ${result.subtitle} ${result.keywords}`);
	if (title === query) return 0;
	if (title.startsWith(query)) return 1;
	if (title.split(' ').some((part) => part.startsWith(query))) return 4;
	const titleIndex = title.indexOf(query);
	if (titleIndex >= 0) return 10 + titleIndex;
	const fullIndex = haystack.indexOf(query);
	return fullIndex >= 0 ? 30 + fullIndex : Number.POSITIVE_INFINITY;
}

function searchGroup(items: SearchResult[], query: string, limit: number): SearchResult[] {
	return items
		.map((item) => ({ item, score: scoreResult(item, query) }))
		.filter(({ score }) => Number.isFinite(score))
		.sort((a, b) => a.score - b.score || a.item.title.localeCompare(b.item.title, readRuntimeLocale()))
		.slice(0, limit)
		.map(({ item }) => item);
}

export function searchApp(query: string, sources: SearchSources, limitPerGroup = 5): SearchResults {
	const normalized = normalizeSearchText(query);
	if (!normalized) return emptyResults();
	const index = buildSearchIndex(sources);
	return {
		matches: searchGroup(index.matches, normalized, limitPerGroup),
		teams: searchGroup(index.teams, normalized, limitPerGroup),
		groups: searchGroup(index.groups, normalized, limitPerGroup),
		leagues: searchGroup(index.leagues, normalized, limitPerGroup)
	};
}

export function totalSearchResults(results: SearchResults): number {
	return results.matches.length + results.teams.length + results.groups.length + results.leagues.length;
}
