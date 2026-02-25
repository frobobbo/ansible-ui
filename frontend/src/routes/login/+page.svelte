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
		<h1>Ansible UI</h1>
		<p class="subtitle">Sign in to manage your playbooks</p>
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
				{loading ? 'Signing in...' : 'Sign In'}
			</button>
		</form>
	</div>
</div>

<style>
	.login-page { min-height: 100vh; display: flex; align-items: center; justify-content: center; background: var(--sidebar-bg); }
	.login-card { background: white; border-radius: 12px; padding: 2.5rem; width: 100%; max-width: 380px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
	h1 { font-size: 1.75rem; font-weight: 700; text-align: center; margin-bottom: 0.5rem; }
	.subtitle { text-align: center; color: var(--text-muted); margin-bottom: 2rem; font-size: 0.875rem; }
</style>
