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

use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\RateLimit\Annotation\RateLimit;

#[Controller]
class Captcha
{
    // 获取验证码
    public function get()
    {
        return '/captcha';
    }

    #[GetMapping('/captcha')]
    #[RateLimit(create: 1, capacity: 1, consume: 1)]
    public function build()
    {
    }

    public function inline(): string
    {
        return '';
    }

    public function check($captcha): bool
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
}
