<?php

/**
 * Automation Hub API client.
 *
 * Reads connection details from pm_Settings and caches the JWT so we only
 * re-authenticate when the cached token is about to expire.
 */
class AutomationHub_Client
{
    private string $baseUrl;

    public function __construct()
    {
        $this->baseUrl = rtrim((string) pm_Settings::get('hub_url', ''), '/');
    }

    // ── Configuration ────────────────────────────────────────────────────────

    public function isConfigured(): bool
    {
        return $this->baseUrl !== ''
            && pm_Settings::get('hub_username', '') !== '';
    }

    // ── Token management ─────────────────────────────────────────────────────

    private function getToken(): string
    {
        $cached = (string) pm_Settings::get('hub_token_cache', '');
        $expiry = (int) pm_Settings::get('hub_token_expiry', '0');

        // Reuse cached token if it still has more than 60 seconds left
        if ($cached !== '' && $expiry > time() + 60) {
            return $cached;
        }

        $resp = $this->rawCall('POST', '/api/auth/login', [
            'username' => pm_Settings::get('hub_username', ''),
            'password' => pm_Settings::get('hub_password', ''),
        ]);

        $token = $resp['token'] ?? '';
        if ($token !== '') {
            // Cache for 23 h (assumes 24 h JWT lifetime)
            pm_Settings::set('hub_token_cache', $token);
            pm_Settings::set('hub_token_expiry', (string) (time() + 23 * 3600));
        }

        return $token;
    }

    public function clearTokenCache(): void
    {
        pm_Settings::set('hub_token_cache', '');
        pm_Settings::set('hub_token_expiry', '0');
    }

    // ── HTTP helpers ─────────────────────────────────────────────────────────

    /**
     * Make an authenticated call to the Automation Hub API.
     */
    public function call(string $method, string $path, ?array $body = null): array
    {
        return $this->rawCall($method, $path, $body, $this->getToken());
    }

    /**
     * Test connectivity by hitting /healthz (no auth needed).
     *
     * @throws RuntimeException on failure
     */
    public function testConnection(): array
    {
        return $this->rawCall('GET', '/healthz');
    }

    private function rawCall(string $method, string $path, ?array $body = null, string $token = ''): array
    {
        $url = $this->baseUrl . $path;
        $ch  = curl_init($url);

        $headers = ['Content-Type: application/json', 'Accept: application/json'];
        if ($token !== '') {
            $headers[] = 'Authorization: Bearer ' . $token;
        }

        curl_setopt_array($ch, [
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_CUSTOMREQUEST  => strtoupper($method),
            CURLOPT_HTTPHEADER     => $headers,
            CURLOPT_TIMEOUT        => 30,
            CURLOPT_SSL_VERIFYPEER => true,
        ]);

        if ($body !== null) {
            curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($body));
        }

        $raw    = curl_exec($ch);
        $status = (int) curl_getinfo($ch, CURLINFO_HTTP_CODE);
        $err    = curl_error($ch);
        curl_close($ch);

        if ($raw === false) {
            throw new RuntimeException('Connection failed: ' . $err);
        }

        $data = json_decode($raw, true);
        if (!is_array($data)) {
            throw new RuntimeException("Non-JSON response (HTTP $status)");
        }

        if ($status >= 400) {
            throw new RuntimeException($data['error'] ?? "HTTP $status");
        }

        return $data;
    }
}
