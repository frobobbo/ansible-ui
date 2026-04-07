<?php

class Modules_AutomationHub_CustomButtons extends pm_Hook_CustomButtons
{
    public function getButtons()
    {
        pm_Context::init('automation-hub');

        return [[
            'place' => [
                self::PLACE_ADMIN_NAVIGATION,
                self::PLACE_HOSTING_PANEL_NAVIGATION,
            ],
            'section' => self::SECTION_NAV_ADDITIONAL,
            'title' => 'Automation Hub',
            'description' => 'Open Automation Hub inside the Plesk panel.',
            'icon' => pm_Context::getBaseUrl() . 'images/automation-hub.svg',
            'link' => pm_Context::getActionUrl('index', 'index'),
            'contextParams' => true,
        ]];
    }
}
