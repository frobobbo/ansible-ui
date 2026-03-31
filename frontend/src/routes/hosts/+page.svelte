<script lang="ts">
	import { onMount } from 'svelte';
	import { hosts as hostsApi, ApiError } from '$lib/api';
	import { isAdmin } from '$lib/stores';
	import { toast, confirmDialog } from '$lib/toast';
	import type { Host } from '$lib/types';

	let list = $state<Host[]>([]);
	let loading = $state(true);
	let error = $state('');
	let filter = $state('');

	let filtered = $derived(
		filter.trim()
			? list.filter(
					(h) =>
						h.name.toLowerCase().includes(filter.toLowerCase()) ||
						h.address.toLowerCase().includes(filter.toLowerCase()) ||
						h.description.toLowerCase().includes(filter.toLowerCase())
				)
			: list
	);

	// Modal state
	let showModal = $state(false);
	let editingId = $state<string | null>(null);
	let form = $state({ name: '', address: '', description: '' });
	// Host vars edited as an array of {key, value} pairs for easy UI binding
	let varPairs = $state<{ key: string; value: string }[]>([]);
	let saving = $state(false);
	let formError = $state('');

	onMount(async () => { await load(); });

	async function load() {
		loading = true;
		try { list = await hostsApi.list(); }
		catch { error = 'Failed to load hosts'; }
		finally { loading = false; }
	}

	function pairsFromVars(vars: Record<string, string>) {
		return Object.entries(vars).map(([key, value]) => ({ key, value }));
	}

	function pairsToVars(pairs: { key: string; value: string }[]) {
		const vars: Record<string, string> = {};
		for (const { key, value } of pairs) {
			if (key.trim()) vars[key.trim()] = value;
		}
		return vars;
	}

	function openCreate() {
		editingId = null;
		form = { name: '', address: '', description: '' };
		varPairs = [];
		formError = '';
		showModal = true;
	}

	function openEdit(host: Host) {
		editingId = host.id;
		form = { name: host.name, address: host.address, description: host.description };
		varPairs = pairsFromVars(host.vars ?? {});
		formError = '';
		showModal = true;
	}

	function addVar() {
		varPairs = [...varPairs, { key: '', value: '' }];
	}

	function removeVar(i: number) {
		varPairs = varPairs.filter((_, idx) => idx !== i);
	}

	async function save() {
		saving = true;
		formError = '';
		const payload = { ...form, vars: pairsToVars(varPairs) };
		try {
			if (editingId) {
				await hostsApi.update(editingId, payload);
			} else {
				await hostsApi.create(payload);
			}
			showModal = false;
			toast.success(editingId ? 'Host updated' : 'Host added');
			await load();
		} catch (err) {
			formError = err instanceof ApiError ? err.message : 'Save failed';
		} finally {
			saving = false;
		}
	}

	async function remove(id: string, name: string) {
		if (!(await confirmDialog(`Delete host "${name}"?`))) return;
		try {
			await hostsApi.delete(id);
			await load();
			toast.success('Host deleted');
		} catch {
			toast.error('Delete failed');
		}
	}
</script>

