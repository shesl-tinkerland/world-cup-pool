<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { homeIntro } from '$lib/homeIntro.svelte';
	import { goto } from '$app/navigation';
	import Avatar from '$lib/components/Avatar.svelte';
	import { language } from '$lib/language.svelte';
	import { strings } from '$lib/strings';
	import { api, type PlayerStats, type NotifyEvent, type NotifyPrefs } from '$lib/api';
	import { push } from '$lib/push.svelte';
	import { onMount } from 'svelte';

	const MAX_AVATAR_BYTES = 5 * 1024 * 1024; // PocketBase users-avatar default

	let name = $state(auth.user?.name ?? '');
	let avatarFile = $state<File | null>(null);
	let previewUrl = $state<string | null>(null);
	let error = $state('');
	let saved = $state(false);
	let busy = $state(false);
	let fileInput: HTMLInputElement;

	let resetBusy = $state(false);
	let resetSent = $state(false);
	let resetError = $state('');
	let deletePhrase = $state('');
	let deleteBusy = $state(false);
	let deleteError = $state('');
	let introReset = $state(false);
	const t = $derived(strings[language.resolved]);
	const isEnglish = $derived(language.isEnglish);
	const introCopy = $derived(strings[language.resolved].introCard);

	// Player Card stats (own profile only).
	let stats = $state<PlayerStats | null>(null);
	let statsBusy = $state(true);
	onMount(async () => {
		try {
			stats = await api.playerStats();
		} catch {
			stats = null;
		} finally {
			statsBusy = false;
		}
	});
	const hitRatePct = $derived(
		stats && stats.hitRate.total > 0
			? Math.round(stats.hitRate.pct * 100)
			: 0
	);
	const hasScored = $derived(!!stats && stats.tipsScored > 0);

	async function sendReset() {
		if (!auth.user?.email) return;
		resetError = '';
		resetSent = false;
		resetBusy = true;
		try {
			await auth.requestPasswordReset(auth.user.email);
			resetSent = true;
		} catch (err: unknown) {
			resetError =
				(err as { message?: string })?.message ??
				language.text('Kunne ikke sende lenken.', 'Kunne ikkje sende tilbakestillingslenke.', 'Could not send reset link.');
		} finally {
			resetBusy = false;
		}
	}

	// Revoke the object URL when it's replaced or the page unmounts.
	$effect(() => {
		const url = previewUrl;
		return () => {
			if (url) URL.revokeObjectURL(url);
		};
	});

	function pickFile(e: Event) {
		const file = (e.target as HTMLInputElement).files?.[0];
		if (!file) return;
		if (!file.type.startsWith('image/')) {
			error = language.text('Velg en bildefil.', 'Vel ei bildefil.', 'Choose an image file.');
			return;
		}
		if (file.size > MAX_AVATAR_BYTES) {
			error = language.text('Bildet må være 5 MB eller mindre.', 'Bildet må vere 5 MB eller mindre.', 'Image must be 5 MB or smaller.');
			return;
		}
		error = '';
		saved = false;
		avatarFile = file;
		previewUrl = URL.createObjectURL(file);
	}

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		saved = false;
		const trimmed = name.trim();
		if (trimmed.length < 1 || trimmed.length > 48) {
			error = language.text(
				'Visningsnavnet må være mellom 1 og 48 tegn.',
				'Visingsnamnet må vere mellom 1 og 48 teikn.',
				'Display name must be between 1 and 48 characters.'
			);
			return;
		}
		busy = true;
		try {
			await auth.updateProfile({ name: trimmed, avatarFile });
			avatarFile = null;
			previewUrl = null;
			if (fileInput) fileInput.value = '';
			saved = true;
		} catch (err: unknown) {
			error =
				(err as { message?: string })?.message ??
				language.text('Kunne ikke lagre endringene.', 'Kunne ikkje lagre endringane.', 'Could not save changes.');
		} finally {
			busy = false;
		}
	}

	async function destroyAccount() {
		deleteError = '';
		const confirmWord = deletePhrase.trim().toUpperCase();
		if (confirmWord !== 'DELETE' && confirmWord !== 'SLETT') {
			deleteError = language.text(
				'Skriv DELETE eller SLETT for å bekrefte sletting.',
				'Skriv DELETE eller SLETT for å stadfeste kontosletting.',
				'Type DELETE to confirm account removal.'
			);
			return;
		}
		if (!confirm(language.text(
			'Vil du virkelig slette kontoen din? Private ligaer du eier, blir også slettet.',
			'Vil du verkeleg slette kontoen din? Private ligaer du eig, blir også sletta.',
			'Do you really want to delete your account? Private leagues you own will also be deleted.'
		))) {
			return;
		}
		deleteBusy = true;
		try {
			await auth.deleteAccount();
			await goto('/register');
		} catch (err: unknown) {
			deleteError =
				(err as { message?: string })?.message ??
				language.text('Kunne ikke slette kontoen.', 'Kunne ikkje slette kontoen.', 'Could not delete account.');
		} finally {
			deleteBusy = false;
		}
	}

	function reactivateIntro() {
		homeIntro.reopen();
		introReset = true;
	}

	// Notification preferences — the user chooses what they receive. Everything
	// is opt-in (off by default); changes save immediately on toggle.
	let notifyEvents = $state<NotifyEvent[]>([]);
	let notifyPrefs = $state<NotifyPrefs>({});
	let notifyBusy = $state(true);
	let notifySaving = $state(false);
	let notifyError = $state('');
	onMount(async () => {
		try {
			const res = await api.notifyPrefs();
			notifyEvents = res.events ?? [];
			notifyPrefs = res.prefs ?? {};
		} catch {
			notifyError = language.text(
				'Kunne ikke laste varselinnstillinger.',
				'Kunne ikkje laste varselinnstillingar.',
				'Could not load notification settings.'
			);
		} finally {
			notifyBusy = false;
		}
	});

	function isOn(key: string, channel: string) {
		return !!notifyPrefs[key]?.[channel as 'email' | 'push'];
	}

	async function toggleNotify(key: string, channel: 'email' | 'push') {
		const turningOn = !isOn(key, channel);
		notifyError = '';

		// Turning push on also mirrors the current browser subscription to this
		// user on the backend. If the user denies it, don't save the preference.
		if (channel === 'push' && turningOn) {
			notifySaving = true;
			const ok = await push.enable();
			notifySaving = false;
			if (!ok) {
				notifyError = language.text(
					'Kunne ikke slå på push. Tillat varsler i nettleseren.',
					'Kunne ikkje slå på push. Tillat varsel i nettlesaren.',
					'Could not enable push. Allow notifications in your browser.'
				);
				return;
			}
		}

		const prev = $state.snapshot(notifyPrefs) as NotifyPrefs;
		const next: NotifyPrefs = { ...prev, [key]: { ...prev[key], [channel]: turningOn } };
		notifyPrefs = next;
		notifySaving = true;
		try {
			const res = await api.updateNotifyPrefs(next);
			notifyPrefs = res.prefs ?? next;
		} catch {
			notifyPrefs = prev;
			notifyError = language.text(
				'Kunne ikke lagre varselinnstillinger.',
				'Kunne ikkje lagre varselinnstillingar.',
				'Could not save notification settings.'
			);
		} finally {
			notifySaving = false;
		}
	}

	function notifyEventLabel(key: string) {
		switch (key) {
			case 'pre_kickoff_reminder':
				return language.text('Påminnelse før avspark', 'Påminning før avspark', 'Pre-kickoff reminder');
			case 'upcoming_matches_not_tipped':
				return language.text('Kamper du ikke har tipset', 'Kampar du ikkje har tipsa', 'Matches you haven’t tipped');
			default:
				return key;
		}
	}
	function notifyEventDesc(key: string) {
		switch (key) {
			case 'pre_kickoff_reminder':
				return language.text(
					'Få en påminnelse dagen før VM starter hvis du ikke har levert alt.',
					'Få ei påminning dagen før VM startar viss du ikkje har levert alt.',
					'Get a reminder the day before the World Cup starts if you haven’t submitted everything.'
				);
			case 'upcoming_matches_not_tipped':
				return language.text(
					'Få beskjed når kamper snart starter og du ikke har tipset dem ennå.',
					'Få beskjed når kampar snart startar og du ikkje har tipsa dei enno.',
					'Get notified when matches are starting soon and you haven’t tipped them yet.'
				);
			default:
				return '';
		}
	}
	function notifyChannelLabel(ch: string) {
		return ch === 'push'
			? language.text('Push', 'Push', 'Push')
			: language.text('E-post', 'E-post', 'Email');
	}

	// Self-service push test: makes sure permission + subscription are in place
	// on this device, then asks the backend to deliver a real test notification.
	let pushTestBusy = $state(false);
	let pushTestResult = $state('');
	let pushTestFailed = $state(false);
	async function sendPushTest() {
		pushTestBusy = true;
		pushTestResult = '';
		pushTestFailed = false;
		try {
			const ready = await push.enable();
			if (!ready) {
				pushTestFailed = true;
				pushTestResult = language.text(
					'Kunne ikke aktivere push. Tillat varsler for siden i nettleseren og prøv igjen.',
					'Kunne ikkje aktivere push. Tillat varsel for sida i nettlesaren og prøv igjen.',
					'Could not enable push. Allow notifications for this site in your browser and try again.'
				);
				return;
			}
			const res = await api.pushTest();
			if (res.sent) {
				pushTestResult = language.text(
					'Testvarsel sendt! Det skal dukke opp på enheten i løpet av få sekunder.',
					'Testvarsel sendt! Det skal dukke opp på eininga i løpet av få sekund.',
					'Test notification sent! It should appear on your device within seconds.'
				);
			} else if (res.devices === 0) {
				pushTestFailed = true;
				pushTestResult = language.text(
					'Ingen enheter er registrert for push på kontoen din ennå.',
					'Ingen einingar er registrerte for push på kontoen din enno.',
					'No devices are registered for push on your account yet.'
				);
			} else {
				pushTestFailed = true;
				pushTestResult = language.text(
					`Leveringen feilet for alle ${res.devices} enhet(er). Slå push av og på igjen, og prøv på nytt.`,
					`Leveringa feila for alle ${res.devices} eining(ar). Slå push av og på igjen, og prøv på nytt.`,
					`Delivery failed for all ${res.devices} device(s). Toggle push off and on again, then retry.`
				);
			}
		} catch {
			pushTestFailed = true;
			pushTestResult = language.text(
				'Noe gikk galt ved sending av testvarselet.',
				'Noko gjekk gale ved sending av testvarselet.',
				'Something went wrong sending the test notification.'
			);
		} finally {
			pushTestBusy = false;
		}
	}
