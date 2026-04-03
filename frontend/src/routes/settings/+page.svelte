<script lang="ts">
	import { onMount } from 'svelte';
	import { settings as settingsApi, ApiError } from '$lib/api';
	import { currentUser } from '$lib/stores';
	import { toast } from '$lib/toast';
	import type { AppSettings, EmailSettings, GitHubSettings } from '$lib/types';

	let loading = $state(true);
	let savingApp = $state(false);
	let saving = $state(false);
	let savingGitHub = $state(false);
	let testing = $state(false);
	let testEmail = $state('');

	let form = $state<EmailSettings>({
		email_provider: '',
		smtp_host: '',
		smtp_port: '587',
		smtp_username: '',
		smtp_password: '',
		smtp_from: '',
		mailgun_api_key: '',
		mailgun_domain: '',
		mailgun_from: '',
		mailgun_region: 'us',
	});

	let app = $state<AppSettings>({ app_url: '' });

	let github = $state<GitHubSettings>({
		github_token: '',
		github_repo: '',
		github_branch: '',
	});

	onMount(async () => {
		try {
			const [appData, emailData, githubData] = await Promise.all([
				settingsApi.getApp(),
				settingsApi.getEmail(),
				settingsApi.getGitHub(),
			]);
			app = { app_url: appData.app_url || '' };
			form = {
				email_provider: emailData.email_provider || '',
				smtp_host: emailData.smtp_host || '',
				smtp_port: emailData.smtp_port || '587',
				smtp_username: emailData.smtp_username || '',
				smtp_password: emailData.smtp_password || '',
				smtp_from: emailData.smtp_from || '',
				mailgun_api_key: emailData.mailgun_api_key || '',
				mailgun_domain: emailData.mailgun_domain || '',
				mailgun_from: emailData.mailgun_from || '',
				mailgun_region: emailData.mailgun_region || 'us',
			};
			github = {
				github_token: githubData.github_token || '',
				github_repo: githubData.github_repo || '',
				github_branch: githubData.github_branch || '',
			};
			testEmail = $currentUser?.email || '';
		} finally {
			loading = false;
		}
	});

	async function save() {
		saving = true;
		try {
			await settingsApi.updateEmail(form);
			toast.success('Email settings saved');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Save failed');
		} finally {
			saving = false;
		}
	}

	async function saveApp() {
		savingApp = true;
		try {
			await settingsApi.updateApp(app);
			toast.success('Application settings saved');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Save failed');
		} finally {
			savingApp = false;
		}
	}

	async function saveGitHub() {
		savingGitHub = true;
		try {
			await settingsApi.updateGitHub(github);
			toast.success('GitHub settings saved');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Save failed');
		} finally {
			savingGitHub = false;
		}
	}

	async function sendTest() {
		if (!testEmail) { toast.error('Enter a test recipient email'); return; }
		testing = true;
		try {
			const res = await settingsApi.testEmail(testEmail, form);
			toast.success(res.message);
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Test failed');
		} finally {
			testing = false;
		}
	}
</script>

<div class="page-header">
	<h1>Settings</h1>
</div>

