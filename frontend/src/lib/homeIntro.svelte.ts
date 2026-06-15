import { pb } from './pb';

const STORAGE_PREFIX = 'home-intro-dismissed-v1';

function storageKey(userId: string) {
	return `${STORAGE_PREFIX}:${userId}`;
}

class HomeIntro {
	ready = $state(false);
	visible = $state(false);
	dismissed = $state(false);
	private userId = '';

	constructor() {
		if (typeof window === 'undefined') return;
		this.sync();
		pb.authStore.onChange(() => this.sync());
	}

	sync() {
		this.userId = pb.authStore.record?.id ?? '';
		if (!this.userId) {
			this.dismissed = false;
			this.visible = false;
			this.ready = true;
			return;
		}

		try {
			this.dismissed = localStorage.getItem(storageKey(this.userId)) === '1';
		} catch {
			this.dismissed = false;
		}

		this.visible = !this.dismissed;
		this.ready = true;
	}

	dismiss() {
		if (!this.userId) return;
		this.dismissed = true;
		this.visible = false;
		try {
			localStorage.setItem(storageKey(this.userId), '1');
		} catch {
			/* ignore (private mode / storage disabled) */
		}
	}

	reopen() {
		if (!this.userId) return;
		this.dismissed = false;
		this.visible = true;
		try {
			localStorage.removeItem(storageKey(this.userId));
		} catch {
			/* ignore (private mode / storage disabled) */
		}
	}
}

export const homeIntro = new HomeIntro();