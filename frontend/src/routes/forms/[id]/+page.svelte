<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { forms as formsApi, servers as serversApi, playbooks as playbooksApi, vaults as vaultsApi, serverGroups as sgApi, hosts as hostsApi, ApiError } from '$lib/api';
	import type { Server, ServerGroup, Playbook, Vault, FormField, FieldType, Host, VarSuggestion } from '$lib/types';

	let id = $derived($page.params.id);

	let serverList      = $state<Server[]>([]);
	let serverGroupList = $state<ServerGroup[]>([]);
	let sourceList      = $state<Playbook[]>([]);
	let vaultList       = $state<Vault[]>([]);
	let hostList        = $state<Host[]>([]);
	let targetMode      = $state<'host' | 'group'>('host');
	let formData        = $state({ name: '', description: '', runner_id: '', host_id: '', server_group_id: '', playbook_id: '', playbook_path: '', vault_id: '', is_quick_action: false, schedule_cron: '', schedule_enabled: false, notify_webhook: '', notify_email: '' });
	let nextRunAt       = $state<string | null>(null);
	let webhookToken    = $state('');
	let imageName       = $state('');
	let imageUploading  = $state(false);
	let fields          = $state<Partial<FormField>[]>([]);
	let loading         = $state(true);
	let saving          = $state(false);
	let error           = $state('');

	// Playbook file discovery
	let playbookFiles   = $state<string[]>([]);
	let filesLoading    = $state(false);
	let filesError      = $state('');

	// Variable suggestions
	let suggestions     = $state<VarSuggestion[]>([]);
	let suggestLoading  = $state(false);

	onMount(async () => {
		const [form, svList, sgList, pbList, vList, hList] = await Promise.all([
			formsApi.get(id), serversApi.list(), sgApi.list(), playbooksApi.list(), vaultsApi.list(), hostsApi.list()
		]);
		serverList = svList;
		serverGroupList = sgList;
		sourceList = pbList;
		vaultList = vList;
		hostList = hList;
		if (form) {
			targetMode = form.server_group_id ? 'group' : 'host';
			formData = {
				name: form.name, description: form.description,
				runner_id: form.server_id ?? '', host_id: form.host_id ?? '',
				server_group_id: form.server_group_id ?? '',
				playbook_id: form.playbook_id, playbook_path: form.playbook_path ?? '',
				vault_id: form.vault_id ?? '', is_quick_action: form.is_quick_action,
				schedule_cron: form.schedule_cron ?? '', schedule_enabled: form.schedule_enabled ?? false,
				notify_webhook: form.notify_webhook ?? '', notify_email: form.notify_email ?? ''
			};
			nextRunAt = form.next_run_at ?? null;
			webhookToken = form.webhook_token ?? '';
			imageName = form.image_name;
			fields = form.fields || [];
			// Pre-load playbook files for the current source
			if (form.playbook_id) {
				filesLoading = true;
				try { playbookFiles = await playbooksApi.listFiles(form.playbook_id); } catch {}
				finally { filesLoading = false; }
			}
		}
		loading = false;
	});

	async function onSourceChange() {
		formData.playbook_path = '';
		suggestions = [];
		playbookFiles = [];
		filesError = '';
		if (!formData.playbook_id) return;
		filesLoading = true;
		try {
			playbookFiles = await playbooksApi.listFiles(formData.playbook_id);
		} catch (e) {
			filesError = e instanceof ApiError ? e.message : 'Failed to list playbooks';
		} finally {
			filesLoading = false;
		}
	}

	async function onPlaybookFileChange() {
		suggestions = [];
		if (!formData.playbook_id || !formData.playbook_path) return;
		suggestLoading = true;
		try {
			suggestions = await playbooksApi.scanVars(formData.playbook_id, formData.playbook_path);
		} catch {
			// non-fatal
		} finally {
			suggestLoading = false;
		}
	}

	function addSuggestion(s: VarSuggestion) {
		if (fields.some(f => f.name === s.name)) return;
		fields = [...fields, {
			name: s.name, label: s.label, field_type: s.type as FieldType,
			default_value: s.default ?? '', options: '[]', required: s.required ?? false, sort_order: fields.length,
		}];
	}

	function isSuggestionAdded(name: string) { return fields.some(f => f.name === name); }

	function addField() {
		fields = [...fields, { name: '', label: '', field_type: 'text' as FieldType, default_value: '', options: '[]', required: false, sort_order: fields.length }];
	}

	function removeField(i: number) { fields = fields.filter((_, idx) => idx !== i); }

	function getOptions(f: Partial<FormField>) {
		try { return JSON.parse(f.options || '[]').join(', '); } catch { return ''; }
	}

	function setOptions(f: Partial<FormField>, val: string) {
		f.options = JSON.stringify(val.split(',').map(s => s.trim()).filter(Boolean));
	}

	async function save() {
		saving = true; error = '';
		try {
			const payload = {
				...formData, server_id: formData.runner_id,
				host_id: targetMode === 'host' ? formData.host_id : '',
				server_group_id: targetMode === 'group' ? formData.server_group_id : '',
				fields,
			};
			await formsApi.update(id, payload);
			goto('/forms');
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Save failed';
		} finally {
			saving = false;
		}
	}

	async function uploadImage(input: HTMLInputElement) {
		const file = input.files?.[0];
		if (!file) return;
		imageUploading = true;
		try { const updated = await formsApi.uploadImage(id, file); imageName = updated.image_name; }
		catch (err) { alert(err instanceof ApiError ? err.message : 'Upload failed'); }
		finally { imageUploading = false; input.value = ''; }
	}

	async function removeImage() {
		if (!confirm('Remove the image from this form?')) return;
		try { const updated = await formsApi.deleteImage(id); imageName = updated.image_name; }
		catch { alert('Failed to remove image'); }
	}

	async function generateWebhookToken() {
		try { const updated = await formsApi.regenerateWebhookToken(id); webhookToken = updated.webhook_token; }
		catch (err) { alert(err instanceof ApiError ? err.message : 'Failed'); }
	}

	async function revokeWebhookToken() {
		if (!confirm('Revoke the webhook token?')) return;
		try { const updated = await formsApi.revokeWebhookToken(id); webhookToken = updated.webhook_token; }
		catch (err) { alert(err instanceof ApiError ? err.message : 'Failed'); }
	}

	let webhookUrl = $derived(webhookToken ? `${location.origin}/api/webhook/forms/${webhookToken}` : '');
