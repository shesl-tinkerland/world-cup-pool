export interface TvChannelLogo {
	id: string;
	label: string;
	src: string;
	plate: string;
	border: string;
	fullBleed?: boolean;
}

function normalize(channel: string) {
	return channel
		.toLowerCase()
		.normalize('NFKD')
		.replace(/[^a-z0-9]+/g, '');
}

export function resolveTvChannel(channel = ''): TvChannelLogo | null {
	const label = channel.trim();
	const key = normalize(label);
	if (!key) return null;

	if (key.startsWith('nrk')) {
		return {
			id: 'nrk',
			label,
			src: '/tv-logos/NRK.png',
			plate: '#0b2a48',
			border: 'rgba(255, 255, 255, 0.16)',
			fullBleed: true
		};
	}

	if (key === 'tv2' || key.startsWith('tv2') || key === '2direkte') {
		return {
			id: 'tv2',
			label,
			src: '/tv-logos/tv2.png',
			plate: '#050505',
			border: 'rgba(255, 255, 255, 0.18)'
		};
	}

	return null;
}