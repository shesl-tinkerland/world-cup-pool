import { api, type LeagueInvite } from '$lib/api';

class LeagueInvitations {
	invites = $state<LeagueInvite[]>([]);
	loaded = $state(false);
	loading = $state(false);

	get pending() {
		return this.invites.filter((invite) => invite.status === 'pending');
	}

	get pendingCount() {
		return this.pending.length;
	}

	async load(force = false) {
		if (this.loading || (this.loaded && !force)) return;
		this.loading = true;
		try {
			this.invites = (await api.myLeagueInvitations()).invites;
			this.loaded = true;
		} catch {
			// Keep the last good list. Wiping on a failed poll makes the next
			// successful load look like brand-new invites and fires stale toasts.
		} finally {
			this.loading = false;
		}
	}

	remove(id: string) {
		this.invites = this.invites.filter((invite) => invite.id !== id);
	}

	clear() {
		this.invites = [];
		this.loaded = false;
		this.loading = false;
	}
}

export const leagueInvitations = new LeagueInvitations();