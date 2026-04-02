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
			<img src="/logo.png" alt="Automation Hub" class="brand-logo" />
		</div>
		<div class="login-body">
			{#if error}
				<div class="alert alert-error">{error}</div>
			{/if}
			<form onsubmit={handleLogin} autocomplete="off">
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
		width: 100%;
		max-width: 300px;
		height: auto;
		display: block;
	}
	.login-body { padding: 1.75rem 2rem 2rem; }
</style>
