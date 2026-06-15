<script lang="ts">
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { language } from '$lib/language.svelte';
	import {
		ArrowLeft,
		CheckCircle2,
		Clock,
		Info,
		ListChecks,
		Mail,
		Medal,
		Network,
		Telescope,
		Trophy,
		Users,
		Volleyball,
		X
	} from '@lucide/svelte';

	const SUPPORT_EMAIL = 'oyvhov@gmail.com';

	let flow = $derived.by(() => [
		{
			icon: Telescope,
			title: language.text('VM-tips før avspark', 'VM-tips før avspark', 'World Cup tips before kickoff'),
			text: language.text(
				'Sett grupperekkefølge, beste treere, hele sluttspillet og ett toppscorarvalg før første avspark. Du kan også søke opp og legge til en outsider hvis listen er for smal.',
				'Set grupperekkjefølgje, beste trearar, heile sluttspelstreet og eitt toppscorarval før første avspark. Du kan òg søkje opp og leggje til ein outsider dersom shortlist er for smal.',
				'Set the group order, best thirds, full knockout bracket, and one top-scorer pick before the first whistle. You can also search and add an outsider if the shortlist is too narrow.'
			)
		},
		{
			icon: Volleyball,
			title: language.text('Kamptips før hver kamp', 'Kamptips før kvar kamp', 'Match tips before every game'),
			text: language.text(
				'Tipp resultatet for hver kamp. Du kan endre helt fram til avspark.',
				'Tipp resultatet for kvar kamp. Du kan endre heilt fram til avspark.',
				'Pick the score for every match. You can change it right up until kickoff.'
			)
		},
		{
			icon: Clock,
			title: language.text('Tipset låses', 'Tipset låser seg', 'The tip locks'),
			text: language.text(
				'Når kampen starter, låses tipset, og tipsene til venner blir synlige i ligaene.',
				'Når kampen startar, blir tipset låst, og tipsa til vener blir synlege i ligaene.',
				'When the game starts, your tip locks and friends’ tips become visible in leagues.'
			)
		},
		{
			icon: Trophy,
			title: language.text('Poeng underveis', 'Poeng undervegs', 'Points along the way'),
			text: language.text(
				'Resultater, tabeller og poeng oppdateres gjennom gruppespill og sluttspill.',
				'Resultat, tabellar og poeng blir oppdaterte gjennom gruppespel og sluttspel.',
				'Results, tables, and points update continuously through the group stage and knockout rounds.'
			)
		}
	]);

	let matchPoints = $derived.by(() => [
		{ label: language.text('Rett utfall', 'Rett utfall', 'Correct outcome'), value: '3', detail: language.text('1/X/2 i gruppespill, laget som går videre i sluttspill', '1/X/2 i gruppespel, laget som går vidare i sluttspel', '1/X/2 in group stage, the team that advances in knockout') },
		{ label: language.text('Eksakt resultat', 'Eksakt resultat', 'Exact score'), value: '+1', detail: language.text('samme resultat som sluttresultatet', 'same resultat som sluttresultatet', 'same score as the final result') },
		{ label: language.text('Totalt mål', 'Totalt mål', 'Total goals'), value: '+1', detail: language.text('for eksempel teller både 2-1 og 3-0 som 3 mål', 'til dømes tel både 2-1 og 3-0 som 3 mål', 'for example 2-1 and 3-0 both count as 3 goals') },
		{ label: language.text('Rett målforskjell', 'Rett målforskjell', 'Correct goal difference'), value: '+1', detail: language.text('for eksempel ettmålsseier eller uavgjort', 'til dømes eittmålsiger eller uavgjort', 'for example a one-goal win or a draw') }
	]);

	let forecastPoints = $derived.by(() => [
		{ label: language.text('Rett gruppeplassering', 'Rett gruppeplassering', 'Correct group placement'), value: '1' },
		{ label: language.text('Perfekt gruppe', 'Perfekt gruppe', 'Perfect group'), value: '+2' },
		{ label: language.text('Rett lag videre', 'Rett lag vidare', 'Correct team through'), value: '+1' },
		{ label: language.text('R32 / R16 / kvart', '32-del / 16-del / kvart', 'R32 / R16 / QF'), value: '1 / 2 / 3' },
		{ label: language.text('Semi / finale / vinner', 'Semi / finale / vinnar', 'SF / Final / Winner'), value: '5 / 8 / 13' },
		{ label: language.text('Rett toppscorer', 'Rett toppscorar', 'Correct Golden Boot winner'), value: '15' }
	]);

	let appFacts = $derived.by(() => [
		{ icon: Users, title: language.text('Ligaer', 'Ligaer', 'Leagues'), text: language.text('Opprett private ligaer, del invitasjon og følg tabellen sammen.', 'Opprett private ligaer, del invitasjon og følg tabellen saman.', 'Create private leagues, share an invite, and follow the table together.') },
		{ icon: Network, title: language.text('Turnering', 'Turnering', 'Tournament'), text: language.text('Se grupper, kamper og sluttspilltreet mens VM går.', 'Sjå grupper, kampar og sluttspelstreet medan VM går føre seg.', 'See groups, fixtures, and the knockout tree as the World Cup unfolds.') },
		{ icon: ListChecks, title: language.text('Oversikt', 'Oversikt', 'Overview'), text: language.text('Forsiden viser hva som mangler, neste frist og plasseringen din.', 'Framsida viser kva som manglar, neste frist og plasseringa di.', 'The home page shows what is missing, the next deadline, and your standing.') }
	]);

	function closeInfo() {
		if (browser && history.length > 1) {
			history.back();
			return;
		}
		void goto('/');
	}