</script>

<div class="page-header">
	<h1>Edit Form</h1>
	<div class="actions">
		<a href="/forms/{id}/run" class="btn btn-primary">Run</a>
		<a href="/forms" class="btn btn-secondary">← Back</a>
	</div>
</div>

{#if loading}
	<p class="empty-state">Loading...</p>
{:else}
	{#if error}<div class="alert alert-error">{error}</div>{/if}
	<form onsubmit={(e) => { e.preventDefault(); save(); }} autocomplete="off">

		<!-- ── Basic Info ── -->
		<div class="card">
			<h2>Basic Info</h2>
			<div class="grid-2">
				<div class="form-group">
					<label>Form Name</label>
					<input class="form-control" bind:value={formData.name} required />
				</div>
				<div class="form-group">
					<label>Description</label>
					<input class="form-control" bind:value={formData.description} />
				</div>
			</div>
		</div>

		<!-- ── Target ── -->
		<div class="card">
			<h2>Target</h2>
			<div class="form-group">
				<div class="toggle-tabs">
					<button type="button" class="tab-btn" class:active={targetMode === 'host'} onclick={() => targetMode = 'host'}>Host</button>
					<button type="button" class="tab-btn" class:active={targetMode === 'group'} onclick={() => targetMode = 'group'}>Host Group</button>
				</div>
			</div>
			{#if targetMode === 'host'}
				<div class="form-group">
					<label>Host</label>
					<select class="form-control" bind:value={formData.host_id} required>
						<option value="">Select host...</option>
						{#each hostList as h}<option value={h.id}>{h.name} ({h.address})</option>{/each}
					</select>
				</div>
			{:else}
				<div class="form-group">
					<label>Host Group</label>
					<select class="form-control" bind:value={formData.server_group_id} required>
						<option value="">Select group...</option>
						{#each serverGroupList as g}<option value={g.id}>{g.name}</option>{/each}
					</select>
					<small class="hint">One run per host in the group.</small>
				</div>
			{/if}
		</div>

		<!-- ── Playbook ── -->
		<div class="card">
			<h2>Playbook</h2>
			<div class="form-group">
				<label>Playbook Source</label>
				<select class="form-control" bind:value={formData.playbook_id} onchange={onSourceChange} required>
					<option value="">Select source...</option>
					{#each sourceList as s}<option value={s.id}>{s.name} — {s.repo_url}</option>{/each}
				</select>
			</div>

			{#if formData.playbook_id}
				<div class="form-group">
					<label>Playbook</label>
					{#if filesLoading}
						<div class="loading-row"><span class="spinner"></span> Scanning repository…</div>
					{:else if filesError}
						<div class="alert alert-error" style="margin:0">{filesError}</div>
					{:else}
						<select class="form-control" bind:value={formData.playbook_path} onchange={onPlaybookFileChange} required>
							<option value="">Select playbook file...</option>
							{#each playbookFiles as f}<option value={f}>{f}</option>{/each}
						</select>
					{/if}
				</div>
			{/if}

			{#if formData.playbook_path && (suggestLoading || suggestions.length > 0)}
				<div class="suggestions-panel">
					<div class="suggestions-label">
						Field Suggestions
						{#if suggestLoading}<span class="spinner sm"></span>{/if}
					</div>
					{#if !suggestLoading}
						<div class="suggestion-chips">
							{#each suggestions as s}
								<button type="button" class="chip" class:added={isSuggestionAdded(s.name)}
									onclick={() => addSuggestion(s)}
									title="{s.type}{s.required ? ' · required' : ''}{s.default ? ` · default: ${s.default}` : ''}"
								>
									{#if isSuggestionAdded(s.name)}<span class="check">✓</span>{/if}
									{s.name}<span class="chip-type">{s.type}</span>
									{#if s.required}<span class="chip-req">req</span>{/if}
								</button>
							{/each}
						</div>
						{#if suggestions.length === 0}
							<p class="no-suggestions">No variables detected in this playbook.</p>
						{/if}
					{/if}
				</div>
			{/if}
		</div>

		<!-- ── Fields ── -->
		<div class="card">
			<div class="section-header">
				<h2>Fields</h2>
				<button type="button" class="btn btn-secondary btn-sm" onclick={addField}>+ Add Field</button>
			</div>
			{#if fields.length === 0}
				<p class="empty-state" style="padding:0.5rem 0">No fields. Add fields to capture Ansible variables.</p>
			{/if}
			{#each fields as field, i}
				<div class="field-row">
					<div class="field-grid">
						<div class="form-group">
							<label>Variable Name</label>
							<input class="form-control" bind:value={field.name} placeholder="e.g. db_host" required />
						</div>
						<div class="form-group">
							<label>Label</label>
							<input class="form-control" bind:value={field.label} required />
						</div>
						<div class="form-group">
							<label>Type</label>
							<select class="form-control" bind:value={field.field_type}>
								<option value="text">Text</option>
								<option value="number">Number</option>
								<option value="bool">Boolean</option>
								<option value="select">Select</option>
							</select>
						</div>
						<div class="form-group">
							<label>Default</label>
							<input class="form-control" bind:value={field.default_value} />
						</div>
						{#if field.field_type === 'select'}
							<div class="form-group" style="grid-column: span 2">
								<label>Options (comma-separated)</label>
								<input class="form-control" value={getOptions(field)} oninput={(e) => setOptions(field, (e.target as HTMLInputElement).value)} />
							</div>
						{/if}
						<div class="form-group field-required">
							<label><input type="checkbox" bind:checked={field.required} /> Required</label>
						</div>
					</div>
					<button type="button" class="btn btn-sm btn-danger field-remove" onclick={() => removeField(i)}>✕</button>
				</div>
			{/each}
		</div>

		<!-- ── Runner ── -->
		<div class="card">
			<h2>Job Runner</h2>
			<div class="form-group">
				<select class="form-control" bind:value={formData.runner_id} required>
					<option value="">Select job runner...</option>
					{#each serverList as sv}<option value={sv.id}>{sv.name}</option>{/each}
				</select>
				<small class="hint">The server or container that executes ansible-playbook.</small>
			</div>
		</div>

		<!-- ── Options ── -->
		<div class="card">
			<h2>Options</h2>
			<div class="grid-2">
				<div class="form-group">
					<label>Vault (optional)</label>
					<select class="form-control" bind:value={formData.vault_id}>
						<option value="">None</option>
						{#each vaultList as v}<option value={v.id}>{v.name}</option>{/each}
					</select>
					<small class="hint">Passes --vault-password-file when running.</small>
				</div>
				<div class="form-group">
					<label class="checkbox-label">
						<input type="checkbox" bind:checked={formData.is_quick_action} />
						Quick Action
					</label>
					<small class="hint">Shows as a card on the dashboard for all users.</small>
				</div>
			</div>
			<div class="form-group" style="margin-top:0.5rem">
				<label>Quick Action Image (optional)</label>
				{#if imageName}
					<div class="image-preview-row">
						<img src="/api/forms/{id}/image" alt={formData.name} class="image-preview" />
						<div>
							<div class="file-badge">{imageName}</div>
							<button type="button" class="btn btn-sm btn-danger" style="margin-top:0.5rem" onclick={removeImage}>Remove Image</button>
						</div>
					</div>
				{:else}
					<label class="btn btn-secondary file-label" class:disabled={imageUploading}>
						{imageUploading ? 'Uploading…' : 'Choose Image…'}
						<input type="file" accept="image/*" style="display:none" disabled={imageUploading}
							onchange={(e) => uploadImage(e.currentTarget as HTMLInputElement)} />
					</label>
				{/if}
			</div>
		</div>

		<!-- ── Scheduling ── -->
		<div class="card">
			<h2>Scheduling</h2>
			<div class="form-group">
				<label class="checkbox-label">
					<input type="checkbox" bind:checked={formData.schedule_enabled} />
					Run on a schedule
				</label>
				<small class="hint">Runs automatically using field default values. Times are UTC.</small>
			</div>
			{#if formData.schedule_enabled}
				<div class="form-group">
					<label>Cron Expression</label>
					<input class="form-control" bind:value={formData.schedule_cron} placeholder="0 2 * * *" required={formData.schedule_enabled} />
					<small class="hint">5-field cron or @hourly · @daily · @weekly</small>
					{#if nextRunAt}<small class="hint">Next run: {new Date(nextRunAt).toLocaleString()}</small>{/if}
				</div>
			{/if}
		</div>

		<!-- ── Notifications ── -->
		<div class="card">
			<h2>Notifications</h2>
			<div class="grid-2">
				<div class="form-group">
					<label>Webhook URL (on completion)</label>
					<input class="form-control" type="url" bind:value={formData.notify_webhook} placeholder="https://hooks.example.com/…" />
					<small class="hint">POST with JSON payload when the run completes.</small>
				</div>
				<div class="form-group">
					<label>Email (on completion)</label>
					<input class="form-control" bind:value={formData.notify_email} placeholder="user@example.com" />
					<small class="hint">Comma-separated. Requires SMTP_HOST env var.</small>
				</div>
			</div>
		</div>

		<div class="actions" style="justify-content:flex-end">
			<a href="/forms" class="btn btn-secondary">Cancel</a>
			<button type="submit" class="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : 'Save Changes'}</button>
		</div>
	</form>

	<!-- ── Webhook ── -->
	<div class="card">
		<h2>Webhook Trigger</h2>
		<p class="hint" style="margin-bottom:1rem">Trigger this form via an unauthenticated HTTP POST. The token acts as the credential — keep it secret.</p>
		{#if webhookToken}
			<div class="form-group">
				<label>Webhook URL</label>
				<div class="webhook-row">
					<code class="webhook-url">{webhookUrl}</code>
					<button type="button" class="btn btn-sm btn-secondary" onclick={() => navigator.clipboard.writeText(webhookUrl)}>Copy</button>
				</div>
				<small class="hint">POST to this URL (with optional JSON body to override field defaults).</small>
			</div>
			<div class="actions">
				<button type="button" class="btn btn-secondary btn-sm" onclick={generateWebhookToken}>Regenerate</button>
				<button type="button" class="btn btn-danger btn-sm" onclick={revokeWebhookToken}>Revoke</button>
			</div>
		{:else}
			<button type="button" class="btn btn-secondary" onclick={generateWebhookToken}>Generate Webhook Token</button>
		{/if}
	</div>
{/if}

<style>
	/* compact card + form spacing */
	:global(form .card) { padding: 1rem 1.25rem; margin-bottom: 0.75rem; }
	:global(form .card h2) { font-size: 0.8rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.06em; color: var(--text-muted); margin: 0 0 0.75rem; }
	:global(form .form-group) { margin-bottom: 0.6rem; }
	:global(form .form-group:last-child) { margin-bottom: 0; }
	:global(form .form-control) { padding: 0.3rem 0.6rem; font-size: 0.875rem; }
	:global(form .grid-2) { gap: 0.75rem; }

	.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem; }
	.section-header h2 { margin-bottom: 0; }
	.toggle-tabs { display: flex; border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; width: fit-content; }
	.tab-btn { padding: 0.3rem 1rem; background: none; border: none; cursor: pointer; font-size: 0.83rem; color: var(--text-muted); transition: background 0.12s, color 0.12s; }
	.tab-btn.active { background: var(--primary); color: #fff; }
	.loading-row { display: flex; align-items: center; gap: 0.5rem; color: var(--text-muted); font-size: 0.875rem; padding: 0.4rem 0; }
	.spinner { display: inline-block; width: 14px; height: 14px; border: 2px solid var(--border); border-top-color: var(--primary); border-radius: 50%; animation: spin 0.7s linear infinite; flex-shrink: 0; }
	.spinner.sm { width: 11px; height: 11px; }
	@keyframes spin { to { transform: rotate(360deg); } }
	.suggestions-panel { margin-top: 0.6rem; padding: 0.6rem 0.75rem; background: var(--bg); border: 1px solid var(--border); border-radius: var(--radius); }
	.suggestions-label { font-size: 0.72rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.07em; color: var(--text-muted); margin-bottom: 0.4rem; display: flex; align-items: center; gap: 0.4rem; }
	.suggestion-chips { display: flex; flex-wrap: wrap; gap: 0.35rem; }
	.chip { display: inline-flex; align-items: center; gap: 0.3rem; padding: 0.2rem 0.55rem; border-radius: 999px; border: 1px solid var(--border); background: var(--surface); font-size: 0.78rem; cursor: pointer; transition: background 0.12s, border-color 0.12s, color 0.12s; color: var(--text); }
	.chip:hover:not(.added) { background: var(--primary); border-color: var(--primary); color: white; }
	.chip.added { background: var(--bg); color: var(--text-muted); cursor: default; border-style: dashed; }
	.chip-type { font-size: 0.68rem; background: rgba(20,184,212,0.12); color: var(--primary); border-radius: 4px; padding: 0 0.25rem; }
	.chip-req { font-size: 0.68rem; background: rgba(224,53,53,0.1); color: var(--danger); border-radius: 4px; padding: 0 0.25rem; }
	.check { color: var(--success); font-size: 0.75rem; }
	.no-suggestions { font-size: 0.8rem; color: var(--text-muted); margin: 0; }
	.field-row { display: flex; gap: 0.6rem; align-items: flex-start; padding: 0.6rem 0.75rem; border: 1px solid var(--border); border-radius: var(--radius); margin-bottom: 0.5rem; }
	.field-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 0.5rem; flex: 1; }
	.field-required { display: flex; align-items: center; }
	.field-required label { display: flex; align-items: center; gap: 0.375rem; font-weight: normal; margin-bottom: 0; }
	.field-remove { align-self: flex-end; margin-bottom: 0.6rem; }
	.checkbox-label { display: flex; align-items: center; gap: 0.5rem; font-weight: 500; cursor: pointer; }
	.image-preview-row { display: flex; align-items: flex-start; gap: 0.75rem; margin-bottom: 0.5rem; }
	.image-preview { width: 64px; height: 64px; object-fit: cover; border-radius: var(--radius); border: 1px solid var(--border); }
	.file-badge { display: inline-flex; align-items: center; background: #f0fdf4; border: 1px solid #bbf7d0; color: #166534; border-radius: 4px; padding: 0.15rem 0.5rem; font-size: 0.8rem; }
	.file-label { cursor: pointer; display: inline-flex; align-items: center; }
	.file-label.disabled { opacity: 0.6; cursor: not-allowed; }
	.hint { display: block; margin-top: 0.2rem; font-size: 0.78rem; color: var(--text-muted); }
	.webhook-row { display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.25rem; }
	.webhook-url { background: var(--bg); border: 1px solid var(--border); border-radius: var(--radius); padding: 0.375rem 0.625rem; font-size: 0.8rem; word-break: break-all; flex: 1; }
</style>
