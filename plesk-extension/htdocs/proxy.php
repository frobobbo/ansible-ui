<?php
/**
 * AJAX proxy — forwards whitelisted API calls to Automation Hub.
 * The JWT token never leaves the server.
 */
pm_Context::init('automation-hub');
require_once __DIR__ . '/../plib/library/AutomationHub/Client.php';

header('Content-Type: application/json');

$input    = json_decode(file_get_contents('php://input'), true) ?? [];
$endpoint = trim((string) ($input['endpoint'] ?? ''));
$method   = strtoupper((string) ($input['method']   ?? 'GET'));
$body     = (isset($input['body']) && is_array($input['body'])) ? $input['body'] : null;

// Whitelisted endpoints only
$allowed = [
    '~^/api/forms$~',
    '~^/api/forms/[0-9a-f-]+/fields$~',
    '~^/api/runs$~',
    '~^/api/runs/[0-9a-f-]+$~',
];
$permitted = false;
foreach ($allowed as $p) {
    if (preg_match($p, $endpoint)) { $permitted = true; break; }
}
if (!$permitted) {
    http_response_code(403);
    echo json_encode(['error' => 'Endpoint not permitted.']);
    exit;
}

$client = new AutomationHub_Client();
if (!$client->isConfigured()) {
    http_response_code(503);
    echo json_encode(['error' => 'Extension is not configured.']);
    exit;
}

try {
    echo json_encode($client->call($method, $endpoint, $body));
} catch (RuntimeException $e) {
    http_response_code(502);
    echo json_encode(['error' => $e->getMessage()]);
}
