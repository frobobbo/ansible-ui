<script lang="ts">
	import { toast } from '$lib/toast';
</script>

<div class="toast-container" aria-live="polite">
	{#each $toast as t (t.id)}
		<div class="toast toast-{t.type}" role="alert">
			<span class="toast-msg">{t.message}</span>
			<button class="toast-close" onclick={() => toast.remove(t.id)} aria-label="Dismiss">✕</button>
		</div>
	{/each}
</div>

<style>
	.toast-container {
		position: fixed;
		bottom: 1.5rem;
		right: 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		z-index: 9999;
		max-width: 400px;
	}
	.toast {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.75rem 1rem;
		border-radius: var(--radius);
		font-size: 0.875rem;
		font-weight: 500;
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
		animation: slide-in 0.2s ease;
	}
	@keyframes slide-in {
		from { transform: translateX(110%); opacity: 0; }
		to   { transform: translateX(0);   opacity: 1; }
	}
	.toast-success { background: #dcfce7; color: #15803d; border: 1px solid #bbf7d0; }
	.toast-error   { background: #fee2e2; color: #b91c1c; border: 1px solid #fecaca; }
	.toast-info    { background: #dbeafe; color: #1d4ed8; border: 1px solid #bfdbfe; }
	.toast-msg  { flex: 1; }
	.toast-close {
		background: none;
		border: none;
		cursor: pointer;
		font-size: 0.8rem;
		opacity: 0.6;
		padding: 0;
		color: inherit;
		line-height: 1;
	}
	.toast-close:hover { opacity: 1; }
</style>
