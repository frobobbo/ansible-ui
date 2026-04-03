<?php

class SettingsController extends pm_Controller_Action
{
    private AutomationHub_Client $client;

    public function init(): void
    {
        parent::init();
        require_once pm_Context::get()->getLibDir() . '/AutomationHub/Client.php';
        $this->client = new AutomationHub_Client();
    }

    // ── Settings form ─────────────────────────────────────────────────────────

    public function indexAction(): void
    {
        $this->view->mainUrl  = pm_Context::get()->getActionUrl('index', 'index');
        $this->view->saveUrl  = pm_Context::get()->getActionUrl('settings', 'save');
        $this->view->testUrl  = pm_Context::get()->getActionUrl('settings', 'test');

        $this->view->hubUrl      = pm_Settings::get('hub_url', '');
        $this->view->hubUsername = pm_Settings::get('hub_username', '');
        // Never send the password back to the browser — just indicate if one is set
        $this->view->hasPassword = pm_Settings::get('hub_password', '') !== '';
    }

    // ── Save ──────────────────────────────────────────────────────────────────

    public function saveAction(): void
    {
        $url      = trim((string) $this->getRequest()->getPost('hub_url', ''));
        $username = trim((string) $this->getRequest()->getPost('hub_username', ''));
        $password = (string) $this->getRequest()->getPost('hub_password', '');

        // Only update password if a new one was entered
        if ($password !== '') {
            pm_Settings::set('hub_password', $password);
        }

        pm_Settings::set('hub_url', $url);
        pm_Settings::set('hub_username', $username);

        // Clear cached token so the next request re-authenticates
        $this->client->clearTokenCache();

        $this->_redirect(pm_Context::get()->getActionUrl('settings', 'index'));
    }

    // ── Test connection (AJAX) ────────────────────────────────────────────────

    public function testAction(): void
    {
        $this->_helper->viewRenderer->setNoRender(true);
        $this->_response->setHeader('Content-Type', 'application/json', true);

        // Allow testing with in-form credentials before saving
        $input    = json_decode(file_get_contents('php://input'), true) ?? [];
        $testUrl  = trim((string) ($input['hub_url']      ?? pm_Settings::get('hub_url', '')));
        $username = trim((string) ($input['hub_username'] ?? pm_Settings::get('hub_username', '')));
        $password = (string) ($input['hub_password'] ?? '');

        if ($password === '') {
            $password = (string) pm_Settings::get('hub_password', '');
        }

        if ($testUrl === '' || $username === '') {
            echo json_encode(['ok' => false, 'message' => 'URL and username are required.']);
            return;
        }

        // Temporarily override settings for the test
        $original = [
            'hub_url'      => pm_Settings::get('hub_url', ''),
            'hub_username' => pm_Settings::get('hub_username', ''),
            'hub_password' => pm_Settings::get('hub_password', ''),
        ];
        pm_Settings::set('hub_url',      $testUrl);
        pm_Settings::set('hub_username', $username);
        pm_Settings::set('hub_password', $password);
        $this->client->clearTokenCache();

        try {
            // Re-create client with updated settings
            require_once pm_Context::get()->getLibDir() . '/AutomationHub/Client.php';
            $testClient = new AutomationHub_Client();
            $testClient->testConnection();
            // Try to authenticate as a real test
            $testClient->call('GET', '/api/forms');
            echo json_encode(['ok' => true, 'message' => 'Connection successful.']);
        } catch (RuntimeException $e) {
            echo json_encode(['ok' => false, 'message' => $e->getMessage()]);
        } finally {
            // Restore original settings
            pm_Settings::set('hub_url',      $original['hub_url']);
            pm_Settings::set('hub_username', $original['hub_username']);
            pm_Settings::set('hub_password', $original['hub_password']);
            $this->client->clearTokenCache();
        }
    }
}
