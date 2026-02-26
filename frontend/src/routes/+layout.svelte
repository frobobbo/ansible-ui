<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { authStore, isAuthenticated, currentUser, isAdmin, isEditor } from '$lib/stores';
	import { auth } from '$lib/api';
	import Toast from '$lib/components/Toast.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

	let { children } = $props();

	$effect(() => {
		const isLoginPage = $page.url.pathname === '/login';
		if (!$isAuthenticated && !isLoginPage) {
			goto('/login');
		}
	});

	async function handleLogout() {
		try { await auth.logout(); } catch {}
		authStore.logout();
		goto('/login');
	}
</script>

<Toast />
<ConfirmDialog />

{#if $page.url.pathname === '/login'}
	{@render children()}
{:else if $isAuthenticated}
	<div class="layout">
		<nav class="sidebar">
			<div class="logo">Ansible UI</div>
			<a href="/" class="nav-link" class:active={$page.url.pathname === '/'}>Dashboard</a>
			{#if $isAdmin}
				<a href="/servers" class="nav-link" class:active={$page.url.pathname.startsWith('/servers')}>Servers</a>
				<a href="/playbooks" class="nav-link" class:active={$page.url.pathname.startsWith('/playbooks')}>Playbooks</a>
			{/if}
			{#if $isEditor}
				<a href="/forms" class="nav-link" class:active={$page.url.pathname.startsWith('/forms')}>Forms</a>
				<a href="/runs" class="nav-link" class:active={$page.url.pathname.startsWith('/runs')}>Run History</a>
			{/if}
			{#if $isAdmin}
				<a href="/vaults" class="nav-link" class:active={$page.url.pathname.startsWith('/vaults')}>Vaults</a>
				<a href="/users" class="nav-link" class:active={$page.url.pathname.startsWith('/users')}>Users</a>
			{/if}
			<div class="sidebar-footer">
				<span class="user-info">{$currentUser?.username} ({$currentUser?.role})</span>
				<button class="btn-logout" onclick={handleLogout}>Logout</button>
				<span class="version-info">v{__APP_VERSION__}</span>
			</div>
		</nav>
		<main class="content">
			{@render children()}
		</main>
	</div>
{/if}

<style>
	.layout { display: flex; min-height: 100vh; }
	.sidebar {
		width: 220px; min-width: 220px; background: var(--sidebar-bg); color: var(--sidebar-text);
		padding: 1.5rem 0; display: flex; flex-direction: column; position: sticky; top: 0; height: 100vh; overflow-y: auto;
	}
	.logo { font-size: 1.1rem; font-weight: 700; color: white; padding: 0 1.25rem 1.5rem; border-bottom: 1px solid #334155; margin-bottom: 0.75rem; }
	.nav-link { display: block; padding: 0.625rem 1.25rem; color: var(--sidebar-text); text-decoration: none; font-size: 0.875rem; border-left: 3px solid transparent; transition: all 0.15s; }
	.nav-link:hover { color: white; background: rgba(255,255,255,0.05); text-decoration: none; }
	:global(.nav-link.active) { color: white; border-left-color: var(--sidebar-active); background: rgba(59,130,246,0.1); }
	.sidebar-footer { margin-top: auto; padding: 1rem 1.25rem; border-top: 1px solid #334155; display: flex; flex-direction: column; gap: 0.5rem; }
	.user-info { font-size: 0.75rem; color: #64748b; }
	.btn-logout { background: none; border: 1px solid #475569; color: var(--sidebar-text); padding: 0.375rem 0.75rem; border-radius: var(--radius); cursor: pointer; font-size: 0.8rem; }
	.btn-logout:hover { background: #ef4444; border-color: #ef4444; color: white; }
	.version-info { font-size: 0.7rem; color: #475569; text-align: center; }
	.content { flex: 1; padding: 2rem; overflow-y: auto; min-height: 100vh; }
</style>
