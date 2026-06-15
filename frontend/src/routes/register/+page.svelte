<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { language } from '$lib/language.svelte';
	import { strings } from '$lib/strings';

	// After registering, resume an invite if one was carried in the URL.
	let invite = $derived($page.url.searchParams.get('invite'));
	function dest() {
		return invite ? `/join/${invite}` : '/';
	}
	let loginHref = $derived(
		invite ? `/login?invite=${encodeURIComponent(invite)}` : '/login'
	);

	let name = $state('');
	let email = $state('');
	let password = $state('');
	let error = $state('');
	let busy = $state(false);
	const t = $derived(strings[language.resolved]);

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		if (password.length < 8) {
			error = t.register.passwordTooShort;
			return;
		}
		busy = true;
		try {
			await auth.register(name, email, password);
			goto(dest());
		} catch (err: unknown) {
			error = (err as { message?: string })?.message ?? t.register.error;
		} finally {
			busy = false;
		}
	}
</script>

<div class="auth">
	<h1>{t.register.title}</h1>
	<p class="muted">{t.register.subtitle}</p>

	<form class="card" onsubmit={submit}>
		<div class="field">
			<label for="nm">{t.register.nameLabel}</label>
			<input id="nm" class="input" bind:value={name} required />
		</div>
		<div class="field">
			<label for="em">{t.auth.emailLabel}</label>
			<input
				id="em"
				class="input"
				type="email"
				bind:value={email}
				autocomplete="email"
				required
			/>
		</div>
		<div class="field">
			<label for="pw">{t.auth.passwordLabel}</label>
			<input
				id="pw"
				class="input"
				type="password"
				bind:value={password}
				autocomplete="new-password"
				required
			/>
		</div>
		{#if error}<p class="error">{error}</p>{/if}
		<button class="btn" disabled={busy}>{busy ? `${t.register.create}…` : t.register.create}</button>
		<p class="muted switch">
			{t.register.loginPrompt} <a href={loginHref}>{t.register.loginLink}</a>
		</p>
	</form>
</div>

<style>
	.auth {
		max-width: 380px;
		margin: 10dvh auto 0;
	}
	h1 {
		margin: 0;
		font-size: 1.8rem;
	}
	.muted {
		margin: 0.25rem 0 1.5rem;
	}
	.switch {
		text-align: center;
		margin: 1rem 0 0;
	}
</style>
