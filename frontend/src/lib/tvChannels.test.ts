import { describe, expect, it } from 'vitest';
import { resolveTvChannel } from './tvChannels';

describe('resolveTvChannel', () => {
	it('resolves NRK variants to the NRK logo', () => {
		expect(resolveTvChannel('NRK1')).toMatchObject({ id: 'nrk', src: '/tv-logos/NRK.png', fullBleed: true });
		expect(resolveTvChannel('NRK 2')).toMatchObject({ id: 'nrk' });
	});

	it('resolves TV 2 variants to the dark TV2 plate', () => {
		expect(resolveTvChannel('TV2')).toMatchObject({ id: 'tv2', src: '/tv-logos/tv2.png', plate: '#050505' });
		expect(resolveTvChannel('TV 2 Direkte')).toMatchObject({ id: 'tv2' });
	});

	it('returns null for unknown channels', () => {
		expect(resolveTvChannel('')).toBeNull();
		expect(resolveTvChannel('Some Channel')).toBeNull();
		expect(resolveTvChannel('MTV2')).toBeNull();
	});
});