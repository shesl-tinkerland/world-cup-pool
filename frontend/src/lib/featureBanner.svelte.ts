import { pb } from './pb';

// Per-user, dismiss-once announcement state for the league "Kampar" view. Mirrors
// homeIntro: a single localStorage flag keyed by user id, so the "Nytt" badge and
// banner disappear for good once the user opens the feature or closes the banner.
// Bumping the version suffix re-announces a future feature to everyone.
const STORAGE_PREFIX = 'feature-league-matches-dismissed-v1';

function storageKey(userId: string) {
	return `${STORAGE_PREFIX}:${userId}`;
}

class FeatureBanner {
	dismissed = $state(false);
	private userId = '';

	constructor() {
		if (typeof window === 'undefined') return;
		this.sync();
		pb.authStore.onChange(() => this.sync());
	}

	get visible() {
		return !!this.userId && !this.dismissed;
	}

	sync() {
		this.userId = pb.authStore.record?.id ?? '';
		if (!this.userId) {
			this.dismissed = false;
			return;
		}
		try {
			this.dismissed = localStorage.getItem(storageKey(this.userId)) === '1';
		} catch {
			this.dismissed = false;
		}
	}

	dismiss() {
		if (!this.userId || this.dismissed) return;
		this.dismissed = true;
		try {
			localStorage.setItem(storageKey(this.userId), '1');
		} catch {
			/* ignore (private mode / storage disabled) */
		}
	}
}

export const featureBanner = new FeatureBanner();
