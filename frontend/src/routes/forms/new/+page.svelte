<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { forms as formsApi, servers as serversApi, playbooks as playbooksApi, vaults as vaultsApi, serverGroups as sgApi, hosts as hostsApi, ApiError } from '$lib/api';
	import type { Server, ServerGroup, Playbook, Vault, FormField, FieldType, Host, VarSuggestion } from '$lib/types';

	let serverList     = $state<Server[]>([]);
	let serverGroupList = $state<ServerGroup[]>([]);
	let sourceList     = $state<Playbook[]>([]);
	let vaultList      = $state<Vault[]>([]);
	let hostList       = $state<Host[]>([]);

	let targetMode     = $state<'host' | 'group'>('host');
	let formData       = $state({ name: '', description: '', runner_id: '', host_id: '', server_group_id: '', playbook_id: '', playbook_path: '', vault_id: '', is_quick_action: false, schedule_cron: '', schedule_enabled: false, notify_webhook: '', notify_email: '' });

	// Playbook file discovery
	let playbookFiles  = $state<string[]>([]);
	let filesLoading   = $state(false);
	let filesError     = $state('');

	// Variable suggestions
	let suggestions    = $state<VarSuggestion[]>([]);
	let suggestLoading = $state(false);

	let fields         = $state<Partial<FormField>[]>([]);
	let stagedImage    = $state<File | null>(null);
	let saving         = $state(false);
	let error          = $state('');
	let dragIndex      = $state<number | null>(null);
	let dragOverIndex  = $state<number | null>(null);

	onMount(async () => {
		[serverList, serverGroupList, sourceList, vaultList, hostList] = await Promise.all([
			serversApi.list(), sgApi.list(), playbooksApi.list(), vaultsApi.list(), hostsApi.list()
		]);
	});

	// When source changes, fetch available .yml files
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

	// When a playbook file is chosen, scan for variable suggestions
	async function onPlaybookFileChange() {
		suggestions = [];
		if (!formData.playbook_id || !formData.playbook_path) return;
		suggestLoading = true;
		try {
			suggestions = await playbooksApi.scanVars(formData.playbook_id, formData.playbook_path);
		} catch {
			// non-fatal — suggestions are best-effort
		} finally {
			suggestLoading = false;
		}
	}

	function addSuggestion(s: VarSuggestion) {
		if (fields.some(f => f.name === s.name)) return;
		fields = [...fields, {
			name: s.name,
			label: s.label,
			field_type: s.type as FieldType,
			default_value: s.default ?? '',
			options: '[]',
			required: s.required ?? false,
			sort_order: fields.length,
		}];
	}

	function isSuggestionAdded(name: string) {
		return fields.some(f => f.name === name);
	}

	function addField() {
		fields = [...fields, { name: '', label: '', field_type: 'text' as FieldType, default_value: '', options: '[]', required: false, sort_order: fields.length, depends_on_name: '', depends_on_operator: 'eq', depends_on_value: '' }];
	}

	function removeField(i: number) {
		const removed = fields[i].name;
		// Clear any dependencies that pointed to this field
		fields = fields
			.filter((_, idx) => idx !== i)
			.map(f => f.depends_on_name === removed ? { ...f, depends_on_name: '', depends_on_value: '' } : f);
	}

	function onDragStart(e: DragEvent, i: number) {
		dragIndex = i;
		e.dataTransfer!.effectAllowed = 'move';
	}

	function onDragOver(e: DragEvent, i: number) {
		e.preventDefault();
		e.dataTransfer!.dropEffect = 'move';
		dragOverIndex = i;
	}

	function onDrop(e: DragEvent, i: number) {
		e.preventDefault();
		if (dragIndex === null || dragIndex === i) { dragIndex = null; dragOverIndex = null; return; }
		const arr = [...fields];
		const [moved] = arr.splice(dragIndex, 1);
		arr.splice(i, 0, moved);
		fields = arr;
		dragIndex = null;
		dragOverIndex = null;
	}

	function onDragEnd() { dragIndex = null; dragOverIndex = null; }

	function dependsOnOptions(i: number) {
		return fields.slice(0, i).filter(f => f.name);
	}

	function onDependsOnChange(field: Partial<FormField>) {
		field.depends_on_operator = 'eq';
		field.depends_on_value = '';
	}

	function operatorNeedsValue(op: string) {
		return op !== 'empty' && op !== 'not_empty';
	}

	function getDependsParent(field: Partial<FormField>) {
		return fields.find(f => f.name === field.depends_on_name);
	}

	function getOptions(f: Partial<FormField>) {
		try { return JSON.parse(f.options || '[]').join(', '); } catch { return ''; }
	}

	function setOptions(f: Partial<FormField>, val: string) {
		f.options = JSON.stringify(val.split(',').map(s => s.trim()).filter(Boolean));
	}

	async function save() {
		saving = true;
		error = '';
		try {
			const payload = {
				...formData,
				server_id: formData.runner_id,
				host_id: targetMode === 'host' ? formData.host_id : '',
				server_group_id: targetMode === 'group' ? formData.server_group_id : '',
				fields,
			};
			const created = await formsApi.create(payload);
			if (stagedImage) {
				await formsApi.uploadImage(created.id, stagedImage);
			}
			goto('/forms');
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Save failed';
		} finally {
			saving = false;
		}
	}
