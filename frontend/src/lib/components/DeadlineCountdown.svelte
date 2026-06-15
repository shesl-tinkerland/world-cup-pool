<script lang="ts">
	import { onDestroy } from 'svelte';
	import { language } from '$lib/language.svelte';
	import { serverClock } from '$lib/serverclock.svelte';

	let {
		deadline = '',
		label = '',
		compact = false
	}: {
		deadline?: string;
		label?: string;
		compact?: boolean;
	} = $props();

	let now = $state(serverClock.now());
	let timer: ReturnType<typeof setTimeout> | null = null;

	function clearTimer() {
		if (timer !== null) {
			clearTimeout(timer);
			timer = null;
		}
	}

	function nextDelay(msRemaining: number) {
		if (msRemaining <= 60_000) return 1_000;
		if (msRemaining <= 3_600_000) return 15_000;
		return 60_000;
	}

	function formatRemaining(ms: number) {
		const totalSeconds = Math.max(0, Math.floor(ms / 1000));
		const days = Math.floor(totalSeconds / 86_400);
		const hours = Math.floor((totalSeconds % 86_400) / 3_600);
		const minutes = Math.floor((totalSeconds % 3_600) / 60);
		const seconds = totalSeconds % 60;
		const hourUnit = language.text('t', 't', 'h');

		if (days > 0) {
			return hours > 0 ? `${days} d ${hours} ${hourUnit}` : `${days} d`;
		}
		if (hours > 0) {
			return minutes > 0 ? `${hours} ${hourUnit} ${minutes} m` : `${hours} ${hourUnit}`;
		}
		if (minutes > 0) {
			return `${minutes} m`;
		}
		return `${Math.max(1, seconds)} s`;
	}

	let deadlineMs = $derived(deadline ? new Date(deadline).getTime() : NaN);
	let remainingMs = $derived(Number.isFinite(deadlineMs) ? deadlineMs - now : -1);
	let expired = $derived(!Number.isFinite(deadlineMs) || remainingMs <= 0);
	let urgency = $derived(
		remainingMs <= 600_000 ? 'critical' : remainingMs <= 3_600_000 ? 'warn' : 'normal'
	);
	let remainingLabel = $derived(expired ? '' : formatRemaining(remainingMs));
	let relationLabel = $derived(language.text('om', 'om', 'in'));

	$effect(() => {
		now = serverClock.now();
		clearTimer();
		if (!Number.isFinite(deadlineMs) || deadlineMs <= now) {
			return;
		}

		let cancelled = false;
		const tick = () => {
			if (cancelled) return;
			now = serverClock.now();
			const msRemaining = deadlineMs - now;
			if (msRemaining <= 0) {
				clearTimer();
				return;
			}
			timer = setTimeout(tick, nextDelay(msRemaining));
		};

		tick();
		return () => {
			cancelled = true;
			clearTimer();
		};
	});

	onDestroy(clearTimer);
</script>

{#if !expired}
	<div
		class="deadline-countdown"
		class:compact
		class:warn={urgency === 'warn'}
		class:critical={urgency === 'critical'}
	>
		{#if label}
			<span class="countdown-label">{label}</span>
		{/if}
		<span class="countdown-joiner">{relationLabel}</span>
		<span class="countdown-value">{remainingLabel}</span>
	</div>
{/if}

<style>
	.deadline-countdown {
		display: inline-flex;
		align-items: baseline;
		flex-wrap: wrap;
		gap: 0.35rem;
		margin-top: 0.75rem;
		font-size: 0.88rem;
		font-weight: 600;
		color: var(--muted);
	}
	.deadline-countdown.compact {
		margin-top: 0.55rem;
		font-size: 0.82rem;
	}
	.countdown-label {
		text-transform: uppercase;
		letter-spacing: 0.08em;
		font-size: 0.7rem;
		font-weight: 700;
		color: var(--muted);
	}
	.deadline-countdown.compact .countdown-label {
		font-size: 0.69rem;
	}
	.countdown-joiner {
		white-space: nowrap;
	}
	.countdown-value {
		display: inline-flex;
		align-items: center;
		white-space: nowrap;
		color: var(--text);
		font-family: var(--font-mono);
		font-size: 0.98rem;
		font-weight: 700;
	}
	.deadline-countdown.compact .countdown-value {
		font-size: 0.9rem;
	}
	.deadline-countdown.warn .countdown-value {
		color: var(--warning);
	}
	.deadline-countdown.critical .countdown-value {
		color: var(--danger);
	}
</style>
