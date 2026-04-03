<script lang="ts">
	import { onMount } from 'svelte';
	import { ee as eeApi, ApiError } from '$lib/api';
	import { isAdmin } from '$lib/stores';
	import { toast } from '$lib/toast';
	import { goto } from '$app/navigation';
	import type { EEFiles } from '$lib/types';

	const tabs = [
		{ key: 'execution_environment_yml', label: 'execution-environment.yml' },
		{ key: 'requirements_yml',          label: 'requirements.yml' },
		{ key: 'requirements_txt',          label: 'requirements.txt' },
		{ key: 'bindep_txt',                label: 'bindep.txt' },
	] as const;

	type TabKey = (typeof tabs)[number]['key'];

	let loading = $state(true);
	let notConfigured = $state(false);
	let activeTab = $state<TabKey>('execution_environment_yml');
	let commitMessage = $state('');
	let saving = $state(false);

	let files = $state<EEFiles>({
		execution_environment_yml: { content: '', sha: '' },
		requirements_yml:          { content: '', sha: '' },
		requirements_txt:          { content: '', sha: '' },
		bindep_txt:                { content: '', sha: '' },
	});

	let activeContent = $derived(files[activeTab].content);

	onMount(async () => {
		if (!$isAdmin) { goto('/'); return; }
		await load();
	});

	async function load() {
		loading = true;
		notConfigured = false;
		try {
			files = await eeApi.get();
		} catch (err) {
			if (err instanceof ApiError && err.status === 503) {
				notConfigured = true;
			} else {
				toast.error('Failed to load EE files');
			}
		} finally {
			loading = false;
		}
	}

	function setContent(value: string) {
		files = { ...files, [activeTab]: { ...files[activeTab], content: value } };
	}

	async function save() {
		if (!commitMessage.trim()) return;
		saving = true;
		try {
			await eeApi.update({ message: commitMessage.trim(), files });
			toast.success('Changes committed to GitHub');
			// Reload to get updated SHAs
			await load();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Commit failed');
		} finally {
			saving = false;
		}
	}
</script>

<div class="page-header">
	<div>
		<h1>Execution Environment</h1>
		<p class="subtitle">Edit EE definition files and commit them to GitHub.</p>
	</div>
</div>

{#if notConfigured}
	<div class="alert alert-error">
		GitHub integration is not configured. Go to <a href="/settings" style="color:inherit;font-weight:600">Settings → GitHub</a> to add your token and repository.
	</div>
{:else if loading}
	<p class="empty-state">Loading...</p>
{:else}
	<div class="card editor-card">
		<div class="form-group">
			<label for="commit-msg">Commit Message <span class="required">*</span></label>
			<input
				id="commit-msg"
				class="form-control"
				placeholder="e.g. Add boto3 to requirements"
				bind:value={commitMessage}
			/>
		</div>

		<div class="tab-bar" role="tablist">
			{#each tabs as tab}
				<button
					role="tab"
					aria-selected={activeTab === tab.key}
					class="tab-btn"
					class:active={activeTab === tab.key}
					onclick={() => (activeTab = tab.key)}
				>
					{tab.label}
				</button>
			{/each}
		</div>

		<textarea
			class="form-control ee-textarea"
			spellcheck="false"
			value={activeContent}
			oninput={(e) => setContent((e.target as HTMLTextAreaElement).value)}
		></textarea>

		<div class="editor-footer">
			<button
				class="btn btn-primary"
				disabled={saving || !commitMessage.trim()}
				onclick={save}
			>
				{saving ? 'Pushing...' : 'Save & Push'}
			</button>
		</div>
	</div>
{/if}

<style>
	.page-header { margin-bottom: 1.5rem; }
	.page-header h1 { margin: 0 0 0.25rem; }
	.subtitle { margin: 0; font-size: 0.875rem; color: var(--text-muted); }
	.subtitle code {
		font-family: monospace;
		font-size: 0.8rem;
		background: var(--bg-code, rgba(0,0,0,0.06));
		padding: 0.1em 0.35em;
		border-radius: 3px;
	}

	.editor-card { padding: 1.5rem; display: flex; flex-direction: column; gap: 1rem; }

	.tab-bar {
		display: flex;
		gap: 0;
		border-bottom: 1px solid var(--border, #e5e7eb);
		margin-bottom: -1rem; /* collapse into textarea top border */
	}
	.tab-btn {
		background: none;
		border: none;
		border-bottom: 2px solid transparent;
		padding: 0.45rem 0.85rem;
		font-size: 0.8rem;
		font-family: monospace;
		cursor: pointer;
		color: var(--text-muted);
		margin-bottom: -1px;
		transition: color 0.15s, border-color 0.15s;
	}
	.tab-btn:hover { color: var(--text); }
	.tab-btn.active {
		color: var(--primary);
		border-bottom-color: var(--primary);
		font-weight: 600;
	}

	.ee-textarea {
		font-family: monospace;
		font-size: 0.85rem;
		min-height: 20lh;
		resize: vertical;
		width: 100%;
		box-sizing: border-box;
		border-radius: 0 0 var(--radius) var(--radius);
	}

	.editor-footer { display: flex; justify-content: flex-end; }

	.required { color: var(--danger); }
</style>
