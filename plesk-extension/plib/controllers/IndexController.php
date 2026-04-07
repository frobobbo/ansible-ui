<?php

require_once dirname(__DIR__) . '/library/AutomationHub/Client.php';

class IndexController extends pm_Controller_Action
{
    public function init()
    {
        parent::init();

        $this->view->pageTitle = '';
        $context = (string) $this->getRequest()->getParam('context', '');
        $params = [];
        if ($context !== '') {
            $params['context'] = $context;
        }

        $this->view->indexUrl = pm_Context::getActionUrl('index', 'index');
        $this->view->settingsUrl = pm_Context::getActionUrl('index', 'settings');
        if ($params) {
            $this->view->indexUrl .= '?' . http_build_query($params);
            $this->view->settingsUrl .= '?' . http_build_query($params);
        }
        $this->view->contextParams = $params;
    }

    public function indexAction()
    {
        $this->view->configured = pm_Settings::get('hub_url', '') !== ''
            && pm_Settings::get('hub_username', '') !== '';
        $this->view->hubBaseUrl = rtrim((string) pm_Settings::get('hub_url', ''), '/');
        $this->view->proxyUrl = pm_Context::getBaseUrl() . 'proxy.php';
    }

    public function settingsAction()
    {
        if ($this->getRequest()->isPost()) {
            $url = trim((string) $this->getRequest()->getPost('hub_url', ''));
            $username = trim((string) $this->getRequest()->getPost('hub_username', ''));
            $password = (string) $this->getRequest()->getPost('hub_password', '');

            pm_Settings::set('hub_url', $url);
            pm_Settings::set('hub_username', $username);
            if ($password !== '') {
                pm_Settings::set('hub_password', $password);
            }

            pm_Settings::set('hub_token_cache', '');
            pm_Settings::set('hub_token_expiry', '0');

            $redirectParams = ['saved' => 1] + $this->view->contextParams;
            $this->_helper->redirector->gotoSimple('settings', 'index', null, $redirectParams);
            return;
        }

        $this->view->hubUrl = (string) pm_Settings::get('hub_url', '');
        $this->view->hubUsername = (string) pm_Settings::get('hub_username', '');
        $this->view->hasPassword = pm_Settings::get('hub_password', '') !== '';
        $this->view->saved = (bool) $this->getRequest()->getParam('saved', false);
        $this->view->testUrl = pm_Context::getBaseUrl() . 'settings.php?action=test';
    }
}
