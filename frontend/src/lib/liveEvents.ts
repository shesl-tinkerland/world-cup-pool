import type { LiveEvent, Team } from '$lib/tips.svelte';

// Shared rendering + filtering helpers for match events, used by both the live
// feed on the home page and the post-match summary in TipCard. Keeping them here
// means the two surfaces stay in lock-step on what an event looks like, which
// events count, and how provider duplicates are collapsed.

export function eventMinute(event: LiveEvent): string {
	return event.extra > 0 ? `${event.elapsed}+${event.extra}'` : `${event.elapsed}'`;
}

export function eventIcon(event: LiveEvent): string {
	if (event.type === 'Goal') return '⚽';
	if (event.type === 'Card' && event.detail === 'Red Card') return '🟥';
	if (event.type === 'Card') return '🟨';
	if (event.type === 'subst') return '↔';
	if (event.type === 'Var') return 'VAR';
	return '•';
}

// A "Goal"-type event is not always a goal: API-Football also files missed
// penalties under type "Goal" (detail "Missed Penalty"), which would otherwise
// render as ⚽ and inflate the score. Disallowed goals arrive as type "Var" and
// are excluded already. Own goals and penalties do count, so they stay.
const NON_SCORING_GOAL = /missed|disallow|cancel/i;

export function isGoal(event: LiveEvent): boolean {
	return event.type === 'Goal' && !NON_SCORING_GOAL.test(event.detail);
}

export function isRedCard(event: LiveEvent): boolean {
	return event.type === 'Card' && event.detail === 'Red Card';
}

// Decisive moments only: real goals and red cards. Yellow cards, substitutions,
// VAR checks and missed penalties are intentionally hidden.
export function isDecisiveEvent(event: LiveEvent): boolean {
	return isGoal(event) || isRedCard(event);
}

// A moment identifies one event in time: a team can't score, be sent off, or
// have a goal chalked off twice in the exact same minute.
function goalMoment(event: LiveEvent): string {
	return `${event.team}|${event.elapsed}|${event.extra}`;
}

// decisiveEvents filters to goals + red cards AND cleans up two provider quirks:
//
//  1. VAR cancellations. When a goal is disallowed, API-Football keeps the
//     original goal row as "Normal Goal" and files the reversal as a *separate*
//     VAR event ("Goal Disallowed - offside"). We collect those and drop the
//     orphaned goal, so a chalked-off goal doesn't inflate the score.
//
//  2. Duplicates. The same goal often arrives several times as the feed backfills
//     the assist or corrects the scorer's name/id (e.g. "F. Balogun" then "Folarin
//     Balogun" at 31'). We collapse by (type, moment) and keep the richest variant
//     — fullest name, with assist — so late corrections win.
export function decisiveEvents(events: LiveEvent[]): LiveEvent[] {
	const cancelledGoals = new Set<string>();
	for (const event of events) {
		if (event.type === 'Var' && /disallow|cancel/i.test(event.detail)) {
			cancelledGoals.add(goalMoment(event));
		}
	}

	const byMoment = new Map<string, LiveEvent>();
	for (const event of events) {
		if (!isDecisiveEvent(event)) continue;
		if (event.type === 'Goal' && cancelledGoals.has(goalMoment(event))) continue;
		const key = `${event.type}|${goalMoment(event)}`;
		const current = byMoment.get(key);
		if (!current || richer(event, current)) byMoment.set(key, event);
	}
	return [...byMoment.values()].sort((a, b) => a.elapsed - b.elapsed || a.extra - b.extra);
}

function richer(candidate: LiveEvent, current: LiveEvent): boolean {
	if (candidate.player.length !== current.player.length) {
		return candidate.player.length > current.player.length;
	}
	return (candidate.assist ? 1 : 0) > (current.assist ? 1 : 0);
}

function canonTeamName(name: string): string {
	return name.toLowerCase().replace(/[^a-z0-9]/g, '');
}

// Resolve which side of the match an event belongs to, so the UI can show the
// scoring country's flag. Prefers the backend-resolved teamId (alias-aware, so
// "Korea Republic" maps to our South Korea) and falls back to a normalized name
// match for events that arrive via realtime before a snapshot fills in teamId.
export function eventTeam(
	event: LiveEvent,
	candidates: (Team | null | undefined)[]
): Team | null {
	if (event.teamId) {
		for (const team of candidates) if (team && team.id === event.teamId) return team;
	}
	const want = canonTeamName(event.team);
	if (want) {
		for (const team of candidates) if (team && canonTeamName(team.name) === want) return team;
	}
	return null;
}

export function eventTitle(event: LiveEvent, assistLabel = 'Assist'): string {
	return [event.detail, event.team, event.assist ? `${assistLabel}: ${event.assist}` : '']
		.filter(Boolean)
		.join(' · ');
}