</script>

<div class="page-header">
	<h1>New Form</h1>
	<a href="/forms" class="btn btn-secondary">← Back</a>
</div>

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
				<small class="hint">One run will be created per host in the group.</small>
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
							<button type="button"
								class="chip"
								class:added={isSuggestionAdded(s.name)}
								onclick={() => addSuggestion(s)}
								title="{s.type}{s.required ? ' · required' : ''}{s.default ? ` · default: ${s.default}` : ''}"
							>
								{#if isSuggestionAdded(s.name)}<span class="check">✓</span>{/if}
								{s.name}
								<span class="chip-type">{s.type}</span>
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
			<p class="empty-state" style="padding:0.5rem 0">No fields yet.{#if suggestions.length > 0} Click a suggestion above to add it.{/if}</p>
		{/if}
		{#each fields as field, i}
			<div class="field-row"
				class:drag-over={dragOverIndex === i && dragIndex !== i}
				ondragover={(e) => onDragOver(e, i)}
				ondrop={(e) => onDrop(e, i)}
				role="listitem"
			>
				<div class="drag-handle" draggable="true"
					ondragstart={(e) => onDragStart(e, i)}
					ondragend={onDragEnd}
					title="Drag to reorder"
				>⠿</div>
				<div class="field-body">
					<div class="field-grid">
						<div class="form-group">
							<label>Variable Name</label>
							<input class="form-control" bind:value={field.name} placeholder="e.g. db_host" required />
						</div>
						<div class="form-group">
							<label>Label</label>
							<input class="form-control" bind:value={field.label} placeholder="Display label" required />
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
							<label>Default Value</label>
							<input class="form-control" bind:value={field.default_value} />
						</div>
						{#if field.field_type === 'select'}
							<div class="form-group" style="grid-column: span 2">
								<label>Options (comma-separated)</label>
								<input class="form-control" value={getOptions(field)} oninput={(e) => setOptions(field, (e.target as HTMLInputElement).value)} placeholder="opt1, opt2, opt3" />
							</div>
						{/if}
						<div class="form-group field-required">
							<label><input type="checkbox" bind:checked={field.required} /> Required</label>
						</div>
					</div>
					{#if dependsOnOptions(i).length > 0}
						<div class="depends-row">
							<span class="depends-label">Depends on</span>
							<select class="form-control depends-select"
								bind:value={field.depends_on_name}
								onchange={() => onDependsOnChange(field)}
							>
								<option value="">— always show —</option>
								{#each dependsOnOptions(i) as opt}
									<option value={opt.name}>{opt.label || opt.name}</option>
								{/each}
							</select>
							{#if field.depends_on_name}
								<select class="form-control depends-op" bind:value={field.depends_on_operator}>
									<option value="eq">equals</option>
									<option value="neq">not equals</option>
									<option value="in">in</option>
									<option value="not_in">not in</option>
									<option value="contains">contains</option>
									<option value="not_contains">not contains</option>
									<option value="empty">is empty</option>
									<option value="not_empty">is not empty</option>
								</select>
								{#if operatorNeedsValue(field.depends_on_operator ?? 'eq')}
									{@const parent = getDependsParent(field)}
									{@const op = field.depends_on_operator ?? 'eq'}
									{#if op === 'in' || op === 'not_in'}
										<input class="form-control depends-val" bind:value={field.depends_on_value} placeholder="val1, val2, …" title="Comma-separated list of values" />
									{:else if parent?.field_type === 'bool' && (op === 'eq' || op === 'neq')}
										<select class="form-control depends-val" bind:value={field.depends_on_value}>
											<option value="true">true</option>
											<option value="false">false</option>
										</select>
									{:else if parent?.field_type === 'select' && (op === 'eq' || op === 'neq')}
										<select class="form-control depends-val" bind:value={field.depends_on_value}>
											<option value="">Select value…</option>
											{#each (() => { try { return JSON.parse(parent.options || '[]'); } catch { return []; } })() as opt}
												<option value={opt}>{opt}</option>
											{/each}
										</select>
									{:else}
										<input class="form-control depends-val" bind:value={field.depends_on_value} placeholder="value" />
									{/if}
								{/if}
							{/if}
						</div>
					{/if}
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
		{#if formData.is_quick_action}
			<div class="form-group" style="margin-top:0.5rem">
				<label>Quick Action Image (optional)</label>
				{#if stagedImage}
					<div style="display:flex;align-items:center;gap:0.75rem">
						<span class="file-badge">{stagedImage.name}</span>
						<button type="button" class="btn btn-sm btn-danger" onclick={() => stagedImage = null}>Remove</button>
					</div>
				{:else}
					<label class="btn btn-secondary file-label">
						Choose Image…
						<input type="file" accept="image/*" style="display:none"
							onchange={(e) => { stagedImage = (e.currentTarget as HTMLInputElement).files?.[0] ?? null; }} />
					</label>
				{/if}
			</div>
		{/if}
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
				<small class="hint">5-field cron (min hr dom mon dow) or @hourly · @daily · @weekly</small>
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
		<button type="submit" class="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : 'Create Form'}</button>
	</div>
</form>

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
	.field-row { display: flex; gap: 0.6rem; align-items: flex-start; padding: 0.6rem 0.75rem; border: 1px solid var(--border); border-radius: var(--radius); margin-bottom: 0.5rem; transition: border-color 0.12s, background 0.12s; }
	.field-row.drag-over { border-color: var(--primary); background: rgba(8,145,178,0.04); }
	.drag-handle { cursor: grab; color: var(--text-muted); font-size: 1.1rem; line-height: 1; padding-top: 1.6rem; user-select: none; flex-shrink: 0; }
	.drag-handle:active { cursor: grabbing; }
	.field-body { flex: 1; min-width: 0; }
	.field-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 0.5rem; }
	.field-required { display: flex; align-items: center; }
	.field-required label { display: flex; align-items: center; gap: 0.375rem; font-weight: normal; margin-bottom: 0; }
	.field-remove { align-self: flex-end; margin-bottom: 0.6rem; flex-shrink: 0; }
	.depends-row { display: flex; align-items: center; gap: 0.4rem; margin-top: 0.4rem; padding-top: 0.4rem; border-top: 1px dashed var(--border); flex-wrap: wrap; }
	.depends-label { font-size: 0.72rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-muted); white-space: nowrap; }
	.depends-eq { font-size: 0.8rem; color: var(--text-muted); }
	.depends-select { width: 160px; flex-shrink: 0; }
	.depends-op { width: 130px; flex-shrink: 0; }
	.depends-val { width: 130px; flex-shrink: 0; }
	.checkbox-label { display: flex; align-items: center; gap: 0.5rem; font-weight: 500; cursor: pointer; }
	.file-badge { display: inline-flex; align-items: center; background: #f0fdf4; border: 1px solid #bbf7d0; color: #166534; border-radius: 4px; padding: 0.15rem 0.5rem; font-size: 0.8rem; }
	.file-label { cursor: pointer; display: inline-flex; align-items: center; }
	.hint { display: block; margin-top: 0.2rem; font-size: 0.78rem; color: var(--text-muted); }
</style>
