import type { Match } from './tips.svelte'

export function applyRealtimeMatchBatch(
	currentMatches: Match[],
	currentLiveMatchIds: Set<string>,
	updates: Match[]
) {
	const nextMatches = [...currentMatches]
	const nextLiveMatchIds = new Set(currentLiveMatchIds)
	let needsSort = false
	let finalizedChanged = false

	for (const match of updates) {
		const idx = nextMatches.findIndex((current) => current.id === match.id)
		if (idx >= 0) nextMatches[idx] = match
		else {
			nextMatches.push(match)
			needsSort = true
		}

		if (isLiveStatus(match.status)) nextLiveMatchIds.add(match.id)
		else nextLiveMatchIds.delete(match.id)

		if (match.finalizedAt) finalizedChanged = true
	}

	if (needsSort) {
		nextMatches.sort(
			(a, b) => new Date(a.kickoff).getTime() - new Date(b.kickoff).getTime()
		)
	}

	return {
		matches: nextMatches,
		liveMatchIds: nextLiveMatchIds,
		finalizedChanged
	}
}

function isLiveStatus(status: string): boolean {
	return ['live', '1H', '2H', 'HT', 'ET', 'BT', 'P', 'LIVE', 'INT'].includes(status)
}