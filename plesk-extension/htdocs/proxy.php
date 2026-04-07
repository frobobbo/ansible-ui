<?php
/**
 * AJAX proxy — forwards whitelisted API calls to Automation Hub.
 * The JWT token never leaves the server.
 */
pm_Context::init('automation-hub');
require_once rtrim(pm_Context::getPlibDir(), '/\\') . '/library/AutomationHub/Client.php';

$client = new AutomationHub_Client();
if (!$client->isConfigured()) {
    http_response_code(503);
    header('Content-Type: application/json');
    echo json_encode(['error' => 'Extension is not configured.']);
    exit;
}

if (($_GET['action'] ?? '') === 'stream') {
    $runId = trim((string) ($_GET['run_id'] ?? ''));
    if (!preg_match('/^[0-9a-f-]+$/i', $runId)) {
        http_response_code(400);
        header('Content-Type: application/json');
        echo json_encode(['error' => 'Invalid run id.']);
        exit;
    }

    header('Content-Type: text/event-stream');
    header('Cache-Control: no-cache');
    header('Connection: keep-alive');
    header('X-Accel-Buffering: no');

    $streamUrl = rtrim((string) pm_Settings::get('hub_url', ''), '/')
        . '/api/runs/' . rawurlencode($runId)
        . '/stream?token=' . rawurlencode($client->getAuthToken());

    $ch = curl_init($streamUrl);
    curl_setopt_array($ch, [
        CURLOPT_RETURNTRANSFER => false,
        CURLOPT_FOLLOWLOCATION => true,
        CURLOPT_HTTPGET        => true,
        CURLOPT_TIMEOUT        => 0,
        CURLOPT_SSL_VERIFYPEER => true,
        CURLOPT_WRITEFUNCTION  => static function ($curl, $chunk) {
            echo $chunk;
            if (function_exists('ob_flush')) {
                @ob_flush();
            }
            flush();
            return strlen($chunk);
        },
    ]);

    $ok = curl_exec($ch);
    if ($ok === false) {
        $msg = json_encode(['error' => 'Stream failed: ' . curl_error($ch)]);
        echo "event: done\ndata: failed\n\n";
        echo "event: proxy-error\ndata: {$msg}\n\n";
        if (function_exists('ob_flush')) {
            @ob_flush();
        }
        flush();
    }
    curl_close($ch);
    exit;
}

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

try {
    echo json_encode($client->call($method, $endpoint, $body));
} catch (RuntimeException $e) {
    http_response_code(502);
    echo json_encode(['error' => $e->getMessage()]);
}