</script>

<svelte:head>
	<title>{language.text('Info om spillet', 'Info om spelet', 'About the game')} · VM Tipping</title>
</svelte:head>

<div class="info-page">
	<button class="close" type="button" aria-label={language.text('Lukk og gå tilbake', 'Lukk og gå tilbake', 'Close and go back')} onclick={closeInfo}>
		<X size={18} />
		<span>{language.text('Lukk', 'Lukk', 'Close')}</span>
	</button>

	<section class="hero" aria-labelledby="info-title">
		<div class="hero-copy">
			<p class="kicker">Info</p>
			<h1 id="info-title">{language.text('Slik fungerer VM Tipping', 'Slik fungerer VM Tipping', 'How VM Tipping works')}</h1>
			<p class="lead">
				{language.text(
					'Tipp hele VM og én toppscorer før avspark, søk opp og legg til en annen spiller hvis du tror på en outsider, legg inn kamptips før hver kamp, og konkurrer med venner i ligaer gjennom turneringen.',
					'Tipp heile VM og éin toppscorar før avspark, søk opp og legg til ein annan spelar dersom du trur på ein overraskande toppscorar, legg inn kamptips før kvar kamp, og konkurrer med vener i ligaer gjennom turneringa.',
					'Pick the full World Cup and one top scorer before kickoff, search and add another player if you back a breakout scorer, enter match tips before every game, and compete with friends in leagues as the tournament rolls on.'
				)}
			</p>
		</div>
		<div class="scoreboard" aria-label={language.text('Kort oversikt', 'Kort oversikt', 'Quick overview')}>
			<div><strong>104</strong><span>{language.text('kamper', 'kampar', 'matches')}</span></div>
			<div><strong>1</strong><span>{language.text('VM-tips', 'VM-tips', 'World Cup tip')}</span></div>
			<div><strong>1</strong><span>{language.text('toppscorervalg', 'toppscorarval', 'top-scorer pick')}</span></div>
			<div><strong>6</strong><span>{language.text('maks per kamp', 'maks per kamp', 'max per game')}</span></div>
		</div>
	</section>

	<section class="section-block" aria-labelledby="journey-title">
		<div class="section-head">
			<Info size={18} />
			<h2 id="journey-title">{language.text('Slik fungerer det', 'Slik går det føre seg', 'How it flows')}</h2>
		</div>
		<div class="flow-grid">
			{#each flow as step, index}
				{@const Icon = step.icon}
				<article class="card flow-card">
					<div class="step-mark"><span>{index + 1}</span><Icon size={22} /></div>
					<h3>{step.title}</h3>
					<p>{step.text}</p>
				</article>
			{/each}
		</div>
	</section>

	<section class="section-block" aria-labelledby="app-title">
		<div class="section-head">
			<CheckCircle2 size={18} />
			<h2 id="app-title">{language.text('Appen og spillet', 'Appen og spelet', 'The app and the game')}</h2>
		</div>
		<div class="facts-grid">
			{#each appFacts as fact}
				{@const Icon = fact.icon}
				<article class="card fact-card">
					<Icon size={22} />
					<div>
						<h3>{fact.title}</h3>
						<p>{fact.text}</p>
					</div>
				</article>
			{/each}
		</div>
	</section>

	<section class="section-block scoring" aria-labelledby="score-title">
		<div class="section-head">
			<Medal size={18} />
			<h2 id="score-title">{language.text('Poengsystem', 'Poengsystem', 'Scoring system')}</h2>
		</div>

		<div class="score-layout">
			<article class="card score-panel match-panel">
				<div class="panel-title">
					<Volleyball size={20} />
					<h3>{language.text('Kamptips', 'Kamptips', 'Match tips')}</h3>
				</div>
				<p>{language.text('Maks 6 poeng per kamp. I sluttspill teller laget som går videre som rett utfall.', 'Maks 6 poeng per kamp. I sluttspel tel laget som går vidare som rett utfall.', 'Max 6 points per match. In knockout, the advancing team counts as the correct outcome.')}</p>
				<div class="point-list">
					{#each matchPoints as point}
						<div class="point-row">
							<strong>{point.value}</strong>
							<div>
								<span>{point.label}</span>
								<small>{point.detail}</small>
							</div>
						</div>
					{/each}
				</div>
			</article>

			<article class="card score-panel forecast-panel">
				<div class="panel-title">
					<Telescope size={20} />
					<h3>{language.text('VM-tips', 'VM-tips', 'World Cup tips')}</h3>
				</div>
				<p>{language.text('VM-tipset låses ved første kamp og gir poeng etter hvert som grupper, runder, mester og toppscorer blir avgjort.', 'VM-tipset låser seg ved første kamp og gir poeng etter kvart som grupper, rundar, meister og toppscorar blir avgjorde.', 'The World Cup tip locks at the first match and scores as groups, rounds, champion, and Golden Boot are decided.')}</p>
				<div class="forecast-grid">
					{#each forecastPoints as point}
						<div>
							<span>{point.label}</span>
							<strong>{point.value}</strong>
						</div>
					{/each}
				</div>
			</article>
		</div>

		<div class="card tie-break">
			<Medal size={18} />
			<p>
				{language.text(
					'Ved poenglikhet sorteres tabellen etter flest eksakte resultater, flest rette vinnere, lavest målforskjell-feil, færrest leverte tips og tidligste levering.',
					'Ved poenglikskap blir tabellen sortert etter flest eksakte resultat, flest rette vinnarar, lågaste målforskjell-feil, færrast leverte tips og tidlegaste levering.',
					'If points are tied, the table sorts by most exact scores, most correct winners, lowest goal-difference error, fewest submitted tips, and earliest submission.'
				)}
			</p>
		</div>
	</section>

	<section class="section-block" aria-labelledby="support-title">
		<div class="section-head">
			<Mail size={18} />
			<h2 id="support-title">{language.text('Support og feil', 'Support og feil', 'Support and bug reports')}</h2>
		</div>
		<div class="card support-card">
			<p>
				{language.text(
					'Trenger du hjelp eller vil melde inn en feil? Send en e-post til adressen under.',
					'Treng du hjelp eller vil melde inn ein feil? Send ein e-post til adressa under.',
					'Need help or want to report a bug? Send an email to the address below.'
				)}
			</p>
			<a class="support-mail" href={`mailto:${SUPPORT_EMAIL}`}>{SUPPORT_EMAIL}</a>
		</div>
	</section>

	<button class="back-bottom" type="button" onclick={closeInfo}>
		<ArrowLeft size={18} />
		{language.text('Tilbake', 'Tilbake', 'Back')}
	</button>

	<footer class="copyright">
		<p>
			© 2026 VM Tipping · {language.text('Support', 'Support', 'Support')}:
			<a href={`mailto:${SUPPORT_EMAIL}`}>{SUPPORT_EMAIL}</a>
		</p>
	</footer>
</div>

<style>
	.info-page {
		max-width: 1080px;
		margin: 0 auto;
		padding: 0 0 2rem;
	}
	:global(.info-page .card) {
		border-color: color-mix(in srgb, var(--border) 55%, transparent);
	}
	.info-page h1,
	.info-page h2,
	.info-page h3 {
		letter-spacing: 0;
	}
	.close {
		position: sticky;
		top: calc(var(--topbar-h) + 0.75rem);
		z-index: 8;
		margin-left: auto;
		display: flex;
		align-items: center;
		gap: 0.4rem;
		width: fit-content;
		padding: 0.55rem 0.8rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: color-mix(in srgb, var(--surface) 86%, transparent);
		color: var(--text);
		font: inherit;
		font-weight: 800;
		box-shadow: var(--shadow-pop);
		backdrop-filter: blur(14px);
		cursor: pointer;
	}
	.hero {
		display: grid;
		grid-template-columns: minmax(0, 1fr);
		gap: 1rem;
		padding: 1.2rem 0 0.9rem;
	}
	.hero-copy {
		padding: 1rem 0 0;
	}
	.kicker {
		margin: 0 0 0.55rem;
	}
	h1 {
		font-size: 2rem;
		line-height: 1.05;
	}
	.lead {
		max-width: 680px;
		margin: 0.8rem 0 0;
		font-size: 1.02rem;
		line-height: 1.55;
		color: var(--muted);
	}
	.scoreboard {
		display: flex;
		flex-wrap: wrap;
		gap: 0.65rem;
		margin-top: 0.6rem;
	}
	.scoreboard div {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.45rem 1rem;
		background: var(--surface-2);
		border-radius: var(--radius-pill);
	}
	.scoreboard strong {
		font-size: 1.15rem;
		line-height: 1;
		color: var(--text);
	}
	.scoreboard span {
		font-size: 0.85rem;
		font-weight: 700;
		color: var(--muted);
	}
	.section-block {
		margin-top: 1.35rem;
	}
	.section-head {
		display: flex;
		align-items: center;
		gap: 0.55rem;
		margin-bottom: 0.75rem;
		color: var(--text);
	}
	:global(.section-head svg) {
		color: var(--accent-2);
	}
	.section-head h2 {
		font-size: 1.35rem;
	}
	.flow-grid,
	.facts-grid,
	.score-layout {
		display: grid;
		gap: 0.75rem;
	}
	.step-mark {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 1rem;
		color: var(--accent);
	}
	.step-mark span {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		border-radius: 50%;
		background: color-mix(in srgb, var(--accent) 14%, transparent);
		font-weight: 900;
		color: var(--text);
	}
	.flow-card h3,
	.fact-card h3,
	.score-panel h3 {
		font-size: 1rem;
	}
	.flow-card p,
	.fact-card p,
	.score-panel p,
	.tie-break p {
		margin: 0.45rem 0 0;
		line-height: 1.48;
		color: var(--muted);
	}
	.fact-card {
		display: flex;
		gap: 0.75rem;
	}
	:global(.fact-card svg) {
		flex: none;
		color: var(--accent);
	}
	.panel-title {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	:global(.panel-title svg) {
		color: var(--accent-2);
	}
	.point-list {
		display: grid;
		gap: 0.6rem;
		margin-top: 0.9rem;
	}
	.point-row {
		display: grid;
		grid-template-columns: 3.25rem minmax(0, 1fr);
		gap: 0.75rem;
		align-items: center;
		padding: 0.72rem;
		border-radius: var(--radius-sm);
		background: var(--surface-2);
	}
	.point-row strong {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 3.25rem;
		height: 3.25rem;
		border-radius: 50%;
		background: var(--text);
		color: var(--bg);
		font-size: 1.05rem;
	}
	.point-row span,
	.forecast-grid span {
		display: block;
		font-weight: 900;
	}
	.point-row small {
		display: block;
		margin-top: 0.18rem;
		line-height: 1.35;
		color: var(--muted);
	}
	.forecast-grid {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 0.55rem;
		margin-top: 0.9rem;
	}
	.forecast-grid div {
		min-height: 88px;
		padding: 0.75rem;
		border-radius: var(--radius-sm);
		background: var(--surface-2);
	}
	.forecast-grid strong {
		display: block;
		margin-top: 0.5rem;
		font-size: 1.35rem;
		color: var(--accent-2);
	}
	.tie-break {
		display: flex;
		gap: 0.7rem;
		align-items: flex-start;
		margin-top: 0.75rem;
		padding: 0.9rem 1rem;
	}
	:global(.tie-break svg) {
		flex: none;
		margin-top: 0.15rem;
		color: var(--gold);
	}
	.tie-break p {
		margin: 0;
	}
	.support-card {
		display: grid;
		gap: 0.85rem;
		padding: 1rem;
	}
	.support-card p {
		margin: 0;
		line-height: 1.5;
		color: var(--muted);
	}
	.support-mail {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: fit-content;
		max-width: 100%;
		padding: 0.8rem 1rem;
		border: 1px solid color-mix(in srgb, var(--accent) 30%, var(--border));
		border-radius: var(--radius-pill);
		background: color-mix(in srgb, var(--accent) 10%, var(--surface-2));
		color: var(--text);
		font-weight: 900;
		text-decoration: none;
		word-break: break-all;
	}
	.support-mail:hover {
		border-color: color-mix(in srgb, var(--accent) 55%, var(--border));
		background: color-mix(in srgb, var(--accent) 16%, var(--surface-2));
	}
	.back-bottom {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		width: 100%;
		margin-top: 1.35rem;
		padding: 0.85rem 1rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: var(--surface-2);
		color: var(--text);
		font: inherit;
		font-weight: 900;
		cursor: pointer;
	}
	@media (min-width: 560px) {
		.flow-grid {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}
	@media (min-width: 760px) {
		.info-page {
			padding-bottom: 3rem;
		}
		h1 {
			font-size: 2.75rem;
		}
		.hero {
			grid-template-columns: minmax(0, 1.2fr) minmax(320px, 0.8fr);
			align-items: end;
			gap: 1.5rem;
			padding-top: 2rem;
		}
		.facts-grid {
			grid-template-columns: repeat(3, minmax(0, 1fr));
		}
		.score-layout {
			grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
		}
		.back-bottom {
			width: fit-content;
			padding-inline: 1.2rem;
		}
	}
	@media (min-width: 1020px) {
		.flow-grid {
			grid-template-columns: repeat(4, minmax(0, 1fr));
		}
	}
	@media (max-width: 759px) {
		.forecast-grid {
			grid-template-columns: minmax(0, 1fr);
		}
	}
	.copyright {
		margin-top: 2rem;
		text-align: center;
		font-size: 0.82rem;
		color: var(--muted);
	}
	.copyright a {
		color: var(--muted);
		text-decoration: underline;
		text-underline-offset: 3px;
	}
</style>
