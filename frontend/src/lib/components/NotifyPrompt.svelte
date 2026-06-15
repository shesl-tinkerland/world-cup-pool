<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type NotifyEvent, type NotifyPrefs } from '$lib/api';
	import { push } from '$lib/push.svelte';
	import { auth } from '$lib/auth.svelte';
	import { language } from '$lib/language.svelte';
	import { Bell, X } from '@lucide/svelte';

	// One-time onboarding popup that offers to turn on notifications. "Seen"
	// state is stored server-side (notifyPromptSeenAt) so it shows exactly once
	// across all the user's devices.
	let open = $state(false);
	let saving = $state(false);
	let events = $state<NotifyEvent[]>([]);
	let emailOn = $state(true);
	let pushOn = $state(false);

	onMount(async () => {
		if (!auth.isAuthed) return;
		try {
			const res = await api.notifyPrefs();
			events = res.events ?? [];
			if (res.promptSeen) return;
			pushOn = push.supported;
			// Small delay so it doesn't slam the user the instant the app paints.
			setTimeout(() => {
				open = true;
			}, 700);
		} catch {
			/* never block the app on this */
		}
	});

	async function markSeen() {
		try {
			await api.markNotifyPromptSeen();
		} catch {
			/* ignore */
		}
	}

	async function dismiss() {
		open = false;
		await markSeen();
	}

	async function enable() {
		saving = true;
		try {
			let pushReady = false;
			if (pushOn && push.supported) {
				pushReady = await push.enable();
			}
			const prefs: NotifyPrefs = {};
			for (const ev of events) {
				const channels: Partial<Record<'email' | 'push', boolean>> = {};
				if (ev.channels.includes('email')) channels.email = emailOn;
				if (ev.channels.includes('push')) channels.push = pushOn && pushReady;
				prefs[ev.key] = channels;
			}
			await api.updateNotifyPrefs(prefs);
		} catch {
			/* ignore — user can still set prefs in Settings */
		} finally {
			saving = false;
			open = false;
			await markSeen();
		}
	}
</script>

