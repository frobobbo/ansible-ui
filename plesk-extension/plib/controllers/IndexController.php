<?php

class IndexController extends pm_Controller_Action
{
    private AutomationHub_Client $client;

    public function init(): void
    {
        parent::init();
        require_once pm_Context::get()->getLibDir() . '/AutomationHub/Client.php';
        $this->client = new AutomationHub_Client();
    }

    // ── Main page ─────────────────────────────────────────────────────────────

    public function indexAction(): void
    {
        $this->view->notConfigured = !$this->client->isConfigured();
        $this->view->settingsUrl   = pm_Context::get()->getActionUrl('settings', 'index');
        $this->view->proxyUrl      = pm_Context::get()->getActionUrl('index', 'proxy');
    }

    // ── AJAX proxy ────────────────────────────────────────────────────────────
    //
    // The frontend never calls Automation Hub directly.  All API calls are
    // routed through here so the JWT token stays server-side.

    public function proxyAction(): void
    {
        $this->_helper->viewRenderer->setNoRender(true);
        $this->_response->setHeader('Content-Type', 'application/json', true);

        if (!$this->client->isConfigured()) {
            echo json_encode(['error' => 'Extension is not configured.']);
            return;
        }

        $input    = json_decode(file_get_contents('php://input'), true) ?? [];
        $endpoint = trim((string) ($input['endpoint'] ?? ''));
        $method   = strtoupper((string) ($input['method'] ?? 'GET'));
        $body     = isset($input['body']) && is_array($input['body']) ? $input['body'] : null;

        // Whitelist of permitted endpoints (UUID pattern: [0-9a-f-]+)
        $allowed = [
            '~^/api/forms$~',
            '~^/api/forms/[0-9a-f-]+/fields$~',
            '~^/api/runs$~',
            '~^/api/runs/[0-9a-f-]+$~',
        ];

        $permitted = false;
        foreach ($allowed as $pattern) {
            if (preg_match($pattern, $endpoint)) {
                $permitted = true;
                break;
            }
        }

        if (!$permitted) {
            http_response_code(403);
            echo json_encode(['error' => 'Endpoint not permitted.']);
            return;
        }

        try {
            $result = $this->client->call($method, $endpoint, $body);
            echo json_encode($result);
        } catch (RuntimeException $e) {
            http_response_code(502);
            echo json_encode(['error' => $e->getMessage()]);
        }
    }
}
