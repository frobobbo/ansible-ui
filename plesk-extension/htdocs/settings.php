<?php
pm_Context::init('automation-hub');
require_once __DIR__ . '/../plib/library/AutomationHub/Client.php';

// Handle test connection (AJAX POST with ?action=test)
if ($_SERVER['REQUEST_METHOD'] === 'POST' && ($_GET['action'] ?? '') === 'test') {
    header('Content-Type: application/json');
    $input    = json_decode(file_get_contents('php://input'), true) ?? [];
    $testUrl  = trim((string) ($input['hub_url']      ?? ''));
    $username = trim((string) ($input['hub_username'] ?? ''));
    $password = (string) ($input['hub_password'] ?? '');
    if ($password === '') $password = (string) pm_Settings::get('hub_password', '');

    if ($testUrl === '' || $username === '') {
        echo json_encode(['ok' => false, 'message' => 'URL and username are required.']);
        exit;
    }
    // Temporarily save test credentials
    $orig = [
        'hub_url'      => pm_Settings::get('hub_url', ''),
        'hub_username' => pm_Settings::get('hub_username', ''),
        'hub_password' => pm_Settings::get('hub_password', ''),
    ];
    pm_Settings::set('hub_url',      $testUrl);
    pm_Settings::set('hub_username', $username);
    pm_Settings::set('hub_password', $password);
    pm_Settings::set('hub_token_cache',  '');
    pm_Settings::set('hub_token_expiry', '0');
    try {
        $c = new AutomationHub_Client();
        $c->call('GET', '/api/forms');
        echo json_encode(['ok' => true, 'message' => 'Connection successful.']);
    } catch (RuntimeException $e) {
        echo json_encode(['ok' => false, 'message' => $e->getMessage()]);
    } finally {
        pm_Settings::set('hub_url',      $orig['hub_url']);
        pm_Settings::set('hub_username', $orig['hub_username']);
        pm_Settings::set('hub_password', $orig['hub_password']);
        pm_Settings::set('hub_token_cache',  '');
        pm_Settings::set('hub_token_expiry', '0');
    }
    exit;
}

// Handle save (POST)
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $url      = trim((string) ($_POST['hub_url']      ?? ''));
    $username = trim((string) ($_POST['hub_username'] ?? ''));
    $password = (string) ($_POST['hub_password'] ?? '');
    pm_Settings::set('hub_url',      $url);
    pm_Settings::set('hub_username', $username);
    if ($password !== '') pm_Settings::set('hub_password', $password);
    pm_Settings::set('hub_token_cache',  '');
    pm_Settings::set('hub_token_expiry', '0');
    header('Location: settings.php?saved=1');
    exit;
}

$hubUrl      = (string) pm_Settings::get('hub_url', '');
$hubUsername = (string) pm_Settings::get('hub_username', '');
$hasPassword = pm_Settings::get('hub_password', '') !== '';
$saved       = isset($_GET['saved']);
?>
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Automation Hub — Settings</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#f1f5f9;color:#1e293b;padding:2rem}
.wrap{max-width:560px;margin:0 auto}
.topbar{display:flex;align-items:center;gap:.75rem;margin-bottom:1.5rem}
.topbar h1{font-size:1.2rem;font-weight:700}
.back{color:#0891b2;text-decoration:none;font-size:.875rem}
.back:hover{text-decoration:underline}
.card{background:#fff;border:1px solid #e2e8f0;border-radius:10px;padding:1.5rem}
.group{margin-bottom:1.25rem}
.group label{display:block;font-size:.75rem;font-weight:700;text-transform:uppercase;
    letter-spacing:.06em;color:#475569;margin-bottom:.4rem}
.group input{width:100%;padding:.55rem .75rem;border:1px solid #cbd5e1;border-radius:6px;
    font-size:.9rem;background:#f8fafc}
.group input:focus{outline:2px solid #0891b2;border-color:transparent;background:#fff}
.hint{font-size:.78rem;color:#64748b;margin-top:.35rem}
.actions{display:flex;gap:.75rem;margin-top:1.5rem;align-items:center;flex-wrap:wrap}
.btn{padding:.5rem 1.1rem;border-radius:6px;border:1px solid #cbd5e1;background:#fff;
    color:#334155;font-size:.875rem;cursor:pointer;transition:all .15s}
.btn:hover{background:#f1f5f9}
.btn-primary{background:#0891b2;border-color:#0891b2;color:#fff}
.btn-primary:hover{background:#0e7490}
.result{font-size:.85rem;padding:.35rem .75rem;border-radius:6px}
.ok{background:#dcfce7;color:#166534}.err{background:#fee2e2;color:#991b1b}
.alert-success{background:#dcfce7;border:1px solid #86efac;color:#166534;
    padding:.75rem 1rem;border-radius:8px;margin-bottom:1rem;font-size:.9rem}
</style>
</head>
<body>
<div class="wrap">
    <div class="topbar">
        <a href="index.php" class="back">&#8592; Back</a>
        <h1>Automation Hub — Settings</h1>
    </div>

    <?php if ($saved): ?>
        <div class="alert-success">Settings saved.</div>
    <?php endif ?>

    <form method="post" class="card">
        <div class="group">
            <label>Automation Hub URL</label>
            <input type="url" name="hub_url"
                value="<?= htmlspecialchars($hubUrl) ?>"
                placeholder="https://ansible.johnsons.casa" required>
            <p class="hint">Base URL of your Automation Hub installation (no trailing slash).</p>
        </div>
        <div class="group">
            <label>Username</label>
            <input type="text" name="hub_username"
                value="<?= htmlspecialchars($hubUsername) ?>"
                placeholder="admin" required autocomplete="off">
        </div>
        <div class="group">
            <label>Password<?= $hasPassword ? ' &nbsp;<span style="font-weight:400;color:#94a3b8">(leave blank to keep existing)</span>' : '' ?></label>
            <input type="password" name="hub_password"
                placeholder="<?= $hasPassword ? '••••••••' : 'Enter password' ?>"
                autocomplete="new-password">
        </div>
        <div class="actions">
            <button type="submit" class="btn btn-primary">Save</button>
            <button type="button" class="btn" onclick="testConn()">Test Connection</button>
            <span id="result" class="result" style="display:none"></span>
        </div>
    </form>
</div>
<script>
async function testConn() {
    const r = document.getElementById('result');
    r.className = 'result'; r.style.display = ''; r.textContent = 'Testing…';
    try {
        const res = await fetch('settings.php?action=test', {
            method: 'POST',
            headers: {'Content-Type':'application/json'},
            body: JSON.stringify({
                hub_url:      document.querySelector('[name=hub_url]').value.trim(),
                hub_username: document.querySelector('[name=hub_username]').value.trim(),
                hub_password: document.querySelector('[name=hub_password]').value,
            })
        });
        const d = await res.json();
        r.className = 'result ' + (d.ok ? 'ok' : 'err');
        r.textContent = d.message;
    } catch(e) {
        r.className = 'result err'; r.textContent = 'Request failed: ' + e.message;
    }
}
</script>
</body>
</html>
