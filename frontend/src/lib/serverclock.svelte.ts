import { pb } from './pb';

// The app's clock comes from the server (/api/now) so client-side lock checks
// honor the dev virtual clock. Real time uses a measured offset; simulated
// time stays pinned to the server-provided timestamp.
class ServerClock {
	offset = $state(0);
	dev = $state(false);
	simulated = $state(false);
	simTime = $state<string | null>(null);
	loaded = $state(false);
	private refreshTimer: ReturnType<typeof setInterval> | null = null;
	private refreshPromise: Promise<void> | null = null;

	async refresh() {
		if (this.refreshPromise) return this.refreshPromise;
		this.refreshPromise = this.refreshInner().finally(() => {
			this.refreshPromise = null;
		});
		return this.refreshPromise;
	}

	private async refreshInner() {
		try {
			const r = await pb.send('/api/now', { method: 'GET' });
			this.offset = r.now - Date.now();
			this.dev = !!r.dev;
			this.simulated = !!r.simulated;
			this.simTime = r.simTime ?? null;
		} catch {
			/* fall back to local time */
		} finally {
			this.loaded = true;
		}
	}

	now(): number {
		if (this.simulated && this.simTime) {
			const fixed = Date.parse(this.simTime);
			if (!Number.isNaN(fixed)) return fixed;
		}
		return Date.now() + this.offset;
	}

	startAutoRefresh(intervalMs = 30_000) {
		if (this.refreshTimer) return;
		void this.refresh();
		this.refreshTimer = setInterval(() => void this.refresh(), intervalMs);
	}

	stopAutoRefresh() {
		if (!this.refreshTimer) return;
		clearInterval(this.refreshTimer);
		this.refreshTimer = null;
	}
}

export const serverClock = new ServerClock();
