<script lang="ts">
	import { auth, ApiError } from '$lib/api';

	let username = $state('');
	let message = $state('');
	let error = $state('');
	let loading = $state(false);
	let sent = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		loading = true;
		try {
			const res = await auth.forgotPassword(username);
			message = res.message;
			sent = true;
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Request failed';
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
			<h2>Forgot Password</h2>
			{#if sent}
				<div class="alert alert-success">{message}</div>
				<p class="back-link"><a href="/login">← Back to Sign In</a></p>
			{:else}
				<p class="hint">Enter your username and we'll send a reset link to your account's email address.</p>
				{#if error}<div class="alert alert-error">{error}</div>{/if}
				<form onsubmit={handleSubmit}>
					<div class="form-group">
						<label for="username">Username</label>
						<input id="username" class="form-control" type="text" bind:value={username} required autocomplete="username" />
					</div>
					<button class="btn btn-primary" type="submit" disabled={loading} style="width:100%">
						{loading ? 'Sending…' : 'Send Reset Link'}
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
	h2 { margin: 0 0 0.75rem; font-size: 1.2rem; }
	.hint { margin: 0 0 1.25rem; font-size: 0.9rem; color: var(--text-muted); }
	.back-link { margin-top: 1rem; text-align: center; font-size: 0.85rem; }
	.back-link a { color: var(--primary); text-decoration: none; }
	.back-link a:hover { text-decoration: underline; }
	.alert-success { background: #d1fae5; border: 1px solid #6ee7b7; color: #065f46; border-radius: 6px; padding: 0.75rem 1rem; margin-bottom: 1rem; }
</style>