</script>

<div class="settings">
	<h1>{t.chrome.settings}</h1>
	<p class="muted">{language.text('Endre hvordan du vises for venner.', 'Endre korleis du visest for vener.', 'Edit how you appear to friends.')}</p>

	<section class="player-card" aria-labelledby="player-card-title">
		<div class="pc-topline">
			<div class="pc-rating" data-testid="stat-hitrate">
				<div class="pc-rating-number">{hitRatePct}</div>
				<div class="pc-rating-unit">%</div>
				<div class="pc-rating-label">{t.playerCard.hitRate}</div>
				{#if stats && stats.hitRate.total > 0}
					<div class="pc-rating-sub">
						{stats.hitRate.count} / {stats.hitRate.total} {t.playerCard.hitRateSub}
					</div>
				{/if}
			</div>

			<div class="pc-crest" aria-hidden="true">
				<span>VM</span>
				<small>26</small>
			</div>
		</div>

		<div class="pc-portrait" aria-hidden="true">
			<div class="pc-avatar-shell">
				<Avatar name={auth.user?.name || '?'} src={auth.user?.avatarUrl} size={112} />
			</div>
		</div>

		<div class="pc-identity">
			<p>{t.playerCard.title}</p>
			<h2 id="player-card-title">{auth.user?.name ?? ''}</h2>
			<div class="pc-stars" aria-hidden="true">
				<span></span><span></span><span></span><span></span><span></span>
			</div>
		</div>

		{#if statsBusy}
			<div class="pc-loading" aria-hidden="true"><span></span><span></span><span></span></div>
			<p class="muted small">…</p>
		{:else if !hasScored}
			<div class="pc-empty">{t.playerCard.noStats}</div>
		{:else if stats}
			<div class="pc-stats">
				<div class="pc-stat">
					<div class="pc-stat-value">{hitRatePct}%</div>
					<div class="pc-stat-label">{t.playerCard.hitRate}</div>
				</div>
				<div class="pc-stat" data-testid="stat-streak">
					<div class="pc-stat-value">{stats.longestStreak}</div>
					<div class="pc-stat-label">{t.playerCard.longestStreak}</div>
					<div class="pc-stat-sub">{t.playerCard.longestStreakSub}</div>
				</div>
				<div class="pc-stat">
					<div class="pc-stat-value">{stats.tipsScored}</div>
					<div class="pc-stat-label">{language.text('Poengtips', 'Scora tips', 'Scored tips')}</div>
				</div>
			</div>

			<div class="pc-miss-panel" data-testid="stat-miss">
				{#if stats.largestMiss}
					<div class="pc-miss-kicker">{t.playerCard.largestMiss}</div>
					<div class="pc-miss-match">
						{stats.largestMiss.homeLabel} {stats.largestMiss.actualHome}–{stats.largestMiss.actualAway} {stats.largestMiss.awayLabel}
					</div>
					<div class="pc-miss-sub">
						{t.playerCard.largestMissSub} {stats.largestMiss.tipHome}–{stats.largestMiss.tipAway}
						· {language.text('avvik', 'avvik', 'gap')} {stats.largestMiss.gdDev}
					</div>
				{:else}
					<div class="pc-miss-kicker">{t.playerCard.largestMiss}</div>
					<div class="pc-miss-match">—</div>
				{/if}
			</div>
		{/if}
	</section>

	<form class="card" onsubmit={submit}>
		<div class="avatar-row">
			<Avatar
				name={name || auth.user?.name || '?'}
				src={previewUrl ?? auth.user?.avatarUrl}
				size={96}
			/>
			<div>
				<button
					type="button"
					class="btn secondary"
					onclick={() => fileInput.click()}
					disabled={busy}
				>
					{language.text('Bytt bilde', 'Byt bilde', 'Change photo')}
				</button>
				<p class="muted hint">{language.text('PNG eller JPG, opptil 5 MB.', 'PNG eller JPG, opptil 5 MB.', 'PNG or JPG, up to 5 MB.')}</p>
			</div>
			<input
				bind:this={fileInput}
				type="file"
				accept="image/*"
				class="hidden-file"
				onchange={pickFile}
			/>
		</div>

		<div class="field">
			<label for="dn">{language.text('Visningsnavn', 'Visingsnamn', 'Display name')}</label>
			<input
				id="dn"
				class="input"
				bind:value={name}
				maxlength="48"
				autocomplete="name"
				required
			/>
		</div>

		{#if error}<p class="error">{error}</p>{/if}
		{#if saved}<p class="ok">{language.text('Lagret.', 'Lagra.', 'Saved.')}</p>{/if}

		<button class="btn" disabled={busy}>{busy ? language.text('Lagrer...', 'Lagrar…', 'Saving…') : language.text('Lagre endringer', 'Lagre endringar', 'Save changes')}</button>
	</form>

	<section class="card intro-pref-card">
		<h3>{introCopy.settingsTitle}</h3>
		<p class="muted small intro-pref-status">
			{homeIntro.dismissed ? introCopy.settingsDismissed : introCopy.settingsActive}
		</p>
		<p class="muted small intro-pref-body">{introCopy.settingsBody}</p>
		{#if introReset}
			<p class="ok intro-pref-ok">
				{introCopy.settingsSuccess} <a href="/">{introCopy.settingsLink}</a>
			</p>
		{/if}
		<div class="intro-pref-actions">
			<button type="button" class="btn secondary" onclick={reactivateIntro}>
				{introCopy.settingsButton}
			</button>
		</div>
	</section>

	<section class="card">
		<h3>{language.text('Passord', 'Passord', 'Password')}</h3>
		<p class="muted small">
			{#if isEnglish}
				We will send a reset link to <strong>{auth.user?.email ?? ''}</strong>.
				Use it to choose a new password.
			{:else if language.isNynorsk}
				Vi sender ei tilbakestillingslenke til <strong>{auth.user?.email ?? ''}</strong>.
				Bruk henne til å velje nytt passord.
			{:else}
				Vi sender en tilbakestillingslenke til <strong>{auth.user?.email ?? ''}</strong>.
				Bruk den til å velge nytt passord.
			{/if}
		</p>
		{#if resetError}<p class="error">{resetError}</p>{/if}
		{#if resetSent}
			<p class="ok">{language.text('Lenke sendt - sjekk innboksen.', 'Tilbakestillingslenke sendt - sjekk innboksen.', 'Reset link sent - check your inbox.')}</p>
		{/if}
		<button
			type="button"
			class="btn secondary"
			onclick={sendReset}
			disabled={resetBusy || resetSent}
		>
			{resetBusy
				? language.text('Sender...', 'Sender…', 'Sending…')
				: resetSent ? language.text('Sendt', 'Sendt', 'Sent') : language.text('Send lenke', 'Send tilbakestillingslenke', 'Send reset link')}
		</button>
	</section>

	<section class="card">
		<h3>{language.text('Varsler', 'Varsel', 'Notifications')}</h3>
		<p class="muted small">
			{language.text(
				'Velg selv hva du vil få beskjed om. Alt er av som standard.',
				'Vel sjølv kva du vil få beskjed om. Alt er av som standard.',
				'Choose what you want to be notified about. Everything is off by default.'
			)}
		</p>
		{#if notifyBusy}
			<p class="muted small">…</p>
		{:else if notifyEvents.length === 0}
			<p class="muted small">
				{language.text(
					'Ingen varseltyper er tilgjengelige ennå.',
					'Ingen varseltypar er tilgjengelege enno.',
					'No notification types are available yet.'
				)}
			</p>
		{:else}
			<ul class="notify-list">
				{#each notifyEvents as ev (ev.key)}
					<li class="notify-row">
						<div class="notify-text">
							<span class="notify-name">{notifyEventLabel(ev.key)}</span>
							<span class="muted small">{notifyEventDesc(ev.key)}</span>
						</div>
						<div class="notify-toggles">
							{#each ev.channels as ch (ch)}
								<label class="notify-toggle" class:disabled={ch === 'push' && !push.supported}>
									<input
										type="checkbox"
										checked={isOn(ev.key, ch)}
										disabled={notifySaving || (ch === 'push' && !push.supported)}
										onchange={() => toggleNotify(ev.key, ch)}
									/>
									<span>{notifyChannelLabel(ch)}</span>
								</label>
							{/each}
						</div>
					</li>
				{/each}
			</ul>
		{/if}
		{#if notifyError}<p class="error">{notifyError}</p>{/if}
		{#if push.supported}
			<div class="push-test">
				<button class="btn ghost push-test-btn" onclick={sendPushTest} disabled={pushTestBusy}>
					{pushTestBusy
						? language.text('Sender…', 'Sender…', 'Sending…')
						: language.text('Send testvarsel', 'Send testvarsel', 'Send test notification')}
				</button>
				<span class="muted small">
					{language.text(
						'Sjekk at push-varsler når denne enheten.',
						'Sjekk at push-varsel når denne eininga.',
						'Check that push notifications reach this device.'
					)}
				</span>
			</div>
			{#if pushTestResult}
				<p class={pushTestFailed ? 'error' : 'push-test-ok'} role="status">{pushTestResult}</p>
			{/if}
		{/if}
	</section>

	<section class="card danger-zone">
		<h3>{language.text('Slett konto', 'Slett konto', 'Delete account')}</h3>
		<p class="muted small danger-copy">
			{language.text(
				'Dette sletter kontoen din permanent. Kamptips, VM-tips, medlemskap, chataktivitet og private ligaer du eier, blir også fjernet.',
				'Dette slettar kontoen din permanent. Kamptips, VM-tips, medlemskap, chataktivitet og private ligaer du eig, blir også fjerna.',
				'This permanently deletes your account. Tips, World Cup tips, memberships, chat activity and private leagues you own will also be removed.'
			)}
		</p>
		<div class="field">
			<label for="delete-confirm">
				{#if language.isEnglish}
					Type <strong>DELETE</strong> to confirm
				{:else if language.isNynorsk}
					Skriv <strong>DELETE</strong> eller <strong>SLETT</strong> for å stadfeste
				{:else}
					Skriv <strong>DELETE</strong> eller <strong>SLETT</strong> for å bekrefte
				{/if}
			</label>
			<input
				id="delete-confirm"
				class="input"
				bind:value={deletePhrase}
				autocomplete="off"
				spellcheck="false"
				disabled={deleteBusy}
			/>
		</div>
		{#if deleteError}<p class="error">{deleteError}</p>{/if}
		<button
			type="button"
			class="btn danger-btn"
			onclick={destroyAccount}
			disabled={deleteBusy}
		>
			{deleteBusy ? language.text('Sletter...', 'Slettar…', 'Deleting…') : language.text('Slett kontoen min', 'Slett kontoen min', 'Delete my account')}
		</button>
	</section>

	<p class="muted switch"><a href="/">{language.text('Tilbake', 'Tilbake', 'Back')}</a></p>
</div>

<style>
	.settings {
		max-width: 760px;
		margin: 8dvh auto 0;
	}
	.settings > h1,
	.settings > .muted,
	.settings > form,
	.settings > section:not(.player-card),
	.settings > .switch {
		max-width: 520px;
		margin-left: auto;
		margin-right: auto;
	}
	h1 {
		margin: 0;
		font-size: 1.8rem;
	}
	.muted {
		margin: 0.25rem 0 1.5rem;
	}
	.player-card {
		position: relative;
		isolation: isolate;
		overflow: clip;
		max-width: 520px;
		min-height: 590px;
		margin: 1.25rem auto 1.2rem;
		padding: 1.15rem 1.1rem 1.1rem;
		color: #2b210a;
		border: 1px solid color-mix(in srgb, var(--gold) 62%, #fff 20%);
		border-radius: 1.5rem;
		background:
			repeating-linear-gradient(135deg, rgba(255, 255, 255, 0.18) 0 1px, transparent 1px 18px),
			linear-gradient(150deg, rgba(255, 247, 214, 0.96), rgba(217, 187, 114, 0.92) 48%, rgba(142, 101, 30, 0.95)),
			var(--gold);
		box-shadow:
			0 28px 58px -34px rgba(0, 0, 0, 0.64),
			0 0 0 1px rgba(255, 255, 255, 0.16) inset,
			0 2px 0 rgba(255, 255, 255, 0.42) inset;
		transition: transform 0.22s ease, box-shadow 0.22s ease;
	}
	.player-card:hover {
		transform: translateY(-4px) rotateX(1deg);
		box-shadow:
			0 34px 68px -34px rgba(0, 0, 0, 0.74),
			0 0 0 1px rgba(255, 255, 255, 0.24) inset,
			0 2px 0 rgba(255, 255, 255, 0.5) inset;
	}
	.player-card::before {
		content: '';
		position: absolute;
		inset: -28% -70%;
		z-index: 0;
		pointer-events: none;
		background: linear-gradient(105deg, transparent 36%, rgba(255, 255, 255, 0.44) 48%, transparent 60%);
		transform: translateX(-58%) rotate(7deg);
		animation: pc-sheen 7s ease-in-out infinite;
	}
	.player-card::after {
		content: '';
		position: absolute;
		inset: 0.7rem;
		z-index: 0;
		pointer-events: none;
		border: 1px solid rgba(255, 255, 255, 0.36);
		border-radius: 1.1rem;
		background: radial-gradient(circle at 50% 28%, rgba(255, 255, 255, 0.16), transparent 34%);
		mask-image: linear-gradient(to bottom, rgba(0, 0, 0, 0.7), transparent 82%);
		opacity: 0.5;
	}
	.player-card > * {
		position: relative;
		z-index: 1;
	}
	.pc-topline {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 1rem;
	}
	.pc-rating {
		min-width: 6rem;
		text-transform: uppercase;
		letter-spacing: 0;
		line-height: 1;
	}
	.pc-rating-number {
		display: inline-block;
		font-family: var(--font-display);
		font-size: 3.45rem;
		font-weight: 800;
		font-variant-numeric: tabular-nums;
		line-height: 0.82;
		color: #3a2a08;
		text-shadow: 0 1px 0 rgba(255, 255, 255, 0.48);
	}
	.pc-rating-unit {
		display: inline-block;
		font-size: 1.05rem;
		font-weight: 800;
		margin-left: 0.08rem;
	}
	.pc-rating-label,
	.pc-rating-sub,
	.pc-stat-label,
	.pc-stat-sub,
	.pc-miss-kicker,
	.pc-miss-sub,
	.pc-identity p {
		letter-spacing: 0;
	}
	.pc-rating-label {
		margin-top: 0.25rem;
		font-size: 0.73rem;
		font-weight: 800;
	}
	.pc-rating-sub {
		max-width: 6.5rem;
		margin-top: 0.35rem;
		font-size: 0.7rem;
		font-weight: 700;
		line-height: 1.2;
		color: rgba(43, 33, 10, 0.74);
	}
	.pc-crest {
		display: grid;
		place-items: center;
		width: 3.7rem;
		aspect-ratio: 1;
		padding-top: 0.18rem;
		clip-path: polygon(50% 0, 92% 16%, 84% 76%, 50% 100%, 16% 76%, 8% 16%);
		background: linear-gradient(180deg, rgba(28, 74, 52, 0.95), rgba(9, 36, 32, 0.96));
		color: #f7e5aa;
		box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.38) inset;
	}
	.pc-crest span {
		font-family: var(--font-display);
		font-size: 1rem;
		font-weight: 800;
		line-height: 1;
	}
	.pc-crest small {
		font-size: 0.72rem;
		font-weight: 800;
		line-height: 1;
	}
	.pc-portrait {
		position: relative;
		display: grid;
		place-items: center;
		width: 58%;
		min-width: 176px;
		max-width: 232px;
		aspect-ratio: 1;
		margin: -0.85rem auto 0.2rem;
	}
	.pc-portrait::before {
		content: '';
		position: absolute;
		inset: 6%;
		border-radius: 50%;
		background:
			radial-gradient(circle at 50% 38%, rgba(255, 255, 255, 0.5), transparent 34%),
			radial-gradient(circle at 50% 52%, rgba(34, 76, 51, 0.34), transparent 58%);
		opacity: 0.56;
	}
	.pc-avatar-shell {
		position: relative;
		z-index: 1;
		display: grid;
		place-items: center;
		padding: 0.52rem;
		border: 1px solid rgba(69, 48, 11, 0.28);
		border-radius: 50%;
		background: linear-gradient(150deg, rgba(255, 255, 255, 0.78), rgba(255, 234, 164, 0.42));
		box-shadow:
			0 16px 32px -20px rgba(43, 33, 10, 0.78),
			0 0 0 5px rgba(255, 255, 255, 0.18);
	}
	.pc-identity {
		text-align: center;
		padding: 0.65rem 0.75rem 0.75rem;
		border-top: 1px solid rgba(71, 48, 8, 0.32);
		border-bottom: 1px solid rgba(255, 255, 255, 0.42);
	}
	.pc-identity p {
		margin: 0 0 0.2rem;
		font-size: 0.75rem;
		font-weight: 800;
		text-transform: uppercase;
		color: rgba(43, 33, 10, 0.72);
	}
	.pc-identity h2 {
		margin: 0;
		font-size: 1.75rem;
		line-height: 1.05;
		color: #251a05;
		overflow-wrap: anywhere;
		text-shadow: 0 1px 0 rgba(255, 255, 255, 0.42);
	}
	.pc-stars {
		display: flex;
		justify-content: center;
		gap: 0.22rem;
		margin-top: 0.42rem;
	}
	.pc-stars span {
		width: 0.38rem;
		height: 0.38rem;
		clip-path: polygon(50% 0, 62% 35%, 99% 35%, 69% 56%, 80% 92%, 50% 70%, 20% 92%, 31% 56%, 1% 35%, 38% 35%);
		background: #3a2a08;
		opacity: 0.74;
	}
	.pc-loading {
		display: grid;
		gap: 0.5rem;
		margin-top: 1rem;
	}
	.pc-loading span {
		height: 0.75rem;
		border-radius: 999px;
		background: rgba(255, 255, 255, 0.32);
		animation: pc-pulse 1.1s ease-in-out infinite;
	}
	.pc-loading span:nth-child(2) {
		width: 76%;
	}
	.pc-loading span:nth-child(3) {
		width: 58%;
	}
	.pc-empty {
		margin-top: 1rem;
		padding: 0.9rem;
		border: 1px solid rgba(71, 48, 8, 0.22);
		border-radius: 0.8rem;
		background: rgba(255, 255, 255, 0.24);
		font-size: 0.9rem;
		font-weight: 700;
		line-height: 1.35;
		text-align: center;
	}
	.pc-stats {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		margin-top: 0.85rem;
		border: 1px solid rgba(71, 48, 8, 0.26);
		border-radius: 0.9rem;
		background: rgba(255, 255, 255, 0.28);
		overflow: hidden;
	}
	.pc-stat {
		min-width: 0;
		padding: 0.72rem 0.45rem;
		text-align: center;
	}
	.pc-stat + .pc-stat {
		border-left: 1px solid rgba(71, 48, 8, 0.2);
	}
	.pc-stat-value {
		font-family: var(--font-display);
		font-size: 1.45rem;
		font-weight: 800;
		font-variant-numeric: tabular-nums;
		line-height: 1;
		color: #251a05;
	}
	.pc-stat-label {
		margin-top: 0.22rem;
		font-size: 0.72rem;
		font-weight: 800;
		line-height: 1.12;
		overflow-wrap: anywhere;
	}
	.pc-stat-sub {
		margin-top: 0.2rem;
		font-size: 0.66rem;
		font-weight: 700;
		line-height: 1.15;
		color: rgba(43, 33, 10, 0.68);
		overflow-wrap: anywhere;
	}
	.pc-miss-panel {
		margin-top: 0.85rem;
		padding: 0.78rem 0.85rem;
		border: 1px solid rgba(255, 255, 255, 0.34);
		border-radius: 0.9rem;
		background: linear-gradient(135deg, rgba(30, 71, 53, 0.92), rgba(20, 43, 38, 0.96));
		color: #f8ebc5;
		box-shadow: 0 14px 28px -22px rgba(0, 0, 0, 0.88) inset;
	}
	.pc-miss-kicker {
		font-size: 0.7rem;
		font-weight: 800;
		text-transform: uppercase;
		color: rgba(248, 235, 197, 0.7);
	}
	.pc-miss-match {
		margin-top: 0.18rem;
		font-size: 1rem;
		font-weight: 800;
		line-height: 1.2;
		overflow-wrap: anywhere;
	}
	.pc-miss-sub {
		margin-top: 0.3rem;
		font-size: 0.76rem;
		font-weight: 700;
		line-height: 1.25;
		color: rgba(248, 235, 197, 0.76);
	}
	@keyframes pc-sheen {
		0%, 45% { transform: translateX(-58%) rotate(7deg); }
		68%, 100% { transform: translateX(58%) rotate(7deg); }
	}
	@keyframes pc-pulse {
		0%, 100% { opacity: 0.42; }
		50% { opacity: 0.82; }
	}
	@media (max-width: 560px) {
		.player-card {
			max-width: 100%;
			min-height: 0;
			padding: 1rem 0.85rem 0.85rem;
			border-radius: 1.25rem;
		}
		.pc-rating-number {
			font-size: 3rem;
		}
		.pc-portrait {
			min-width: 162px;
			max-width: 214px;
			margin-top: -0.7rem;
		}
		.pc-identity h2 {
			font-size: 1.45rem;
		}
		.pc-stats {
			grid-template-columns: 1fr;
		}
		.pc-stat + .pc-stat {
			border-left: 0;
			border-top: 1px solid rgba(71, 48, 8, 0.2);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.player-card,
		.player-card:hover,
		.player-card::before,
		.pc-loading span {
			animation: none;
			transition: none;
			transform: none;
		}
	}
	.avatar-row {
		display: flex;
		align-items: center;
		gap: 1rem;
		margin-bottom: 1.25rem;
	}
	.hint {
		margin: 0.5rem 0 0;
		font-size: 0.8rem;
	}
	.hidden-file {
		display: none;
	}
	.intro-pref-card {
		display: grid;
		gap: 0.75rem;
	}
	.intro-pref-status,
	.intro-pref-body,
	.intro-pref-ok {
		margin: 0;
	}
	.intro-pref-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.65rem;
		align-items: center;
	}
	.ok {
		color: var(--success);
		font-size: 0.9rem;
	}
	.small {
		font-size: 0.85rem;
		margin: 0.25rem 0 0.9rem;
	}
	.danger-zone {
		border-color: color-mix(in srgb, var(--danger) 30%, var(--border));
		background: color-mix(in srgb, var(--danger) 6%, var(--surface));
	}
	.danger-copy {
		margin-bottom: 1rem;
	}
	.danger-btn {
		background: color-mix(in srgb, var(--danger) 18%, var(--surface-2));
		border-color: color-mix(in srgb, var(--danger) 38%, var(--border));
		color: var(--danger);
	}
	h3 {
		margin: 0 0 0.5rem;
		font-size: 1rem;
	}
	.switch {
		text-align: center;
		margin: 1rem 0 0;
	}
	.notify-list {
		list-style: none;
		margin: 0.5rem 0 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}
	.notify-row {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem 1rem;
		padding-top: 0.75rem;
		border-top: 1px solid var(--border);
	}
	.notify-row:first-child {
		padding-top: 0;
		border-top: none;
	}
	.notify-text {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
		min-width: 12rem;
		flex: 1 1 14rem;
	}
	.notify-name {
		font-weight: 600;
	}
	.notify-toggles {
		display: flex;
		gap: 1rem;
		flex-shrink: 0;
	}
	.notify-toggle {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		cursor: pointer;
		font-size: 0.9rem;
	}
	.notify-toggle.disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
	.push-test {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.5rem 0.85rem;
		margin-top: 1rem;
		padding-top: 0.85rem;
		border-top: 1px solid var(--border);
	}
	.push-test-btn {
		width: auto;
		padding: 0.55rem 0.95rem;
		font-size: 0.85rem;
	}
	.push-test-ok {
		margin: 0.5rem 0 0;
		color: var(--accent);
		font-size: 0.85rem;
		font-weight: 600;
	}
	.notify-toggle input {
		width: 1.05rem;
		height: 1.05rem;
		cursor: pointer;
	}
</style>
