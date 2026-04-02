<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { auth, ApiError } from '$lib/api';

	let password = $state('');
	let confirm = $state('');
	let error = $state('');
	let loading = $state(false);
	let done = $state(false);

	const token = $derived($page.url.searchParams.get('token') ?? '');

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		if (password !== confirm) {
			error = 'Passwords do not match';
			return;
		}
		if (password.length < 8) {
			error = 'Password must be at least 8 characters';
			return;
		}
		loading = true;
		try {
			await auth.resetPassword(token, password);
			done = true;
			setTimeout(() => goto('/login'), 3000);
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Reset failed';
		} finally {
			loading = false;
		}
	}
</script>

<div class="login-page">
	<div class="login-card">
		<div class="login-brand">
			<img src="/logo.png" alt="Automation Hub" class="brand-logo" />
		</div>
		<div class="login-body">
			<h2>Set New Password</h2>
			{#if !token}
				<div class="alert alert-error">Invalid or missing reset token. Please request a new reset link.</div>
				<p class="back-link"><a href="/forgot-password">Request new link</a></p>
			{:else if done}
				<div class="alert alert-success">Password updated successfully! Redirecting to sign in…</div>
			{:else}
				{#if error}<div class="alert alert-error">{error}</div>{/if}
				<form onsubmit={handleSubmit}>
					<div class="form-group">
						<label for="password">New Password</label>
						<input id="password" class="form-control" type="password" bind:value={password} required minlength="8" autocomplete="new-password" />
					</div>
					<div class="form-group">
						<label for="confirm">Confirm Password</label>
						<input id="confirm" class="form-control" type="password" bind:value={confirm} required autocomplete="new-password" />
					</div>
					<button class="btn btn-primary" type="submit" disabled={loading} style="width:100%">
						{loading ? 'Saving…' : 'Set Password'}
					</button>
				</form>
				<p class="back-link"><a href="/login">← Back to Sign In</a></p>
			{/if}
		</div>
	</div>
</div>

<style>
	.login-page {
		min-height: 100vh;
		display: flex; align-items: center; justify-content: center;
		background: var(--sidebar-bg);
		padding: 1rem;
	}
	.login-card {
		background: white;
		border-radius: 12px;
		width: 100%; max-width: 400px;
		box-shadow: 0 24px 64px rgba(0,0,0,0.4);
		overflow: hidden;
	}
	.login-brand {
		background: #f0f7ff;
		border-bottom: 1px solid #d6eaf8;
		padding: 1.75rem 2rem;
		display: flex; align-items: center; justify-content: center;
	}
	.brand-logo {
		width: 100%; max-width: 300px; height: auto; display: block;
	}
	.login-body { padding: 1.75rem 2rem 2rem; }
	h2 { margin: 0 0 1rem; font-size: 1.2rem; }
	.back-link { margin-top: 1rem; text-align: center; font-size: 0.85rem; }
	.back-link a { color: var(--primary); text-decoration: none; }
	.back-link a:hover { text-decoration: underline; }
	.alert-success { background: #d1fae5; border: 1px solid #6ee7b7; color: #065f46; border-radius: 6px; padding: 0.75rem 1rem; margin-bottom: 1rem; }
</style>
