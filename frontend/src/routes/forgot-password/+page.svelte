<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { language } from '$lib/language.svelte';
	import { strings } from '$lib/strings';

	let email = $state('');
	let busy = $state(false);
	let sent = $state(false);
	let error = $state('');
	const t = $derived(strings[language.resolved]);

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		busy = true;
		try {
			await auth.requestPasswordReset(email.trim());
			sent = true;
		} catch (err: unknown) {
			error = (err as { message?: string })?.message ?? t.forgotPassword.error;
		} finally {
			busy = false;
		}
	}
</script>

<div class="auth">
	<h1>{t.forgotPassword.title}</h1>
	<p class="muted">{t.forgotPassword.subtitle}</p>

	{#if sent}
		<div class="card">
			<p class="ok">{t.forgotPassword.success}</p>
			<p class="muted switch"><a href="/login">{t.forgotPassword.back}</a></p>
		</div>
	{:else}
		<form class="card" onsubmit={submit}>
			<div class="field">
				<label for="em">{t.forgotPassword.emailLabel}</label>
				<input
					id="em"
					class="input"
					type="email"
					bind:value={email}
					autocomplete="email"
					required
				/>
			</div>
			{#if error}<p class="error">{error}</p>{/if}
			<button class="btn" disabled={busy || !email.trim()}>
				{busy ? `${t.forgotPassword.send}…` : t.forgotPassword.send}
			</button>
			<p class="muted switch"><a href="/login">{t.forgotPassword.back}</a></p>
		</form>
	{/if}
</div>

<style>
	.auth {
		max-width: 380px;
		margin: 12dvh auto 0;
	}
	h1 {
		margin: 0;
		font-size: 1.8rem;
	}
	.muted {
		margin: 0.25rem 0 1.5rem;
	}
	.ok {
		color: var(--success);
		font-size: 0.95rem;
		margin: 0;
	}
	.switch {
		text-align: center;
		margin: 1rem 0 0;
	}
</style>