{#if loading}
	<p class="empty-state">Loading…</p>
{:else}
	<!-- ── Application Section ──────────────────────────────────────── -->
	<form onsubmit={(e) => { e.preventDefault(); saveApp(); }}>
		<div class="section-header">
			<h2>Application</h2>
			<p class="section-hint">General application settings.</p>
		</div>
		<div class="card">
			<div class="form-group">
				<label for="app_url">Base URL</label>
				<input id="app_url" class="form-control" type="url" bind:value={app.app_url} placeholder="https://ansible.johnsons.casa" />
				<span class="form-hint">
					Used to build links in password reset emails. Leave blank to auto-detect from the incoming request.
					Set this if the app is behind a reverse proxy and auto-detection produces the wrong URL.
				</span>
			</div>
		</div>
		<div class="form-actions">
			<button type="submit" class="btn btn-primary" disabled={savingApp}>
				{savingApp ? 'Saving…' : 'Save Application Settings'}
			</button>
		</div>
	</form>

	<form onsubmit={(e) => { e.preventDefault(); save(); }} style="margin-top:2rem">

		<!-- ── Email Section ─────────────────────────────────────────── -->
		<div class="section-header">
			<h2>Email</h2>
			<p class="section-hint">Used for password reset emails and run notifications.</p>
		</div>

		<div class="card">
			<div class="form-group">
				<label>Provider</label>
				<div class="provider-tabs">
					<button type="button" class="provider-tab" class:active={form.email_provider === ''} onclick={() => form.email_provider = ''}>
						None
					</button>
					<button type="button" class="provider-tab" class:active={form.email_provider === 'smtp'} onclick={() => form.email_provider = 'smtp'}>
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
							<rect x="2" y="4" width="20" height="16" rx="2"/>
							<polyline points="2,4 12,13 22,4"/>
						</svg>
						SMTP
					</button>
					<button type="button" class="provider-tab" class:active={form.email_provider === 'mailgun'} onclick={() => form.email_provider = 'mailgun'}>
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
							<path d="M22 12a10 10 0 1 1-5.93-9.14"/>
							<path d="M22 4L12 14.01l-3-3"/>
						</svg>
						Mailgun
					</button>
				</div>
			</div>

			{#if form.email_provider === 'smtp'}
				<div class="provider-fields">
					<div class="form-row">
						<div class="form-group">
							<label for="smtp_host">SMTP Host</label>
							<input id="smtp_host" class="form-control" type="text" bind:value={form.smtp_host} placeholder="smtp.example.com" />
						</div>
						<div class="form-group" style="max-width:120px">
							<label for="smtp_port">Port</label>
							<input id="smtp_port" class="form-control" type="text" bind:value={form.smtp_port} placeholder="587" />
						</div>
					</div>
					<div class="form-row">
						<div class="form-group">
							<label for="smtp_username">Username</label>
							<input id="smtp_username" class="form-control" type="text" bind:value={form.smtp_username} placeholder="user@example.com" autocomplete="off" />
						</div>
						<div class="form-group">
							<label for="smtp_password">Password</label>
							<input id="smtp_password" class="form-control" type="password" bind:value={form.smtp_password} placeholder="••••••••" autocomplete="new-password" />
						</div>
					</div>
					<div class="form-group">
						<label for="smtp_from">From Address</label>
						<input id="smtp_from" class="form-control" type="text" bind:value={form.smtp_from} placeholder="Automation Hub <noreply@example.com>" />
						<span class="form-hint">Leave blank to use the username above.</span>
					</div>
				</div>
			{/if}

			{#if form.email_provider === 'mailgun'}
				<div class="provider-fields">
					<div class="form-group">
						<label for="mg_api_key">API Key</label>
						<input id="mg_api_key" class="form-control" type="password" bind:value={form.mailgun_api_key} placeholder="key-••••••••••••••••••••••••••••••••" autocomplete="new-password" />
						<span class="form-hint">
							Find your API key in the <strong>Mailgun dashboard → API Keys</strong>. Use the private key, not the public validation key.
						</span>
					</div>
					<div class="form-row">
						<div class="form-group">
							<label for="mg_domain">Sending Domain</label>
							<input id="mg_domain" class="form-control" type="text" bind:value={form.mailgun_domain} placeholder="mg.example.com" />
						</div>
						<div class="form-group" style="max-width:140px">
							<label for="mg_region">Region</label>
							<select id="mg_region" class="form-control" bind:value={form.mailgun_region}>
								<option value="us">US (api.mailgun.net)</option>
								<option value="eu">EU (api.eu.mailgun.net)</option>
							</select>
						</div>
					</div>
					<div class="form-group">
						<label for="mg_from">From Address</label>
						<input id="mg_from" class="form-control" type="text" bind:value={form.mailgun_from} placeholder="Automation Hub <noreply@mg.example.com>" />
						<span class="form-hint">Leave blank to auto-generate from your sending domain.</span>
					</div>
				</div>
			{/if}

			{#if form.email_provider === ''}
				<p class="provider-none">Email notifications and password resets are disabled.</p>
			{/if}
		</div>

		{#if form.email_provider !== ''}
			<!-- ── Test Email ─────────────────────────────────────────────── -->
			<div class="section-header" style="margin-top:1.5rem">
				<h2>Test Email</h2>
				<p class="section-hint">Send a test message using the configuration above (changes don't need to be saved first).</p>
			</div>
			<div class="card">
				<div class="test-row">
					<div class="form-group" style="flex:1;margin:0">
						<label for="test_email">Recipient</label>
						<input id="test_email" class="form-control" type="email" bind:value={testEmail} placeholder="you@example.com" />
					</div>
					<button type="button" class="btn btn-secondary" onclick={sendTest} disabled={testing} style="align-self:flex-end">
						{testing ? 'Sending…' : 'Send Test'}
					</button>
				</div>
			</div>
		{/if}

		<div class="form-actions">
			<button type="submit" class="btn btn-primary" disabled={saving}>
				{saving ? 'Saving…' : 'Save Email Settings'}
			</button>
		</div>
	</form>

	<!-- ── GitHub Section ──────────────────────────────────────────── -->
	<form onsubmit={(e) => { e.preventDefault(); saveGitHub(); }} style="margin-top:2rem">
		<div class="section-header">
			<h2>GitHub</h2>
			<p class="section-hint">Used by the EE Editor to read and commit execution environment definition files.</p>
		</div>
		<div class="card">
			<div class="form-group">
				<label for="gh_token">Personal Access Token</label>
				<input id="gh_token" class="form-control" type="password" bind:value={github.github_token} placeholder="github_pat_••••••••••••" autocomplete="new-password" />
				<span class="form-hint">
					Needs <strong>Contents: Read &amp; Write</strong> scope on the target repository.
					Create one at <strong>GitHub → Settings → Developer settings → Personal access tokens</strong>.
				</span>
			</div>
			<div class="form-row">
				<div class="form-group">
					<label for="gh_repo">Repository</label>
					<input id="gh_repo" class="form-control" type="text" bind:value={github.github_repo} placeholder="owner/repo" />
				</div>
				<div class="form-group" style="max-width:160px">
					<label for="gh_branch">Branch</label>
					<input id="gh_branch" class="form-control" type="text" bind:value={github.github_branch} placeholder="main" />
				</div>
			</div>
		</div>
		<div class="form-actions">
			<button type="submit" class="btn btn-primary" disabled={savingGitHub}>
				{savingGitHub ? 'Saving…' : 'Save GitHub Settings'}
			</button>
		</div>
	</form>
{/if}

<style>
	.section-header { margin: 0 0 0.75rem; }
	.section-header h2 {
		margin: 0 0 0.2rem;
		font-size: 1rem; font-weight: 600;
	}
	.section-hint { margin: 0; font-size: 0.85rem; color: var(--text-muted); }

	.card { margin-bottom: 0; }

	.provider-tabs {
		display: flex; gap: 0.5rem; flex-wrap: wrap;
	}
	.provider-tab {
		display: flex; align-items: center; gap: 0.4rem;
		padding: 0.45rem 1rem; border-radius: var(--radius);
		border: 1px solid var(--border); background: var(--bg);
		color: var(--text-muted); font-size: 0.875rem; cursor: pointer;
		transition: all 0.15s;
	}
	.provider-tab:hover { border-color: var(--primary); color: var(--text); }
	.provider-tab.active {
		background: var(--primary); border-color: var(--primary);
		color: white; font-weight: 500;
	}

	.provider-fields { margin-top: 1.25rem; border-top: 1px solid var(--border); padding-top: 1.25rem; }
	.provider-none { color: var(--text-muted); font-size: 0.9rem; margin: 1rem 0 0; }

	.form-row { display: flex; gap: 1rem; }
	.form-row .form-group { flex: 1; }

	.form-hint { display: block; margin-top: 0.3rem; font-size: 0.8rem; color: var(--text-muted); }

	.test-row { display: flex; gap: 1rem; align-items: flex-end; }

	.form-actions { margin-top: 1.5rem; display: flex; justify-content: flex-end; }
</style>
