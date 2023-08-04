<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\CodeFec;

class Captcha
{
    public function check($captcha): bool
    {
        return match (get_options('admin_captcha_service', 'cloudflare')) {
            'cloudflare' => $this->checkCloudflare($captcha),
            'google' => $this->checkGoogle($captcha),
        };
    }

    private function checkCloudflare($captcha): bool
    {
        if (! get_options('admin_captcha_cloudflare_turnstile_website_key') || ! get_options('admin_captcha_cloudflare_turnstile_key')) {
            return true;
        }

        $key = get_options('admin_captcha_cloudflare_turnstile_key');
        $res = http()->post('https://challenges.cloudflare.com/turnstile/v0/siteverify', [
            'secret' => $key,
            'response' => $captcha,
            'remoteip' => get_client_ip(),
        ]);
        return $res['success'];
    }

    private function checkGoogle($captcha)
    {
        if (! get_options('admin_captcha_recaptcha_key')) {
            return true;
        }
        $res = http()->post('https://www.recaptcha.net/recaptcha/api/siteverify', [
            'secret' => get_options('admin_captcha_recaptcha_key'),
            'response' => $captcha,
            'remoteip' => get_client_ip(),
        ]);
        return $res['success'];
    }

}
