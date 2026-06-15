<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { Check, Mail, X } from '@lucide/svelte';
	import { api, type LeagueInvite } from '$lib/api';
	import { leagueInvitations } from '$lib/leagueInvitations.svelte';
	import { language } from '$lib/language.svelte';
	import Avatar from '$lib/components/Avatar.svelte';

	let {
		compact = false,
		homeTile = false,
		class: className = ''
	}: { compact?: boolean; homeTile?: boolean; class?: string } = $props();

	let busyId = $state('');
	let error = $state('');
	const locale = $derived(language.locale);
	const copy = $derived(
		language.text(
			{
				kicker: 'Ligainvitasjon',
				title: 'Ventende invitasjoner',
				invitedBy: 'Invitert av',
				accepting: 'Godtar...',
				accept: 'Godta',
				declining: 'Avslår...',
				decline: 'Avslå',
				acceptError: 'Kunne ikke godta invitasjonen.',
				declineError: 'Kunne ikke avslå invitasjonen.'
			},
			{
				kicker: 'Ligainvitasjon',
				title: 'Ventande invitasjonar',
				invitedBy: 'Invitert av',
				accepting: 'Godtek...',
				accept: 'Godta',
				declining: 'Avslår...',
				decline: 'Avslå',
				acceptError: 'Kunne ikkje godta invitasjonen.',
				declineError: 'Kunne ikkje avslå invitasjonen.'
			},
			{
				kicker: 'League invite',
				title: 'Pending invites',
				invitedBy: 'Invited by',
				accepting: 'Accepting...',
				accept: 'Accept',
				declining: 'Declining...',
				decline: 'Decline',
				acceptError: 'Could not accept the invite.',
				declineError: 'Could not decline the invite.'
			}
		)
	);

	let visibleInvites = $derived(leagueInvitations.pending);

	onMount(() => {
		void leagueInvitations.load(true);
	});

	function sentLabel(iso: string) {
		const date = new Date(iso);
		if (!Number.isFinite(date.getTime())) return '';
		return new Intl.DateTimeFormat(locale, {
			day: '2-digit',
			month: 'short',
			hour: '2-digit',
			minute: '2-digit'
		}).format(date);
	}

	async function accept(invite: LeagueInvite) {
		busyId = `${invite.id}:accept`;
		error = '';
		try {
			const result = await api.acceptLeagueInvitation(invite.id);
			leagueInvitations.remove(invite.id);
			await goto(`/leagues/${result.league.id}`);
		} catch {
			error = copy.acceptError;
		} finally {
			busyId = '';
		}
	}

	async function decline(invite: LeagueInvite) {
		busyId = `${invite.id}:decline`;
		error = '';
		try {
			await api.declineLeagueInvitation(invite.id);
			leagueInvitations.remove(invite.id);
		} catch {
			error = copy.declineError;
		} finally {
			busyId = '';
		}
	}
</script>

{#if leagueInvitations.loaded && visibleInvites.length > 0}
	<section class={`card pending-invites ${className}`} class:compact class:home-tile={homeTile} aria-labelledby="pending-invites-title">
		<div class="invite-title">
			<span class="invite-icon"><Mail size={18} /></span>
			<div>
				<p class="kicker">{copy.kicker}</p>
				<h2 id="pending-invites-title">{copy.title}</h2>
			</div>
		</div>

		<div class="invite-list">
			{#each visibleInvites as invite (invite.id)}
				<article class="invite-row">
					<Avatar name={invite.invitedBy.name} src={invite.invitedBy.avatarUrl} size={42} />
					<div class="invite-main">
						<b>{invite.leagueName}</b>
						<span>
							{copy.invitedBy} {invite.invitedBy.name}
							{#if sentLabel(invite.created)} · {sentLabel(invite.created)}{/if}
						</span>
					</div>
					<div class="invite-actions">
						<button
							class="btn accept"
							disabled={!!busyId}
							onclick={() => accept(invite)}
						>
							<Check size={16} /> {busyId === `${invite.id}:accept` ? copy.accepting : copy.accept}
						</button>
						<button
							class="btn secondary decline"
							disabled={!!busyId}
							onclick={() => decline(invite)}
						>
							<X size={16} /> {busyId === `${invite.id}:decline` ? copy.declining : copy.decline}
						</button>
					</div>
				</article>
			{/each}
		</div>

		{#if error}<p class="error">{error}</p>{/if}
	</section>
{/if}

<style>
	.pending-invites {
		display: grid;
		gap: 0.85rem;
		border-color: color-mix(in srgb, var(--accent) 34%, var(--border));
		background: color-mix(in srgb, var(--accent) 5%, var(--surface));
	}
	.home-tile {
		grid-column: 1 / -1;
		min-height: 160px;
	}
	.invite-title {
		display: flex;
		align-items: center;
		gap: 0.7rem;
	}
	.invite-title h2,
	.invite-title p {
		margin: 0;
	}
	.invite-title h2 {
		font-size: 1.08rem;
	}
	.invite-icon {
		display: grid;
		place-items: center;
		width: 42px;
		height: 42px;
		border-radius: var(--radius);
		background: color-mix(in srgb, var(--accent) 13%, var(--surface-2));
		color: var(--accent);
	}
	.invite-list {
		display: grid;
		gap: 0.6rem;
	}
	.invite-row {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		align-items: center;
		gap: 0.75rem;
		padding: 0.72rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface);
	}
	.invite-main {
		display: grid;
		gap: 0.2rem;
		min-width: 0;
	}
	.invite-main b {
		overflow-wrap: anywhere;
	}
	.invite-main span {
		color: var(--muted);
		font-size: 0.86rem;
	}
	.invite-actions {
		display: flex;
		gap: 0.45rem;
		flex-wrap: wrap;
		justify-content: flex-end;
	}
	.invite-actions .btn {
		width: auto;
		min-width: 6.5rem;
		padding: 0.62rem 0.75rem;
	}
	.invite-actions .btn.accept {
		background: var(--success);
		border-color: var(--success);
		color: var(--bg);
	}
	.compact .invite-title h2 {
		font-size: 1rem;
	}
	.home-tile .invite-title h2 {
		font-size: 1.05rem;
	}
	.compact .invite-row {
		grid-template-columns: minmax(0, 1fr);
	}
	.home-tile .invite-row {
		grid-template-columns: minmax(0, 1fr);
		align-items: stretch;
	}
	.compact .invite-row :global(.avatar) {
		display: none;
	}
	.home-tile .invite-row :global(.avatar) {
		display: none;
	}
	.compact .invite-actions {
		justify-content: stretch;
	}
	.home-tile .invite-actions {
		justify-content: stretch;
	}
	.compact .invite-actions .btn {
		flex: 1;
	}
	.home-tile .invite-actions .btn {
		flex: 1;
	}
	@media (min-width: 640px) {
		.home-tile {
			grid-column: span 4;
		}
	}
	@media (min-width: 900px) {
		.home-tile {
			grid-column: span 3;
		}
	}
	@media (min-width: 1200px) {
		.home-tile {
			grid-column: span 4;
		}
	}
	@media (max-width: 620px) {
		.invite-row {
			grid-template-columns: auto minmax(0, 1fr);
		}
		.invite-actions {
			grid-column: 1 / -1;
			justify-content: stretch;
		}
		.invite-actions .btn {
			flex: 1;
		}
	}
</style>
