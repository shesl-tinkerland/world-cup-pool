<script lang="ts">
	import { Minus, Plus } from '@lucide/svelte';
	import { language } from '$lib/language.svelte';

	let {
		value = $bindable(0),
		min = 0,
		max = 99,
		disabled = false
	}: { value: number; min?: number; max?: number; disabled?: boolean } =
		$props();

	function bump(d: number) {
		const n = value + d;
		if (n >= min && n <= max) value = n;
	}

	const decrementLabel = $derived(language.text('Trekk fra mål', 'Trekk frå mål', 'Remove goal'));
	const incrementLabel = $derived(language.text('Legg til mål', 'Legg til mål', 'Add goal'));
</script>

<div class="stepper" class:disabled>
	<button type="button" aria-label={decrementLabel} onclick={() => bump(-1)} {disabled}>
		<Minus size={16} />
	</button>
	<span class="val">{value}</span>
	<button type="button" aria-label={incrementLabel} onclick={() => bump(1)} {disabled}>
		<Plus size={16} />
	</button>
</div>

<style>
	.stepper {
		display: inline-flex;
		align-items: center;
		background: var(--bg);
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
	}
	.stepper.disabled {
		opacity: 0.6;
	}
	.stepper button {
		display: grid;
		place-items: center;
		width: 44px;
		height: 44px;
		background: none;
		border: none;
		color: var(--accent);
		cursor: pointer;
	}
	.stepper button:disabled {
		color: var(--muted);
	}
	.val {
		min-width: 1.6rem;
		text-align: center;
		font-weight: 800;
		font-size: 1.05rem;
	}
</style>
