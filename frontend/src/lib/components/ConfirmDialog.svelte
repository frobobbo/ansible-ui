<script lang="ts">
	import { confirmState } from '$lib/toast';

	function respond(value: boolean) {
		if ($confirmState) {
			$confirmState.resolve(value);
			confirmState.set(null);
		}
	}
</script>

{#if $confirmState}
	<div class="overlay" role="presentation" onclick={() => respond(false)}>
		<div class="dialog" role="alertdialog" aria-modal="true" onclick={(e) => e.stopPropagation()}>
			<p class="message">{$confirmState.message}</p>
			<div class="btns">
				<button class="btn btn-secondary" onclick={() => respond(false)}>Cancel</button>
				<button
					class="btn {$confirmState.danger ? 'btn-danger' : 'btn-primary'}"
					onclick={() => respond(true)}
				>
					{$confirmState.confirmText}
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.45);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 9998;
	}
	.dialog {
		background: white;
		border-radius: var(--radius);
		padding: 1.5rem;
		max-width: 380px;
		width: 100%;
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.18);
	}
	.message {
		margin-bottom: 1.25rem;
		font-size: 0.95rem;
		line-height: 1.5;
		color: var(--text);
	}
	.btns {
		display: flex;
		gap: 0.5rem;
		justify-content: flex-end;
	}
</style>
