<script lang="ts">
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores';
	import { auth, ApiError } from '$lib/api';

	let username = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);

	async function handleLogin(e: Event) {
		e.preventDefault();
		error = '';
		loading = true;
		try {
			const res = await auth.login(username, password);
			authStore.login(res.token, res.user);
			goto('/');
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Login failed';
		} finally {
			loading = false;
		}
	}
</script>

<div class="login-page">
	<div class="login-card">
		<div class="login-brand">
			<svg class="brand-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
				<polygon points="5,3 19,12 5,21"/>
			</svg>
			<h1>Ansible UI</h1>
			<p class="subtitle">Sign in to manage your playbooks</p>
		</div>
		<div class="login-body">
			{#if error}
				<div class="alert alert-error">{error}</div>
			{/if}
			<form onsubmit={handleLogin}>
				<div class="form-group">
					<label for="username">Username</label>
					<input id="username" class="form-control" type="text" bind:value={username} required autocomplete="username" />
				</div>
				<div class="form-group">
					<label for="password">Password</label>
					<input id="password" class="form-control" type="password" bind:value={password} required autocomplete="current-password" />
				</div>
				<button class="btn btn-primary" type="submit" disabled={loading} style="width:100%">
					{loading ? 'Signing in…' : 'Sign In'}
				</button>
			</form>
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
		width: 100%; max-width: 380px;
		box-shadow: 0 24px 64px rgba(0,0,0,0.4);
		overflow: hidden;
	}
	.login-brand {
		background: linear-gradient(135deg, #5636d1 0%, #e2498a 100%);
		padding: 2rem 2.5rem 1.75rem;
		text-align: center;
		display: flex; flex-direction: column; align-items: center; gap: 0.5rem;
	}
	.brand-icon { width: 36px; height: 36px; color: rgba(255,255,255,0.9); }
	h1 { font-size: 1.75rem; font-weight: 700; color: white; margin: 0; }
	.subtitle { color: rgba(255,255,255,0.75); font-size: 0.875rem; margin: 0; }
	.login-body { padding: 1.75rem 2rem 2rem; }
</style>
