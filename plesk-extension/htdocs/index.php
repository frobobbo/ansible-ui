<?php
pm_Context::init('automation-hub');
$configured  = pm_Settings::get('hub_url', '')      !== ''
            && pm_Settings::get('hub_username', '') !== '';
$settingsUrl = 'settings.php';
$proxyUrl    = 'proxy.php';
?>
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Automation Hub</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#f1f5f9;color:#1e293b;padding:2rem}
.wrap{max-width:960px;margin:0 auto}
/* Header */
.topbar{display:flex;align-items:center;justify-content:space-between;margin-bottom:1.5rem}
.topbar h1{font-size:1.25rem;font-weight:700;color:#0f172a}
/* Buttons */
.btn{display:inline-flex;align-items:center;gap:.4rem;padding:.45rem 1rem;
    border-radius:6px;border:1px solid #cbd5e1;background:#fff;color:#334155;
    font-size:.875rem;cursor:pointer;text-decoration:none;transition:all .15s}
.btn:hover{background:#f1f5f9;border-color:#94a3b8}
.btn-primary{background:#0891b2;border-color:#0891b2;color:#fff}
.btn-primary:hover{background:#0e7490;border-color:#0e7490}
.btn-primary:disabled{background:#a5f3fc;border-color:#a5f3fc;cursor:not-allowed}
/* Alerts */
.alert{padding:.875rem 1.25rem;border-radius:8px;margin-bottom:1.5rem;font-size:.9rem}
.alert-warn{background:#fef9c3;border:1px solid #fde047;color:#713f12}
.alert a{color:inherit;font-weight:600}
/* Forms grid */
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(280px,1fr));gap:1rem}
.card{background:#fff;border:1px solid #e2e8f0;border-radius:10px;padding:1.25rem;
    display:flex;flex-direction:column;gap:.75rem;transition:box-shadow .15s}
.card:hover{box-shadow:0 4px 16px rgba(0,0,0,.08);border-color:#94a3b8}
.card-name{font-weight:600;font-size:1rem;color:#0f172a}
.card-desc{font-size:.85rem;color:#64748b;flex:1}
.card-foot{display:flex;align-items:center;justify-content:space-between}
.badge{font-size:.7rem;font-weight:600;padding:.2rem .6rem;border-radius:999px;
    background:#e0f2fe;color:#0369a1}
/* Runner panel */
.runner{background:#fff;border:1px solid #0891b2;border-radius:10px;margin-top:1.5rem;overflow:hidden}
.runner-head{background:#0891b2;color:#fff;padding:.875rem 1.25rem;
    display:flex;align-items:center;justify-content:space-between}
.runner-head h2{font-size:1rem;margin:0}
.close-btn{background:none;border:none;color:#fff;font-size:1.25rem;cursor:pointer;
    opacity:.8;line-height:1;padding:0}
.close-btn:hover{opacity:1}
.runner-body{padding:1.25rem}
/* Fields */
.fields{display:grid;gap:.875rem;margin-bottom:1.25rem}
.field label{display:block;font-size:.75rem;font-weight:700;text-transform:uppercase;
    letter-spacing:.06em;color:#475569;margin-bottom:.35rem}
.field input[type=text],.field input[type=number],.field select{
    width:100%;padding:.5rem .75rem;border:1px solid #cbd5e1;border-radius:6px;
    font-size:.9rem;background:#f8fafc}
.field input:focus,.field select:focus{outline:2px solid #0891b2;border-color:transparent;background:#fff}
.field-bool{display:flex;align-items:center;gap:.5rem}
.field-bool input{width:18px;height:18px;accent-color:#0891b2}
.run-bar{display:flex;gap:.75rem;align-items:center}
/* Status badge */
.status{font-size:.8rem;font-weight:600;padding:.3rem .7rem;border-radius:6px}
.s-pending,.s-running{background:#e0f2fe;color:#0369a1}
.s-success{background:#dcfce7;color:#166534}
.s-failed{background:#fee2e2;color:#991b1b}
/* Output */
.output{margin-top:1.25rem;border-radius:8px;overflow:hidden;border:1px solid #e2e8f0}
.output-bar{background:#1e293b;color:#94a3b8;font-size:.75rem;padding:.4rem .875rem;
    display:flex;justify-content:space-between;font-family:monospace}
.output pre{margin:0;padding:.875rem;background:#0f172a;color:#e2e8f0;
    font-family:'Courier New',monospace;font-size:.8rem;line-height:1.6;
    max-height:420px;overflow-y:auto;white-space:pre-wrap;word-break:break-all}
/* Misc */
.empty{text-align:center;padding:3rem 1rem;color:#94a3b8;font-size:.95rem}
.spinner{display:inline-block;width:14px;height:14px;border:2px solid #bae6fd;
    border-top-color:#0891b2;border-radius:50%;animation:spin .7s linear infinite;vertical-align:middle}
@keyframes spin{to{transform:rotate(360deg)}}
</style>
</head>
<body>
<div class="wrap">

    <div class="topbar">
        <h1>Automation Hub</h1>
        <a href="<?= htmlspecialchars($settingsUrl) ?>" class="btn">&#9881; Settings</a>
    </div>

    <?php if (!$configured): ?>
        <div class="alert alert-warn">
            Not configured. <a href="<?= htmlspecialchars($settingsUrl) ?>">Open Settings</a>
            to enter your Automation Hub URL and credentials.
        </div>
    <?php else: ?>

        <div id="forms-area">
            <div class="empty"><span class="spinner"></span>&nbsp; Loading forms&hellip;</div>
        </div>

        <div id="runner" class="runner" style="display:none">
            <div class="runner-head">
                <h2 id="runner-title">Run Form</h2>
                <button class="close-btn" onclick="AH.close()" title="Close">&#x2715;</button>
            </div>
            <div class="runner-body">
                <div id="fields" class="fields"></div>
                <div class="run-bar">
                    <button id="run-btn" class="btn btn-primary" onclick="AH.run()">&#9654; Run</button>
                    <span id="run-status"></span>
                </div>
                <div id="output" class="output" style="display:none">
                    <div class="output-bar">
                        <span>Output</span>
                        <span id="output-status"></span>
                    </div>
                    <pre id="output-text"></pre>
                </div>
            </div>
        </div>

    <?php endif ?>
</div>

<script>
const AH = (() => {
    const PROXY = <?= json_encode($proxyUrl) ?>;
    let form = null, fields = [], poll = null;

    async function api(method, endpoint, body = null) {
        const r = await fetch(PROXY, {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({method, endpoint, body}),
        });
        const d = await r.json();
        if (d.error) throw new Error(d.error);
        return d;
    }

    // ── Forms list ────────────────────────────────────────────────────────────

    async function loadForms() {
        try {
            const forms = await api('GET', '/api/forms');
            renderForms(Array.isArray(forms) ? forms : []);
        } catch(e) {
            document.getElementById('forms-area').innerHTML =
                `<div class="alert alert-warn">Failed to load forms: ${esc(e.message)}</div>`;
        }
    }

    function renderForms(list) {
        const el = document.getElementById('forms-area');
        if (!list.length) {
            el.innerHTML = '<div class="empty">No forms found. Create forms in Automation Hub first.</div>';
            return;
        }
        el.innerHTML = '<div class="grid">' + list.map(f => `
            <div class="card">
                <div class="card-name">${esc(f.name)}</div>
                <div class="card-desc">${esc(f.description || '')}</div>
                <div class="card-foot">
                    ${f.is_quick_action ? '<span class="badge">Quick Action</span>' : '<span></span>'}
                    <button class="btn btn-primary" onclick="AH.open(${esc(JSON.stringify(f))})">&#9654; Run</button>
                </div>
            </div>
        `).join('') + '</div>';
    }

    // ── Runner ────────────────────────────────────────────────────────────────

    async function open(f) {
        stopPoll();
        form = f;
        document.getElementById('runner-title').textContent = f.name;
        document.getElementById('fields').innerHTML = '<div class="empty"><span class="spinner"></span>&nbsp; Loading&hellip;</div>';
        document.getElementById('output').style.display = 'none';
        document.getElementById('run-status').textContent = '';
        document.getElementById('run-status').className = '';
        document.getElementById('run-btn').disabled = false;
        document.getElementById('runner').style.display = '';
        document.getElementById('runner').scrollIntoView({behavior:'smooth', block:'start'});

        try {
            const f2 = await api('GET', `/api/forms/${f.id}/fields`);
            fields = Array.isArray(f2) ? f2 : [];
            renderFields(fields);
        } catch(e) {
            document.getElementById('fields').innerHTML =
                `<div class="alert alert-warn">Failed to load fields: ${esc(e.message)}</div>`;
        }
    }

    function renderFields(list) {
        const el = document.getElementById('fields');
        if (!list.length) {
            el.innerHTML = '<p style="color:#64748b;font-size:.9rem">No input fields — click Run to execute immediately.</p>';
            return;
        }
        el.innerHTML = list.map(f => {
            const req = f.required ? ' <span style="color:#e11d48">*</span>' : '';
            if (f.field_type === 'bool') {
                return `<div class="field">
                    <div class="field-bool">
                        <input type="checkbox" id="f_${f.id}" ${f.default_value === 'true' ? 'checked' : ''}>
                        <label for="f_${f.id}" style="text-transform:none;font-size:.9rem">${esc(f.label)}${req}</label>
                    </div>
                </div>`;
            }
            if (f.field_type === 'select') {
                let opts = []; try { opts = JSON.parse(f.options); } catch(_){}
                return `<div class="field"><label>${esc(f.label)}${req}</label>
                    <select id="f_${f.id}">${opts.map(o =>
                        `<option value="${esc(o)}"${o===f.default_value?' selected':''}>${esc(o)}</option>`
                    ).join('')}</select></div>`;
            }
            return `<div class="field"><label>${esc(f.label)}${req}</label>
                <input type="${f.field_type === 'number' ? 'number' : 'text'}" id="f_${f.id}"
                    value="${esc(f.default_value)}"${f.required ? ' required' : ''}></div>`;
        }).join('');
    }

    function close() {
        stopPoll();
        document.getElementById('runner').style.display = 'none';
        form = null; fields = [];
    }

    // ── Run ───────────────────────────────────────────────────────────────────

    async function run() {
        if (!form) return;
        stopPoll();

        const vars = {};
        for (const f of fields) {
            const el = document.getElementById('f_' + f.id);
            if (!el) continue;
            const v = f.field_type === 'bool' ? (el.checked ? 'true' : 'false') : el.value.trim();
            if (f.required && v === '') { el.focus(); return; }
            vars[f.name] = v;
        }

        document.getElementById('run-btn').disabled = true;
        setStatus('running', 'Starting\u2026');
        const outEl   = document.getElementById('output');
        const outText = document.getElementById('output-text');
        outEl.style.display = '';
        outText.textContent = '';

        try {
            const res = await api('POST', '/api/runs', {form_id: form.id, variables: vars});
            startPoll(res.run_id);
        } catch(e) {
            setStatus('failed', 'Error: ' + e.message);
            document.getElementById('run-btn').disabled = false;
        }
    }

    function startPoll(runId) {
        poll = setInterval(async () => {
            try {
                const r = await api('GET', `/api/runs/${runId}`);
                document.getElementById('output-text').textContent = stripAnsi(r.output || '');
                document.getElementById('output-status').textContent = r.status;
                setStatus(r.status, r.status.charAt(0).toUpperCase() + r.status.slice(1));
                if (r.status === 'success' || r.status === 'failed') {
                    stopPoll();
                    document.getElementById('run-btn').disabled = false;
                }
            } catch(_) {}
        }, 2000);
    }

    function stopPoll() { if (poll) { clearInterval(poll); poll = null; } }

    function setStatus(state, text) {
        const el = document.getElementById('run-status');
        el.className = 'status s-' + state;
        el.textContent = text;
    }

    function esc(s) {
        return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
    }

    function stripAnsi(s) {
        return s.replace(/\x1b\[[0-9;]*[mGKHF]/g, '');
    }

    <?php if ($configured): ?>loadForms();<?php endif ?>

    return { open, close, run };
})();
</script>
</body>
</html>
