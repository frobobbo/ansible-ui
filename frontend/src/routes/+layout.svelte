<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { authStore, isAuthenticated, currentUser, isAdmin, isEditor } from '$lib/stores';
	import { auth } from '$lib/api';
	import Toast from '$lib/components/Toast.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

	let { children } = $props();

	let sidebarOpen = $state(false);
	let darkMode = $state(typeof localStorage !== 'undefined' && localStorage.getItem('theme') === 'dark');

	$effect(() => {
		document.documentElement.setAttribute('data-theme', darkMode ? 'dark' : 'light');
		localStorage?.setItem('theme', darkMode ? 'dark' : 'light');
	});

	// Close sidebar when navigating (mobile)
	$effect(() => {
		$page.url.pathname;
		sidebarOpen = false;
	});

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

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') sidebarOpen = false; }} />

<Toast />
<ConfirmDialog />

{#if $page.url.pathname === '/login'}
	{@render children()}
{:else if $isAuthenticated}
	<div class="layout">

		<!-- Mobile top bar -->
		<header class="topbar">
			<button
				class="hamburger"
				class:is-open={sidebarOpen}
				onclick={() => (sidebarOpen = !sidebarOpen)}
				aria-label="Toggle navigation"
				aria-expanded={sidebarOpen}
			>
				<span></span>
				<span></span>
				<span></span>
			</button>
			<span class="topbar-brand">Ansible UI</span>
		</header>

		<!-- Backdrop (mobile) -->
		{#if sidebarOpen}
			<div class="backdrop" onclick={() => (sidebarOpen = false)}></div>
		{/if}

		<nav class="sidebar" class:open={sidebarOpen} aria-label="Main navigation">
			<div class="logo">
				<svg class="logo-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
					<polygon points="5,3 19,12 5,21"/>
				</svg>
				Ansible UI
			</div>

			<div class="nav-items">
				<a href="/" class="nav-link" class:active={$page.url.pathname === '/'}>
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<rect x="3" y="3" width="7" height="7" rx="1"/>
						<rect x="14" y="3" width="7" height="7" rx="1"/>
						<rect x="3" y="14" width="7" height="7" rx="1"/>
						<rect x="14" y="14" width="7" height="7" rx="1"/>
					</svg>
					Dashboard
				</a>

				{#if $isAdmin}
					<div class="nav-group">
						<div class="nav-group-label">Infrastructure</div>
						<a href="/servers" class="nav-link" class:active={$page.url.pathname.startsWith('/servers')}>
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<rect x="2" y="4" width="20" height="5" rx="1"/>
								<rect x="2" y="11" width="20" height="5" rx="1"/>
								<rect x="2" y="18" width="20" height="3" rx="1"/>
								<circle cx="6" cy="6.5" r="1" fill="currentColor" stroke="none"/>
								<circle cx="6" cy="13.5" r="1" fill="currentColor" stroke="none"/>
							</svg>
							Servers
						</a>
						<a href="/playbooks" class="nav-link" class:active={$page.url.pathname.startsWith('/playbooks')}>
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
								<polyline points="14,2 14,8 20,8"/>
								<line x1="16" y1="13" x2="8" y2="13"/>
								<line x1="16" y1="17" x2="8" y2="17"/>
							</svg>
							Playbooks
						</a>
						<a href="/vaults" class="nav-link" class:active={$page.url.pathname.startsWith('/vaults')}>
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<rect x="3" y="11" width="18" height="11" rx="2"/>
								<path d="M7 11V7a5 5 0 0 1 10 0v4"/>
								<circle cx="12" cy="16" r="1.5" fill="currentColor" stroke="none"/>
							</svg>
							Vaults
						</a>
					</div>
				{/if}

				{#if $isEditor}
					<div class="nav-group">
						<div class="nav-group-label">Automation</div>
						<a href="/forms" class="nav-link" class:active={$page.url.pathname.startsWith('/forms')}>
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M9 5H7a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2h-2"/>
								<rect x="9" y="3" width="6" height="4" rx="1"/>
								<line x1="9" y1="12" x2="15" y2="12"/>
								<line x1="9" y1="16" x2="13" y2="16"/>
							</svg>
							Forms
						</a>
						<a href="/runs" class="nav-link" class:active={$page.url.pathname.startsWith('/runs')}>
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<polyline points="22,12 18,12 15,21 9,3 6,12 2,12"/>
							</svg>
							Run History
						</a>
					</div>
				{/if}

				{#if $isAdmin}
					<div class="nav-group">
						<div class="nav-group-label">Administration</div>
						<a href="/users" class="nav-link" class:active={$page.url.pathname.startsWith('/users')}>
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
								<circle cx="9" cy="7" r="4"/>
								<path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
								<path d="M16 3.13a4 4 0 0 1 0 7.75"/>
							</svg>
							Users
						</a>
					</div>
				{/if}
			</div>

			<div class="sidebar-footer">
				<div class="user-chip">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
						<circle cx="12" cy="8" r="4"/>
						<path d="M4 20c0-4 3.6-7 8-7s8 3 8 7"/>
					</svg>
					<span class="user-name">{$currentUser?.username}</span>
					<span class="user-role">{$currentUser?.role}</span>
				</div>
				<button class="btn-theme" onclick={() => (darkMode = !darkMode)} aria-label="Toggle dark mode">
					{#if darkMode}
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
							<circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/>
							<line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/>
							<line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/>
							<line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/>
							<line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
						</svg>
						Light mode
					{:else}
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
							<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
						</svg>
						Dark mode
					{/if}
				</button>
				<button class="btn-logout" onclick={handleLogout}>Sign Out</button>
				<span class="version-info">v{__APP_VERSION__}</span>
			</div>
		</nav>

		<main class="content">
			{@render children()}
		</main>
	</div>
{/if}

<style>
	/* ── Layout shell ──────────────────────────────────────────── */
	.layout { display: flex; min-height: 100vh; }

	/* ── Mobile top bar ────────────────────────────────────────── */
	.topbar {
		display: none;
		position: fixed; top: 0; left: 0; right: 0;
		height: 52px; z-index: 150;
		background: var(--sidebar-bg);
		padding: 0 1rem;
		align-items: center;
		gap: 0.75rem;
		border-bottom: 1px solid rgba(255,255,255,0.07);
	}
	.topbar-brand { color: white; font-weight: 700; font-size: 1rem; letter-spacing: -0.01em; }

	/* Hamburger */
	.hamburger {
		display: flex; flex-direction: column; justify-content: center;
		gap: 5px; width: 36px; height: 36px;
		background: none; border: none; cursor: pointer; padding: 6px; flex-shrink: 0;
	}
	.hamburger span {
		display: block; width: 100%; height: 2px;
		background: #cbd5e1; border-radius: 2px;
		transition: transform 0.22s ease, opacity 0.22s ease;
		transform-origin: center;
	}
	.hamburger.is-open span:nth-child(1) { transform: translateY(7px) rotate(45deg); }
	.hamburger.is-open span:nth-child(2) { opacity: 0; transform: scaleX(0); }
	.hamburger.is-open span:nth-child(3) { transform: translateY(-7px) rotate(-45deg); }

	/* Backdrop */
	.backdrop {
		position: fixed; inset: 0;
		background: rgba(0,0,0,0.55);
		z-index: 180;
		backdrop-filter: blur(2px);
	}

	/* ── Sidebar ───────────────────────────────────────────────── */
	.sidebar {
		width: 240px; min-width: 240px;
		background: var(--sidebar-bg);
		color: var(--sidebar-text);
		display: flex; flex-direction: column;
		position: sticky; top: 0;
		height: 100vh; overflow-y: auto;
		flex-shrink: 0;
	}

	.logo {
		display: flex; align-items: center; gap: 0.625rem;
		font-size: 1rem; font-weight: 700; color: white;
		padding: 1.25rem 1.25rem 1rem;
		border-bottom: 1px solid rgba(255,255,255,0.07);
		margin-bottom: 0.5rem;
		letter-spacing: -0.01em;
	}
	.logo-icon { width: 20px; height: 20px; color: var(--primary); flex-shrink: 0; }

	.nav-items { flex: 1; padding: 0.25rem 0; }

	.nav-group { margin-top: 0.5rem; }
	.nav-group-label {
		padding: 0.75rem 1.25rem 0.2rem;
		font-size: 0.65rem; font-weight: 700;
		text-transform: uppercase; letter-spacing: 0.1em;
		color: #4b5563;
	}

	.nav-link {
		display: flex; align-items: center; gap: 0.625rem;
		padding: 0.55rem 1.25rem;
		color: var(--sidebar-text);
		text-decoration: none; font-size: 0.875rem;
		border-left: 3px solid transparent;
		transition: color 0.15s, background 0.15s;
	}
	.nav-link svg { width: 15px; height: 15px; flex-shrink: 0; opacity: 0.7; transition: opacity 0.15s; }
	.nav-link:hover { color: white; background: rgba(255,255,255,0.06); text-decoration: none; }
	.nav-link:hover svg { opacity: 1; }
	:global(.nav-link.active) {
		color: white;
		border-left-color: var(--primary);
		background: rgba(86,54,209,0.15);
	}
	:global(.nav-link.active) svg { opacity: 1; color: #a78bfa; }

	/* ── Sidebar footer ────────────────────────────────────────── */
	.sidebar-footer {
		padding: 0.875rem 1.25rem;
		border-top: 1px solid rgba(255,255,255,0.07);
		display: flex; flex-direction: column; gap: 0.5rem;
	}
	.user-chip {
		display: flex; align-items: center; gap: 0.4rem;
		font-size: 0.75rem; overflow: hidden;
	}
	.user-chip svg { flex-shrink: 0; color: #6b7280; }
	.user-name { font-weight: 500; color: #94a3b8; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.user-role {
		font-size: 0.65rem; background: rgba(86,54,209,0.25);
		color: #a78bfa; padding: 0.1rem 0.4rem; border-radius: 999px; white-space: nowrap;
	}
	.btn-theme {
		display: flex; align-items: center; gap: 0.4rem;
		background: none; border: 1px solid rgba(255,255,255,0.1);
		color: var(--sidebar-text); padding: 0.375rem 0.75rem;
		border-radius: var(--radius); cursor: pointer; font-size: 0.8rem;
		transition: background 0.15s, border-color 0.15s, color 0.15s; width: 100%;
	}
	.btn-theme:hover { background: rgba(86,54,209,0.15); border-color: var(--primary); color: #a78bfa; }
	.btn-logout {
		background: none; border: 1px solid rgba(255,255,255,0.1);
		color: var(--sidebar-text); padding: 0.375rem 0.75rem;
		border-radius: var(--radius); cursor: pointer; font-size: 0.8rem;
		transition: background 0.15s, border-color 0.15s, color 0.15s; text-align: center;
	}
	.btn-logout:hover { background: rgba(226,73,138,0.15); border-color: #e2498a; color: #e2498a; }
	.version-info { font-size: 0.65rem; color: #374151; text-align: center; }

	/* ── Main content ──────────────────────────────────────────── */
	.content { flex: 1; padding: 2rem; overflow-y: auto; min-height: 100vh; min-width: 0; }

	/* ── Responsive ────────────────────────────────────────────── */
	@media (max-width: 768px) {
		.topbar { display: flex; }
		.sidebar {
			position: fixed;
			top: 0; left: 0; bottom: 0;
			z-index: 200;
			width: 260px; min-width: 260px;
			height: 100%;
			transform: translateX(-100%);
			transition: transform 0.25s cubic-bezier(0.4, 0, 0.2, 1);
		}
		.sidebar.open { transform: translateX(0); }
		.content {
			padding: 1.25rem;
			padding-top: calc(52px + 1.25rem);
		}
	}

	@media (max-width: 480px) {
		.content {
			padding: 0.75rem;
			padding-top: calc(52px + 0.75rem);
		}
	}
</style>
