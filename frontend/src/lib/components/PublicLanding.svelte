<script lang="ts">
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth.svelte';
	import { language } from '$lib/language.svelte';
	import Flag from '$lib/components/Flag.svelte';
	import Logo from '$lib/components/Logo.svelte';
	import { fly } from 'svelte/transition';
	import {
		ArrowRight,
		CheckCircle2,
		Clock,
		Mail,
		MessageCircle,
		ShieldCheck,
		Trophy,
		Users,
		Volleyball
	} from '@lucide/svelte';

	const SUPPORT_EMAIL = 'oyvhov@gmail.com';

	let busy = $state(false);
	let error = $state('');
	let targetIndex = $state(0);
	const copy = $derived(
		language.text(
			{
				targets: ['kollegaen', 'kompisen'],
				verb: 'Logg inn og slå',
				description: 'Bli med venner med VM-tips, ligaer, poeng og chat.',
				tagline: 'Kamptips og VM-tips med venner',
				lead: 'Tipp hver kamp, bygg VM-sluttspillet ditt og følg ligadramaet når poengene tikker inn.',
				signInActions: 'Innloggingsvalg',
				openingGoogle: 'Åpner Google...',
				google: 'Logg inn med Google',
				useEmail: 'Bruk e-post',
				createAccount: 'Opprett konto',
				appPreview: 'Appvisning',
				nextDeadline: 'Neste frist',
				norway: 'Norge',
				outcome: 'utfall',
				exact: 'eksakt',
				goals: 'mål',
				diff: 'diff',
				submitTip: 'Lever tips',
				inside: 'Inne i appen',
				showcaseTitle: 'Alt føles som kampdag.',
				groupTable: 'Gruppetabell',
				team: 'Lag',
				pointsSystem: 'Poengsystem',
				maxPerMatch: 'maks per kamp',
				correctOutcome: 'Rett utfall',
				exactScore: 'Eksakt resultat',
				totalGoals: 'Totalt mål',
				goalDiff: 'Målforskjell',
				leagueChat: 'Liga-chat',
				changedEverything: 'Det målet på overtid endret alt.',
				you: 'Du',
				myTip: 'Jeg hadde 2-1. Seks poeng!',
				chatMeta: 'Private ligaer, levende reaksjoner',
				ready: 'Klar før avspark',
				firstPick: 'Legg inn første tips nå.',
				login: 'Logg inn',
				googleFailed: 'Google-innlogging feilet.'
			},
			{
				targets: ['kollegaen', 'kompisen'],
				verb: 'Logg inn og vinn over',
				description: 'Bli med vener med VM-tips, ligaer, poeng og chat.',
				tagline: 'Kamptips og VM-tips med vener',
				lead: 'Tipp kvar kamp, bygg VM-braketten din, og følg ligadramaet når poenga kjem inn.',
				signInActions: 'Innloggingsval',
				openingGoogle: 'Opnar Google...',
				google: 'Logg inn med Google',
				useEmail: 'Bruk e-post',
				createAccount: 'Opprett konto',
				appPreview: 'Appvising',
				nextDeadline: 'Neste frist',
				norway: 'Noreg',
				outcome: 'utfall',
				exact: 'eksakt',
				goals: 'mål',
				diff: 'diff',
				submitTip: 'Lever tips',
				inside: 'Inne i appen',
				showcaseTitle: 'Alt kjennest som kampdag.',
				groupTable: 'Gruppetabell',
				team: 'Lag',
				pointsSystem: 'Poengsystem',
				maxPerMatch: 'maks per kamp',
				correctOutcome: 'Rett utfall',
				exactScore: 'Eksakt resultat',
				totalGoals: 'Totalt mål',
				goalDiff: 'Målforskjell',
				leagueChat: 'Liga-chat',
				changedEverything: 'Det målet på overtid snudde alt.',
				you: 'Du',
				myTip: 'Eg hadde 2-1. Seks poeng!',
				chatMeta: 'Private ligaer, levande reaksjonar',
				ready: 'Klar før avspark',
				firstPick: 'Legg inn første tips no.',
				login: 'Logg inn',
				googleFailed: 'Google-innlogging feila.'
			},
			{
				targets: ['coworker', 'friend'],
				verb: 'Log in and beat your',
				description: 'Join friends for World Cup match tips, leagues, points and chat.',
				tagline: 'World Cup tips with friends',
				lead: 'Tip every match, build your World Cup bracket, and follow the league drama as the points land.',
				signInActions: 'Sign in actions',
				openingGoogle: 'Opening Google...',
				google: 'Log in with Google',
				useEmail: 'Use email',
				createAccount: 'Create account',
				appPreview: 'App preview',
				nextDeadline: 'Next deadline',
				norway: 'Norway',
				outcome: 'outcome',
				exact: 'exact',
				goals: 'goals',
				diff: 'diff',
				submitTip: 'Submit tip',
				inside: 'Inside the app',
				showcaseTitle: 'Everything feels like match day.',
				groupTable: 'Group table',
				team: 'Team',
				pointsSystem: 'Points system',
				maxPerMatch: 'max per match',
				correctOutcome: 'Correct outcome',
				exactScore: 'Exact score',
				totalGoals: 'Total goals',
				goalDiff: 'Goal difference',
				leagueChat: 'League chat',
				changedEverything: 'That 90th minute goal changed everything.',
				you: 'You',
				myTip: 'I had 2-1. Six points!',
				chatMeta: 'Private leagues, live reactions',
				ready: 'Ready before kickoff',
				firstPick: 'Make the first pick now.',
				login: 'Log in',
				googleFailed: 'Google sign-in failed.'
			}
		)
	);
	const landingTargets = $derived(copy.targets);
	const landingVerb = $derived(copy.verb);
	const landingTarget = $derived(
		landingTargets[targetIndex] ?? landingTargets[0] ?? ''
	);

	$effect(() => {
		targetIndex = 0;
	});

	$effect(() => {
		const words = landingTargets;
		if (words.length < 2) return;
		const timer = setInterval(() => {
			targetIndex = (targetIndex + 1) % words.length;
		}, 2200);
		return () => clearInterval(timer);
	});

	async function google() {
		error = '';
		busy = true;
		try {
			await auth.loginGoogle();
			await goto('/');
	} catch (e: unknown) {
			error = (e as { message?: string })?.message ?? copy.googleFailed;
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head>
	<title>VM Tipping</title>
	<meta
		name="description"
		content={copy.description}
	/>
</svelte:head>

<section class="public-landing" aria-labelledby="landing-title">
	<div class="landing-shell">
		<section class="landing-hero">
			<div class="hero-copy">
				<Logo variant="hero" tagline={copy.tagline} />
				<p class="kicker">VM 2026</p>
				<h1 id="landing-title" class="landing-headline">
					<span class="landing-verb">{landingVerb}</span>
					<span class="landing-target-slot">
						{#key `${language.resolved}-${landingTarget}`}
							<span
								class="landing-target"
								in:fly={{ y: 18, duration: 220, opacity: 0.15 }}
								out:fly={{ y: -18, duration: 180, opacity: 0.05 }}
							>
								{landingTarget}
							</span>
						{/key}
					</span>
				</h1>
				<p class="lead">
					{copy.lead}
				</p>

				<div class="hero-actions" aria-label={copy.signInActions}>
					<button type="button" class="google" disabled={busy} onclick={google}>
						<svg class="gsi-logo" viewBox="0 0 48 48" aria-hidden="true">
							<path fill="#EA4335" d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.43 13.72 17.74 9.5 24 9.5z" />
							<path fill="#4285F4" d="M46.98 24.55c0-1.57-.15-3.09-.38-4.55H24v9.02h12.94c-.58 2.96-2.26 5.48-4.78 7.18l7.73 6c4.51-4.18 7.09-10.36 7.09-17.65z" />
							<path fill="#FBBC05" d="M10.53 28.59c-.48-1.45-.76-2.99-.76-4.59s.27-3.14.76-4.59l-7.98-6.19C.92 16.46 0 20.12 0 24c0 3.88.92 7.54 2.56 10.78l7.97-6.19z" />
							<path fill="#34A853" d="M24 48c6.48 0 11.93-2.13 15.89-5.81l-7.73-6c-2.15 1.45-4.92 2.3-8.16 2.3-6.26 0-11.57-4.22-13.47-9.91l-7.98 6.19C6.51 42.62 14.62 48 24 48z" />
						</svg>
						<span>{busy ? copy.openingGoogle : copy.google}</span>
					</button>
					<div class="secondary-actions">
						<a class="btn secondary" href="/login">{copy.useEmail}</a>
						<a class="btn ghost" href="/register">{copy.createAccount}</a>
					</div>
					{#if error}<p class="error">{error}</p>{/if}
				</div>
			</div>

			<div class="hero-board card" aria-label={copy.appPreview}>
				<div class="board-top">
					<span class="live"><Volleyball size={15} /> {copy.nextDeadline}</span>
					<span class="digits">21:00</span>
				</div>
				<div class="match-row">
					<span><Flag iso2="br" code="BRA" size={22} /> Brasil</span>
					<strong class="digits">2-1</strong>
					<span class="right">{copy.norway} <Flag iso2="no" code="NOR" size={22} /></span>
				</div>
				<div class="points-strip">
					<span><b>3</b>{copy.outcome}</span>
					<span><b>+1</b>{copy.exact}</span>
					<span><b>+1</b>{copy.goals}</span>
					<span><b>+1</b>{copy.diff}</span>
				</div>
				<a class="preview-cta" href="/login">
					<CheckCircle2 size={16} /> {copy.submitTip}
				</a>
			</div>
		</section>

		<section class="showcase" aria-labelledby="showcase-title">
			<div class="section-head">
				<p class="kicker">{copy.inside}</p>
				<h2 id="showcase-title">{copy.showcaseTitle}</h2>
			</div>

			<div class="mock-grid">
				<article class="card mock-card table-card">
					<div class="mock-head"><Trophy size={18} /><h3>{copy.groupTable}</h3></div>
					<table>
						<thead><tr><th>#</th><th>{copy.team}</th><th>P</th><th>GD</th><th>Pts</th></tr></thead>
						<tbody>
							<tr><td>1</td><td><Flag iso2="no" code="NOR" size={17} /> {copy.norway}</td><td>3</td><td>+4</td><td>7</td></tr>
							<tr><td>2</td><td><Flag iso2="de" code="GER" size={17} /> Tyskland</td><td>3</td><td>+2</td><td>6</td></tr>
							<tr><td>3</td><td><Flag iso2="mx" code="MEX" size={17} /> Mexico</td><td>3</td><td>0</td><td>4</td></tr>
							<tr><td>4</td><td><Flag iso2="jp" code="JPN" size={17} /> Japan</td><td>3</td><td>-6</td><td>0</td></tr>
						</tbody>
					</table>
				</article>

				<article class="card mock-card score-card">
					<div class="mock-head"><ShieldCheck size={18} /><h3>{copy.pointsSystem}</h3></div>
					<div class="score-total"><strong class="digits">6</strong><span>{copy.maxPerMatch}</span></div>
					<ul>
						<li><span>{copy.correctOutcome}</span><b>3 p</b></li>
						<li><span>{copy.exactScore}</span><b>+1 p</b></li>
						<li><span>{copy.totalGoals}</span><b>+1 p</b></li>
						<li><span>{copy.goalDiff}</span><b>+1 p</b></li>
					</ul>
				</article>

				<article class="card mock-card chat-card">
					<div class="mock-head"><MessageCircle size={18} /><h3>{copy.leagueChat}</h3></div>
					<div class="bubble theirs"><b>Anna</b><span>{copy.changedEverything}</span></div>
					<div class="bubble mine"><b>{copy.you}</b><span>{copy.myTip}</span></div>
					<div class="chat-meta"><Users size={15} /> {copy.chatMeta}</div>
				</article>
			</div>
		</section>

		<section class="bottom-cta card">
			<div>
				<p class="kicker">{copy.ready}</p>
				<h2>{copy.firstPick}</h2>
			</div>
			<a class="btn" href="/login"><Clock size={16} /> {copy.login} <ArrowRight size={16} /></a>
		</section>

		<footer class="landing-support">
			<p>{language.text('Support eller feil? Send e-post til', 'Support eller feil? Send e-post til', 'Need support or want to report a bug? Email')}</p>
			<a class="support-link" href={`mailto:${SUPPORT_EMAIL}`}><Mail size={16} /> {SUPPORT_EMAIL}</a>
		</footer>
	</div>
</section>

<style>
	:global(.app-shell.public-shell) {
		max-width: none;
		padding-top: 0;
		padding-inline: 0;
		padding-bottom: 0;
	}
	.public-landing {
		--accent: #8fc58f;
		--accent-2: #d8b86c;
		--bg: #071019;
		--surface: #0b171f;
		--surface-2: #10242b;
		--surface-3: #18343a;
		--border: rgba(214, 190, 128, 0.12);
		--border-strong: rgba(214, 190, 128, 0.22);
		--text: #f3f6ee;
		--muted: #a7b7ae;
		--gold: #d9bb72;
		min-height: 100dvh;
		overflow-x: clip;
		color: var(--text);
		background:
			radial-gradient(80% 42% at 50% 0%, rgba(217, 187, 114, 0.12), transparent 62%),
			radial-gradient(52% 32% at 80% 12%, rgba(143, 197, 143, 0.13), transparent 70%),
			linear-gradient(180deg, #081824 0%, #071019 52%, #050a0f 100%);
	}
	.landing-shell {
		width: 100%;
		max-width: 1120px;
		margin: 0 auto;
		padding: clamp(1rem, 4vw, 2rem);
	}
	.landing-hero {
		display: grid;
		gap: 1rem;
		padding: min(8dvh, 4rem) 0 1.5rem;
		min-width: 0;
	}
	.hero-copy {
		display: grid;
		gap: 0.8rem;
		min-width: 0;
	}
	h1 {
		max-width: 10ch;
		font-size: clamp(2.45rem, 15vw, 5.8rem);
		line-height: 0.92;
		letter-spacing: 0;
	}
	.landing-headline {
		display: grid;
		gap: 0.08em;
		max-width: 11ch;
		min-width: 0;
	}
	.landing-verb {
		color: color-mix(in srgb, var(--text) 84%, var(--muted));
		font-weight: 500;
		letter-spacing: -0.02em;
	}
	.landing-target-slot {
		display: grid;
		min-height: 1.02em;
		min-width: 0;
		width: 100%;
	}
	.landing-target {
		grid-area: 1 / 1;
		font-weight: 800;
		background: linear-gradient(110deg, #f3dfae 8%, var(--gold) 44%, var(--accent) 100%);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
		background-clip: text;
		will-change: transform, opacity;
	}
	.lead {
		max-width: 52ch;
		min-width: 0;
		margin: 0;
		color: var(--muted);
		font-size: 1.03rem;
		line-height: 1.55;
	}
	.hero-actions {
		display: grid;
		gap: 0.75rem;
		max-width: 430px;
		min-width: 0;
		margin-top: 0.35rem;
	}
	.google {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 12px;
		width: 100%;
		min-height: 48px;
		padding: 0 16px;
		background: #ffffff;
		border: 1px solid #747775;
		border-radius: 6px;
		color: #1f1f1f;
		font-family: 'Roboto', arial, sans-serif;
		font-size: 15px;
		font-weight: 600;
		cursor: pointer;
	}
	.google:disabled {
		opacity: 0.6;
		cursor: wait;
	}
	.gsi-logo {
		width: 20px;
		height: 20px;
		flex: none;
	}
	.secondary-actions {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.55rem;
	}
	.secondary-actions .btn {
		min-height: 44px;
		padding: 0.75rem 0.8rem;
	}
	.hero-board,
	.bottom-cta,
	.mock-card {
		background:
			radial-gradient(circle at 18% 0%, rgba(217, 187, 114, 0.08), transparent 32%),
			linear-gradient(180deg, rgba(13, 34, 40, 0.95), rgba(8, 21, 30, 0.98));
		border-color: var(--border-strong);
	}
	.landing-support {
		display: grid;
		justify-items: center;
		gap: 0.7rem;
		padding: 1.1rem 0 2rem;
		text-align: center;
		color: var(--muted);
	}
	.landing-support p {
		margin: 0;
		font-size: 0.92rem;
	}
	.support-link {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.8rem 1rem;
		border: 1px solid var(--border-strong);
		border-radius: var(--radius-pill);
		background: rgba(255, 255, 255, 0.04);
		color: var(--text);
		font-weight: 800;
		text-decoration: none;
		word-break: break-all;
	}
	.support-link:hover {
		border-color: color-mix(in srgb, var(--gold) 45%, var(--border-strong));
		background: rgba(255, 255, 255, 0.07);
	}
	.hero-board {
		display: grid;
		gap: 0.85rem;
		padding: 1rem;
		width: min(100%, 420px);
		min-width: 0;
	}
	.board-top,
	.match-row,
	.mock-head,
	.bottom-cta {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
		min-width: 0;
		flex-wrap: wrap;
	}
	.live,
	.chat-meta {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		min-width: 0;
		flex-wrap: wrap;
		color: var(--accent);
		font-size: 0.8rem;
		font-weight: 800;
	}
	.match-row {
		padding: 0.85rem;
		border-radius: var(--radius-sm);
		background: rgba(255, 255, 255, 0.035);
	}
	.match-row span {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		font-weight: 800;
		min-width: 0;
		flex-wrap: wrap;
	}
	.match-row .right {
		justify-content: flex-end;
	}
	.match-row strong {
		padding: 0.35rem 0.65rem;
		border-radius: var(--radius-pill);
		background: #050a0f;
		color: var(--gold);
	}
	.points-strip {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 0.45rem;
	}
	.points-strip span {
		display: grid;
		gap: 0.1rem;
		padding: 0.62rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		min-width: 0;
		color: var(--muted);
		font-size: 0.72rem;
	}
	.points-strip b {
		color: var(--text);
		font-family: var(--font-mono);
		font-size: 1rem;
	}
	.preview-cta {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.45rem;
		min-height: 44px;
		border-radius: var(--radius-pill);
		background: var(--accent-2);
		color: #071019;
		font-weight: 900;
	}
	.showcase {
		display: grid;
		gap: 1rem;
		padding: 1.2rem 0 2rem;
		min-width: 0;
	}
	.section-head {
		display: grid;
		gap: 0.25rem;
		min-width: 0;
	}
	.section-head h2,
	.bottom-cta h2 {
		font-size: clamp(1.4rem, 7vw, 2.15rem);
		letter-spacing: 0;
	}
	.mock-grid {
		display: grid;
		gap: 0.85rem;
		min-width: 0;
	}
	.mock-card {
		display: grid;
		gap: 0.9rem;
		min-width: 0;
	}
	.mock-head {
		justify-content: flex-start;
		color: var(--gold);
	}
	.mock-head h3 {
		font-size: 1.05rem;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.86rem;
	}
	th,
	td {
		padding: 0.48rem 0.2rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.07);
		text-align: right;
		overflow-wrap: anywhere;
	}
	th:nth-child(2),
	td:nth-child(2) {
		text-align: left;
	}
	.table-card table {
		table-layout: auto;
	}
	.table-card th,
	.table-card td {
		white-space: nowrap;
		overflow-wrap: normal;
	}
	.table-card th:nth-child(1),
	.table-card td:nth-child(1),
	.table-card th:nth-child(3),
	.table-card td:nth-child(3),
	.table-card th:nth-child(4),
	.table-card td:nth-child(4),
	.table-card th:nth-child(5),
	.table-card td:nth-child(5) {
		width: 2.4rem;
	}
	td:nth-child(2) {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		font-weight: 800;
	}
	.score-total {
		display: flex;
		align-items: baseline;
		gap: 0.55rem;
	}
	.score-total strong {
		font-size: 3rem;
		color: var(--accent);
	}
	.score-card ul {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		gap: 0.45rem;
	}
	.score-card li {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
		min-width: 0;
		flex-wrap: wrap;
		padding: 0.55rem 0;
		border-bottom: 1px solid rgba(255, 255, 255, 0.07);
	}
	.score-card b {
		color: var(--accent);
		font-family: var(--font-mono);
	}
	.bubble {
		display: grid;
		gap: 0.2rem;
		max-width: 82%;
		padding: 0.75rem;
		border-radius: var(--radius-sm);
		background: rgba(255, 255, 255, 0.07);
	}
	.bubble.mine {
		justify-self: end;
		background: color-mix(in srgb, var(--accent) 18%, transparent);
	}
	.bubble b {
		font-size: 0.8rem;
		color: var(--gold);
	}
	.chat-meta {
		color: var(--muted);
	}
	.bottom-cta {
		align-items: stretch;
		margin-bottom: 2rem;
		min-width: 0;
	}
	.bottom-cta .btn {
		width: auto;
		min-width: 190px;
	}

	@media (min-width: 780px) {
		.hero-board {
			justify-self: stretch;
		}
		.mock-grid {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}

	@media (min-width: 980px) {
		.landing-hero {
			grid-template-columns: minmax(0, 1.08fr) minmax(340px, 0.72fr);
			align-items: center;
			min-height: min(760px, 86dvh);
			gap: clamp(1.5rem, 3vw, 2.75rem);
		}
		.hero-board {
			justify-self: end;
		}
		.mock-grid {
			grid-template-columns: 1.05fr 0.85fr 1fr;
		}
	}

	@media (min-width: 521px) and (max-width: 1099px) {
		h1 {
			font-size: clamp(2.15rem, 8.6vw, 4rem);
			max-width: 10.8ch;
		}
		.lead {
			font-size: 0.98rem;
			max-width: 46ch;
		}
		.section-head h2,
		.bottom-cta h2 {
			font-size: clamp(1.25rem, 4vw, 1.8rem);
		}
		.mock-head h3 {
			font-size: 1rem;
		}
		table {
			font-size: 0.82rem;
		}
	}

	@media (min-width: 1100px) {
		h1 {
			font-size: clamp(2.85rem, 5vw, 4.95rem);
			max-width: 11.5ch;
		}
		.lead {
			font-size: 0.98rem;
			max-width: 48ch;
		}
		.section-head h2,
		.bottom-cta h2 {
			font-size: clamp(1.3rem, 2.4vw, 1.95rem);
		}
		.mock-head h3 {
			font-size: 1rem;
		}
		table {
			font-size: 0.82rem;
		}
	}

	@media (max-width: 520px) {
		.landing-shell {
			padding-top: 4.6rem;
		}
		h1 {
			font-size: clamp(2rem, 11vw, 3.1rem);
		}
		.landing-headline,
		.hero-board,
		.bottom-cta > div {
			max-width: 100%;
			min-width: 0;
		}
		.secondary-actions,
		.points-strip {
			grid-template-columns: 1fr;
		}
		.match-row {
			grid-template-columns: 1fr;
			align-items: stretch;
		}
		.match-row,
		.bottom-cta {
			flex-direction: column;
		}
		.bottom-cta .btn {
			width: 100%;
		}
	}
</style>
