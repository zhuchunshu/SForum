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
use Hyperf\Utils\ApplicationContext;
use Irooit\Captcha\CaptchaFactory;

#[Controller]
class Captcha
{
    // 获取验证码
    public function get()
    {
        return '/captcha';
    }

    #[GetMapping(path: '/captcha')]
    #[RateLimit(create: 1, capacity: 1, consume: 1)]
    public function build()
    {
        $captchaFactory = ApplicationContext::getContainer()->get(CaptchaFactory::class);
        $captcha = $captchaFactory->make('default',true);
        $image = $captcha['img'];
        session()->set('captcha', $captcha['key']);
        $image = base64_decode(explode('base64,', $image)[1]);
        return response()->raw($image)->withHeader('Content-Type', 'image/png');
    }

    public function inline(): string
    {
        $captchaFactory = ApplicationContext::getContainer()->get(CaptchaFactory::class);
        $captcha = $captchaFactory->make('default',true);
        session()->set('captcha', $captcha['key']);
        return $captcha['img'];
    }

    public function check($captcha): bool
    {
        $captchaFactory = ApplicationContext::getContainer()->get(CaptchaFactory::class);
        return $captchaFactory->validate($captcha, session()->get('captcha'));
    }
}
