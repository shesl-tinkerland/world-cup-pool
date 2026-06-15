import { language } from '$lib/language.svelte';

const knockoutLabels = {
	nb: {
		R32: '32-delsfinale',
		R16: 'Åttedelsfinale',
		QF: 'Kvartfinale',
		SF: 'Semifinale',
		'3RD': 'Bronsefinale',
		FINAL: 'Finale'
	},
	nn: {
		R32: 'Sluttspel 32',
		R16: 'Åttedelsfinale',
		QF: 'Kvartfinale',
		SF: 'Semifinale',
		'3RD': 'Bronsefinale',
		FINAL: 'Finale'
	},
	en: {
		R32: 'Round of 32',
		R16: 'Round of 16',
		QF: 'Quarter-finals',
		SF: 'Semi-finals',
		'3RD': 'Third-place play-off',
		FINAL: 'Final'
	}
} as const;

export function stageName(stage: string) {
	return knockoutLabels[language.resolved][stage as keyof (typeof knockoutLabels)[typeof language.resolved]] ?? stage;
}

export function matchStageLabel(match: { stage: string; groupLetter?: string }) {
	if (match.stage === 'group') {
		return `${language.text('Gruppespill', 'Gruppespel', 'Group stage')} · ${language.text('Gruppe', 'Gruppe', 'Group')} ${match.groupLetter ?? ''}`;
	}
	return stageName(match.stage);
}

export const stageOrder = ['R32', 'R16', 'QF', 'SF', '3RD', 'FINAL'];
