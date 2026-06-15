// Reactive PWA install state. One singleton (`pwa`) tracks whether the app
// can be installed on the current device and exposes a single `install()`
// action: on Android/desktop Chromium it fires the captured native prompt;
// on iOS it opens an inline coaching panel (Apple doesn't expose a prompt
// API — Add-to-Home-Screen is only reachable from the Share menu).

interface BeforeInstallPromptEvent extends Event {
	prompt(): Promise<void>;
	userChoice: Promise<{ outcome: 'accepted' | 'dismissed'; platform: string }>;
}

const BANNER_STORAGE_ID = 'pwa-banner-dismissed-v1';

class Pwa {
	installed = $state(false); // running as an installed PWA already
	canPrompt = $state(false); // native beforeinstallprompt is queued
	iosCoach = $state(false); // iOS device, show Share-button instructions
	bannerOpen = $state(false); // one-time banner is currently shown
	iosHelpOpen = $state(false); // user opened the iOS help panel
	dismissed = $state(false); // first-visit banner has been dismissed

	private deferred: BeforeInstallPromptEvent | null = null;

	constructor() {
		if (typeof window === 'undefined') return;

		try {
			this.dismissed = localStorage.getItem(BANNER_STORAGE_ID) === '1';
		} catch {
			/* ignore (private mode / storage disabled) */
		}

		const mql = window.matchMedia('(display-mode: standalone)');
		const refreshInstalled = () => {
			this.installed =
				mql.matches ||
				// iOS Safari uses the legacy navigator.standalone flag.
				(navigator as unknown as { standalone?: boolean }).standalone ===
					true;
		};
		refreshInstalled();
		mql.addEventListener?.('change', refreshInstalled);

		// iOS has no beforeinstallprompt event — detect the platform up-front
		// so we can still surface a button and instructions.
		const ua = navigator.userAgent;
		const isIos = /iPad|iPhone|iPod/.test(ua);
		if (isIos && !this.installed) {
			this.iosCoach = true;
			this.maybeOpenBanner();
		}

		window.addEventListener('beforeinstallprompt', (e) => {
			e.preventDefault();
			this.deferred = e as BeforeInstallPromptEvent;
			this.canPrompt = true;
			this.maybeOpenBanner();
		});

		window.addEventListener('appinstalled', () => {
			this.installed = true;
			this.canPrompt = false;
			this.iosCoach = false;
			this.iosHelpOpen = false;
			this.bannerOpen = false;
			this.deferred = null;
		});
	}

	/** Anything to show in the topbar at all? */
	get available() {
		return !this.installed && (this.canPrompt || this.iosCoach);
	}

	async install() {
		// iOS path takes priority when active. In real-world use only one
		// branch is ever set (Safari doesn't fire beforeinstallprompt and
		// Android isn't an iOS UA), but Chrome emulating an iPhone trips
		// both — and there we want to see the iOS coaching sheet.
		if (this.iosCoach) {
			this.iosHelpOpen = true;
			this.bannerOpen = false;
			return;
		}
		if (this.deferred) {
			try {
				await this.deferred.prompt();
				const { outcome } = await this.deferred.userChoice;
				if (outcome === 'accepted') this.installed = true;
			} catch {
				// Some browsers throw if prompt() was already consumed; the
				// state below still resets so the UI stops offering it.
			} finally {
				this.deferred = null;
				this.canPrompt = false;
				this.bannerOpen = false;
			}
		}
	}

	dismissBanner() {
		this.bannerOpen = false;
		this.dismissed = true;
		try {
			localStorage.setItem(BANNER_STORAGE_ID, '1');
		} catch {
			/* ignore (private mode / storage disabled) */
		}
	}

	closeIosHelp() {
		this.iosHelpOpen = false;
	}

	private maybeOpenBanner() {
		if (this.installed || this.bannerOpen) return;
		try {
			if (localStorage.getItem(BANNER_STORAGE_ID) === '1') return;
		} catch {
			/* ignore */
		}
		this.bannerOpen = true;
	}
}

export const pwa = new Pwa();
