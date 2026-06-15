// Reactive Web Push state and actions. Wraps the browser Push API: registers
// the push service worker, subscribes with the backend's VAPID key, and mirrors
// the subscription to the server so the backend can deliver notifications.

import { api } from './api';
import { pb } from './pb';

const SW_URL = '/service-worker.js';
const SW_SCOPE = '/';

function urlBase64ToUint8Array(base64String: string): Uint8Array {
	const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
	const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
	const raw = atob(base64);
	const out = new Uint8Array(raw.length);
	for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i);
	return out;
}

function bufferToBase64Url(buf: ArrayBuffer | null | undefined): string {
	if (!buf) return '';
	const bytes = new Uint8Array(buf);
	let bin = '';
	for (const b of bytes) bin += String.fromCharCode(b);
	return btoa(bin).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

// Does the subscription use the given VAPID public key? A subscription created
// with an older key can never receive pushes signed with the current one.
function matchesServerKey(sub: PushSubscription, publicKey: string): boolean {
	const subKey = bufferToBase64Url(sub.options?.applicationServerKey);
	if (!subKey) return true; // unknown — assume fine rather than churn
	return subKey === publicKey.replace(/=+$/, '');
}

class Push {
	supported = $state(false);
	permission = $state<NotificationPermission>('default');
	subscribed = $state(false);
	busy = $state(false);

	constructor() {
		if (typeof window === 'undefined') return;
		this.supported =
			'serviceWorker' in navigator &&
			'PushManager' in window &&
			'Notification' in window;
		if (!this.supported) return;
		this.permission = Notification.permission;
		void this.refresh();
		pb.authStore.onChange(() => void this.syncAccount(), true);
	}

	private get uid(): string | null {
		return pb.authStore.isValid ? (pb.authStore.record?.id ?? null) : null;
	}

	private async registerSubscription(sub: PushSubscription) {
		const json = sub.toJSON();
		await api.pushSubscribe({
			endpoint: sub.endpoint,
			keys: {
				p256dh: json.keys?.p256dh ?? '',
				auth: json.keys?.auth ?? ''
			}
		});
	}

	// Re-create the subscription when the server's VAPID key no longer matches
	// the one it was created with (key rotation, backup restore). Requires
	// permission to already be granted; returns a live subscription either way.
	private async freshenSubscription(
		reg: ServiceWorkerRegistration,
		sub: PushSubscription
	): Promise<PushSubscription> {
		try {
			const { publicKey } = await api.pushVapidKey();
			if (!publicKey || matchesServerKey(sub, publicKey)) return sub;
			await sub.unsubscribe().catch(() => {});
			return await reg.pushManager.subscribe({
				userVisibleOnly: true,
				applicationServerKey: urlBase64ToUint8Array(publicKey) as BufferSource
			});
		} catch {
			return sub;
		}
	}

	// On sign-in (and app open while signed in), mirror this device's existing
	// subscription to the server unconditionally. The upsert is idempotent and
	// cheap, and trusting local "already registered" state goes stale whenever
	// the server loses the row (backup restore, account switch).
	private async syncAccount() {
		if (!this.supported || !this.uid) return;
		try {
			const reg = await navigator.serviceWorker.getRegistration(SW_SCOPE);
			let sub = await reg?.pushManager.getSubscription();
			if (!reg || !sub) return;
			if (Notification.permission === 'granted') {
				sub = await this.freshenSubscription(reg, sub);
			}
			await this.registerSubscription(sub);
			this.subscribed = true;
		} catch {
			/* best effort; the next auth change or enable() call can retry */
		}
	}

	/** Check whether this device already has an active push subscription. */
	async refresh() {
		if (!this.supported) return;
		try {
			const reg = await navigator.serviceWorker.getRegistration(SW_SCOPE);
			const sub = await reg?.pushManager.getSubscription();
			this.subscribed = !!sub;
		} catch {
			this.subscribed = false;
		}
	}

	/** Request permission, subscribe, and register with the backend. Returns
	 *  true on success. Safe to call when already subscribed (idempotent). */
	async enable(): Promise<boolean> {
		if (!this.supported || this.busy) return false;
		this.busy = true;
		try {
			const permission = await Notification.requestPermission();
			this.permission = permission;
			if (permission !== 'granted') return false;

			const { publicKey } = await api.pushVapidKey();
			if (!publicKey) return false;

			const reg = await navigator.serviceWorker.register(SW_URL);
			await navigator.serviceWorker.ready;

			let sub = await reg.pushManager.getSubscription();
			if (sub && !matchesServerKey(sub, publicKey)) {
				// Stale VAPID key — replace the subscription or the server's
				// pushes will be rejected by the push service forever.
				await sub.unsubscribe().catch(() => {});
				sub = null;
			}
			if (!sub) {
				sub = await reg.pushManager.subscribe({
					userVisibleOnly: true,
					applicationServerKey: urlBase64ToUint8Array(publicKey) as BufferSource
				});
			}

			await this.registerSubscription(sub);
			this.subscribed = true;
			return true;
		} catch {
			return false;
		} finally {
			this.busy = false;
		}
	}

	/** Unsubscribe this device and tell the backend to drop it. */
	async disable(): Promise<void> {
		if (!this.supported || this.busy) return;
		this.busy = true;
		try {
			const reg = await navigator.serviceWorker.getRegistration(SW_SCOPE);
			const sub = await reg?.pushManager.getSubscription();
			if (sub) {
				const endpoint = sub.endpoint;
				await sub.unsubscribe().catch(() => {});
				await api.pushUnsubscribe(endpoint).catch(() => {});
			}
			this.subscribed = false;
		} finally {
			this.busy = false;
		}
	}
}

export const push = new Push();
