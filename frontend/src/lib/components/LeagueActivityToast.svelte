<script lang="ts">
	import { goto } from '$app/navigation';
	import { Mail, MessageCircle, X } from '@lucide/svelte';
	import { leagueBadges } from '$lib/leagueBadges.svelte';
	import { leagueInvitations } from '$lib/leagueInvitations.svelte';
	import { language } from '$lib/language.svelte';

	type ToastKind = 'chat' | 'invite';

	let visible = $state(false);
	let kind = $state<ToastKind>('chat');
	let title = $state('');
	let body = $state('');
	let href = $state('/leagues');
	// Plain (non-reactive) bookkeeping: these are only compared inside the
	// effect, and $state here would re-trigger the effect on its own writes.
	let baselineReady = false;
	let previousChatUnread = 0;
	let previousInviteCount = 0;
	let toastLeagueId = '';
	let hideTimer: ReturnType<typeof setTimeout> | null = null;

	const Icon = $derived(kind === 'chat' ? MessageCircle : Mail);

	$effect(() => {
		leagueBadges.start();
		return () => leagueBadges.stop();
	});

	$effect(() => {
		const chatUnread = leagueBadges.chatUnreadCount;
		const inviteCount = leagueInvitations.pendingCount;
		const firstUnread = leagueBadges.firstUnreadChat;

		// Wait for BOTH stores before taking the baseline. Invites load in
		// parallel with the chat overview; a baseline taken in between would
		// fire a toast for invites that are days old on every app open.
		if (!leagueBadges.loaded || !leagueInvitations.loaded) return;
		if (!baselineReady) {
			previousChatUnread = chatUnread;
			previousInviteCount = inviteCount;
			baselineReady = true;
			return;
		}

		const chatGrew = chatUnread > previousChatUnread;
		const invitesGrew = inviteCount > previousInviteCount;
		previousChatUnread = chatUnread;
		previousInviteCount = inviteCount;

		if (chatGrew && firstUnread) {
			toastLeagueId = firstUnread.leagueId;
			showToast(
				'chat',
				firstUnread.leagueName,
				firstUnread.unread === 1
					? language.text('Ny melding i liga-chatten', 'Ny melding i liga-chatten', 'New league chat message')
					: language.text(`${firstUnread.unread} nye meldinger`, `${firstUnread.unread} nye meldingar`, `${firstUnread.unread} new messages`),
				`/leagues/${firstUnread.leagueId}#chat`
			);
		} else if (invitesGrew) {
			toastLeagueId = '';
			showToast(
				'invite',
				language.text('Ny ligainvitasjon', 'Ny ligainvitasjon', 'New league invite'),
				language.text('Åpne Ligaer for å svare.', 'Opne Ligaer for å svare.', 'Open Leagues to respond.'),
				'/leagues'
			);
		} else if (visible) {
			// The activity the toast points at was handled elsewhere (its chat
			// was opened, the invite answered) — don't keep advertising it.
			if (kind === 'chat' && toastLeagueId && leagueBadges.unreadForLeague(toastLeagueId) === 0) dismiss();
			else if (kind === 'invite' && inviteCount === 0) dismiss();
		}
	});

	function showToast(nextKind: ToastKind, nextTitle: string, nextBody: string, nextHref: string) {
		kind = nextKind;
		title = nextTitle;
		body = nextBody;
		href = nextHref;
		visible = true;
		if (hideTimer) clearTimeout(hideTimer);
		hideTimer = setTimeout(() => {
			visible = false;
			hideTimer = null;
		}, 6500);
	}

	function dismiss(event?: MouseEvent) {
		event?.stopPropagation();
		visible = false;
		if (hideTimer) clearTimeout(hideTimer);
		hideTimer = null;
	}

	async function openToast() {
		dismiss();
		await goto(href);
	}
</script>

<div class="toast-region" role="status" aria-live="polite">
	{#if visible}
		<div
			class="league-activity-toast"
			class:invite={kind === 'invite'}
			role="button"
			tabindex="0"
			onclick={openToast}
			onkeydown={(event) => {
				if (event.key === 'Enter' || event.key === ' ') {
					event.preventDefault();
					void openToast();
				}
			}}
		>
			<span class="toast-icon"><Icon size={17} /></span>
			<span class="toast-copy">
				<b>{title}</b>
				<span>{body}</span>
			</span>
			<button class="toast-close" type="button" aria-label={language.text('Lukk varsel', 'Lukk varsel', 'Dismiss notification')} onclick={dismiss}>
				<X size={15} />
			</button>
		</div>
	{/if}
</div>

<style>
	.toast-region {
		position: fixed;
		right: max(1rem, env(safe-area-inset-right));
		bottom: max(1rem, calc(env(safe-area-inset-bottom) + 1rem));
		z-index: 80;
		pointer-events: none;
	}
	.league-activity-toast {
		pointer-events: auto;
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		align-items: center;
		gap: 0.7rem;
		width: min(25rem, calc(100vw - 2rem));
		padding: 0.78rem 0.82rem;
		border: 1px solid color-mix(in srgb, var(--live) 36%, var(--border));
		border-radius: 14px;
		background:
			linear-gradient(135deg, color-mix(in srgb, var(--live) 12%, transparent), transparent 58%),
			color-mix(in srgb, var(--surface) 96%, transparent);
		color: var(--text);
		box-shadow: 0 20px 44px -26px rgba(0, 0, 0, 0.58);
		text-align: left;
		cursor: pointer;
		animation: toastIn 180ms ease-out;
	}
	.league-activity-toast.invite {
		border-color: color-mix(in srgb, var(--accent) 34%, var(--border));
		background:
			linear-gradient(135deg, color-mix(in srgb, var(--accent) 11%, transparent), transparent 58%),
			color-mix(in srgb, var(--surface) 96%, transparent);
	}
	.toast-icon {
		display: grid;
		place-items: center;
		width: 2.25rem;
		height: 2.25rem;
		border-radius: var(--radius-pill);
		background: var(--live);
		color: var(--bg);
	}
	.invite .toast-icon {
		background: var(--accent);
	}
	.toast-copy {
		display: grid;
		gap: 0.14rem;
		min-width: 0;
	}
	.toast-copy b,
	.toast-copy span {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.toast-copy b {
		font-size: 0.88rem;
	}
	.toast-copy span {
		color: var(--muted);
		font-size: 0.78rem;
		font-weight: 650;
	}
	.toast-close {
		border: 0;
		background: transparent;
		display: grid;
		place-items: center;
		width: 1.9rem;
		height: 1.9rem;
		border-radius: var(--radius-pill);
		color: var(--muted);
	}
	.toast-close:hover,
	.toast-close:focus-visible {
		background: var(--surface-2);
		color: var(--text);
		outline: none;
	}
	@keyframes toastIn {
		from {
			opacity: 0;
			transform: translateY(8px) scale(0.98);
		}
		to {
			opacity: 1;
			transform: translateY(0) scale(1);
		}
	}
	@media (max-width: 700px) {
		.toast-region {
			right: max(0.8rem, env(safe-area-inset-right));
			bottom: max(5.6rem, calc(env(safe-area-inset-bottom) + 5.4rem));
			left: max(0.8rem, env(safe-area-inset-left));
			display: flex;
			justify-content: flex-end;
		}
		.league-activity-toast {
			width: min(100%, 24rem);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.league-activity-toast {
			animation: none;
		}
	}
</style>