<div class="page-header">
	<h1>Hosts</h1>
	<div class="header-right">
		<input class="form-control search" placeholder="Search hosts..." bind:value={filter} />
		{#if $isAdmin}
			<button class="btn btn-primary" onclick={openCreate}>+ Add Host</button>
		{/if}
	</div>
</div>

{#if error}<div class="alert alert-error">{error}</div>{/if}

{#if loading}
	<p class="empty-state">Loading...</p>
{:else if list.length === 0}
	<div class="empty-state">No hosts configured. {#if $isAdmin}Add one to get started.{/if}</div>
{:else if filtered.length === 0}
	<div class="empty-state">No hosts match "{filter}".</div>
{:else}
	<div class="card" style="padding:0">
		<table class="table">
			<thead>
				<tr>
					<th>Name</th>
					<th>Address</th>
					<th>Host Vars</th>
					<th>Actions</th>
				</tr>
			</thead>
			<tbody>
				{#each filtered as host}
					<tr>
						<td>
							<strong>{host.name}</strong>
							{#if host.description}
								<div class="row-desc">{host.description}</div>
							{/if}
						</td>
						<td class="mono">{host.address}</td>
						<td>
							{#if host.vars && Object.keys(host.vars).length > 0}
								<div class="var-chips">
									{#each Object.entries(host.vars) as [k, v]}
										<span class="var-chip"><span class="var-key">{k}</span>=<span class="var-val">{v}</span></span>
									{/each}
								</div>
							{:else}
								<span class="none">—</span>
							{/if}
						</td>
						<td>
							<div class="actions">
								{#if $isAdmin}
									<button class="btn btn-sm btn-secondary" onclick={() => openEdit(host)}>Edit</button>
									<button class="btn btn-sm btn-danger" onclick={() => remove(host.id, host.name)}>Delete</button>
								{/if}
							</div>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

{#if showModal}
	<div class="modal-overlay" onclick={() => showModal = false} role="presentation">
		<div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
			<h2>{editingId ? 'Edit Host' : 'Add Host'}</h2>
			{#if formError}<div class="alert alert-error">{formError}</div>{/if}
			<form onsubmit={(e) => { e.preventDefault(); save(); }} autocomplete="off">

				<div class="grid-2">
					<div class="form-group">
						<label>Name</label>
						<input class="form-control" bind:value={form.name} required placeholder="web-01" />
					</div>
					<div class="form-group">
						<label>Address</label>
						<input class="form-control" bind:value={form.address} required placeholder="192.168.1.10 or host.example.com" />
						<small class="hint">IP address or FQDN — used as the Ansible inventory host.</small>
					</div>
				</div>

				<div class="form-group">
					<label>Description <span class="hint-inline">(optional)</span></label>
					<input class="form-control" bind:value={form.description} placeholder="e.g. Primary web server" />
				</div>

				<div class="form-group">
					<div class="vars-header">
						<label>Host Vars <span class="hint-inline">(optional)</span></label>
						<button type="button" class="btn btn-sm btn-secondary" onclick={addVar}>+ Add Var</button>
					</div>
					<small class="hint">Key-value pairs written as <code>host_vars</code> in the Ansible inventory for this host.</small>

					{#if varPairs.length > 0}
						<div class="var-rows">
							{#each varPairs as pair, i}
								<div class="var-row">
									<input
										class="form-control var-input"
										bind:value={pair.key}
										placeholder="ansible_user"
										aria-label="Variable name"
									/>
									<span class="var-eq">=</span>
									<input
										class="form-control var-input"
										bind:value={pair.value}
										placeholder="ubuntu"
										aria-label="Variable value"
									/>
									<button type="button" class="btn-remove-var" onclick={() => removeVar(i)} aria-label="Remove variable">
										<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" width="14" height="14">
											<line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
										</svg>
									</button>
								</div>
							{/each}
						</div>
					{:else}
						<p class="no-vars">No host vars defined. Click "+ Add Var" to add one.</p>
					{/if}
				</div>

				<div class="actions" style="justify-content:flex-end; margin-top:1rem">
					<button type="button" class="btn btn-secondary" onclick={() => showModal = false}>Cancel</button>
					<button type="submit" class="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.header-right { display: flex; gap: 0.75rem; align-items: center; }
	.search { width: 220px; }
	.mono { font-family: monospace; font-size: 0.85rem; }
	.row-desc { font-size: 0.78rem; color: var(--text-muted); margin-top: 0.1rem; }
	.none { color: var(--text-muted); }
	.var-chips { display: flex; flex-wrap: wrap; gap: 0.3rem; }
	.var-chip {
		font-family: monospace; font-size: 0.75rem;
		background: var(--bg-alt, #f1f5f9); border: 1px solid var(--border);
		border-radius: 4px; padding: 0.1rem 0.4rem;
	}
	.var-key { color: var(--primary); }
	.var-val { color: var(--text-muted); }
	.vars-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.25rem; }
	.vars-header label { margin: 0; }
	.var-rows { display: flex; flex-direction: column; gap: 0.4rem; margin-top: 0.5rem; }
	.var-row { display: flex; align-items: center; gap: 0.4rem; }
	.var-input { flex: 1; }
	.var-eq { color: var(--text-muted); font-family: monospace; flex-shrink: 0; }
	.btn-remove-var {
		background: none; border: none; cursor: pointer; padding: 0.25rem;
		color: var(--text-muted); border-radius: 4px; display: flex; align-items: center;
		flex-shrink: 0;
	}
	.btn-remove-var:hover { color: var(--danger); background: color-mix(in srgb, var(--danger) 10%, transparent); }
	.no-vars { font-size: 0.85rem; color: var(--text-muted); margin: 0.4rem 0 0; }
	.hint-inline { font-weight: normal; font-size: 0.8rem; color: var(--text-muted); }
	.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 100; }
	.modal { background: white; border-radius: var(--radius); padding: 2rem; width: 100%; max-width: 600px; max-height: 90vh; overflow-y: auto; }
</style>