{#if open}
	<button class="np-backdrop" aria-label="Lukk" onclick={() => dismiss()}></button>
	<div class="np-card" role="dialog" aria-labelledby="np-title" aria-modal="true">
		<button class="np-x" aria-label={language.text('Lukk', 'Lukk', 'Close')} onclick={() => dismiss()}>
			<X size={18} />
		</button>

		<div class="np-head">
			<div class="np-crest"><Bell size={22} /></div>
			<div class="np-kicker">{language.text('VM 2026', 'VM 2026', 'World Cup 2026')}</div>
			<h2 id="np-title">
				{language.text('Vil du ha varsler?', 'Vil du ha varsel?', 'Want notifications?')}
			</h2>
			<p class="np-lead">
				{language.text(
					'Vi kan minne deg på å levere tipsene dine før kampene starter. Velg selv hvordan.',
					'Vi kan minne deg på å levere tipsa dine før kampane startar. Vel sjølv korleis.',
					'We can remind you to submit your tips before matches start. Choose how.'
				)}
			</p>
		</div>

		<div class="np-options">
			<label class="np-opt">
				<input type="checkbox" bind:checked={emailOn} />
				<span>
					<strong>{language.text('E-post', 'E-post', 'Email')}</strong>
					<span class="muted small"
						>{language.text(
							'Påminnelser på e-post.',
							'Påminningar på e-post.',
							'Reminders by email.'
						)}</span
					>
				</span>
			</label>

			<label class="np-opt" class:disabled={!push.supported}>
				<input type="checkbox" bind:checked={pushOn} disabled={!push.supported} />
				<span>
					<strong>{language.text('Push', 'Push', 'Push')}</strong>
					<span class="muted small">
						{#if push.supported}
							{language.text(
								'Varsler rett på enheten.',
								'Varsel rett på eininga.',
								'Notifications straight to your device.'
							)}
						{:else}
							{language.text(
								'Ikke støttet i denne nettleseren.',
								'Ikkje støtta i denne nettlesaren.',
								'Not supported in this browser.'
							)}
						{/if}
					</span>
				</span>
			</label>
		</div>

		<div class="np-actions">
			<button class="np-btn ghost" onclick={() => dismiss()} disabled={saving}>
				{language.text('Ikke nå', 'Ikkje no', 'Not now')}
			</button>
			<button class="np-btn primary" onclick={() => enable()} disabled={saving || (!emailOn && !pushOn)}>
				{saving
					? language.text('Lagrer…', 'Lagrar…', 'Saving…')
					: language.text('Slå på', 'Slå på', 'Turn on')}
			</button>
		</div>

		<p class="np-foot muted small">
			{language.text(
				'Du kan endre dette når som helst i Innstillinger.',
				'Du kan endre dette når som helst i Innstillingar.',
				'You can change this any time in Settings.'
			)}
		</p>
	</div>
{/if}

<style>
	.np-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(3, 9, 14, 0.55);
		border: none;
		padding: 0;
		z-index: 80;
		cursor: pointer;
		backdrop-filter: blur(2px);
	}
	.np-card {
		position: fixed;
		left: 50%;
		top: 50%;
		transform: translate(-50%, -50%);
		z-index: 81;
		width: min(420px, calc(100vw - 2rem));
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		box-shadow: var(--shadow-pop, 0 24px 60px rgba(3, 9, 14, 0.4));
		padding: 1.4rem 1.3rem 1.1rem;
		overflow: hidden;
	}
	.np-card::before {
		content: '';
		position: absolute;
		inset: 0 0 auto 0;
		height: 4px;
		background: linear-gradient(90deg, #ffe27a, #ffcf3a, #b88408);
	}
	.np-x {
		position: absolute;
		top: 0.6rem;
		right: 0.6rem;
		display: inline-grid;
		place-items: center;
		width: 34px;
		height: 34px;
		border-radius: 999px;
		background: transparent;
		color: var(--muted);
		border: 1px solid transparent;
		cursor: pointer;
	}
	.np-x:hover {
		color: var(--text);
		background: color-mix(in srgb, var(--text) 8%, transparent);
	}
	.np-head {
		text-align: center;
		margin-top: 0.3rem;
	}
	.np-crest {
		display: inline-grid;
		place-items: center;
		width: 54px;
		height: 54px;
		margin: 0 auto 0.6rem;
		border-radius: 16px;
		color: #2b210a;
		background: linear-gradient(150deg, #fff7d6, #d9bb72 55%, #8e651e);
		box-shadow: 0 6px 18px rgba(142, 101, 30, 0.35);
	}
	.np-kicker {
		font-size: 0.72rem;
		font-weight: 700;
		letter-spacing: 0.14em;
		text-transform: uppercase;
		color: #b88408;
	}
	.np-head h2 {
		margin: 0.25rem 0 0.4rem;
		font-size: 1.25rem;
	}
	.np-lead {
		margin: 0 auto;
		max-width: 32ch;
		color: var(--muted);
		font-size: 0.92rem;
		line-height: 1.45;
	}
	.np-options {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
		margin: 1.1rem 0 0.4rem;
	}
	.np-opt {
		display: flex;
		align-items: flex-start;
		gap: 0.65rem;
		padding: 0.7rem 0.8rem;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		cursor: pointer;
	}
	.np-opt.disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}
	.np-opt input {
		margin-top: 0.15rem;
		width: 1.1rem;
		height: 1.1rem;
		flex: none;
		cursor: inherit;
	}
	.np-opt span {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		line-height: 1.3;
	}
	.np-actions {
		display: flex;
		gap: 0.6rem;
		margin-top: 0.9rem;
	}
	.np-btn {
		flex: 1;
		padding: 0.7rem 1rem;
		border-radius: 10px;
		font-size: 0.95rem;
		font-weight: 600;
		cursor: pointer;
		border: 1px solid var(--border);
	}
	.np-btn:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}
	.np-btn.ghost {
		background: transparent;
		color: var(--text);
	}
	.np-btn.primary {
		border: none;
		color: #071019;
		background: linear-gradient(180deg, #ffd95a, #f0b400);
	}
	.np-foot {
		text-align: center;
		margin: 0.8rem 0 0;
	}
</style>
