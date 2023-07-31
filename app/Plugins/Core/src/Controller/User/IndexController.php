<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Controller\User;

use App\Plugins\User\src\Middleware\LoginMiddleware;
use App\Plugins\User\src\Models\User;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\HttpServer\Request;
use Psr\Http\Message\ResponseInterface;
#[Controller]
#[Middleware(LoginMiddleware::class)]
class IndexController
{
    public \Psr\Log\LoggerInterface $logger;
    /**
     * 强制验证邮箱.
     */
    #[GetMapping('/user/ver_email')]
    public function user_ver_email()
    {
        if (User::query()->where('id', auth()->id())->value('email_ver_time')) {
            return redirect()->url('/')->with('info', '你已验证邮箱,无需重复操作')->go();
        }
        return view('App::user.ver_email');
    }
    #[PostMapping('/user/ver_email')]
    public function user_ver_email_post()
    {
        $send = request()->input('send', null);
        $captcha = request()->input('captcha', null);
        if ($send === 'send') {
            if (!core_user_ver_email()->ifsend()) {
                return redirect()->back()->with('danger', '冷却期间,请' . core_user_ver_email()->sendTime() . '秒后再试')->go();
            }
            core_user_ver_email()->send(auth()->id());
            return redirect()->back()->with('success', '验证码邮件已发送')->go();
        }
        if (!$captcha) {
            return redirect()->back()->with('danger', '请填写验证码')->go();
        }
        if (!core_user_ver_email()->check($captcha)) {
            return redirect()->back()->with('danger', '验证码错误')->go();
        }
        User::query()->where('id', auth()->id())->update(['email_ver_time' => date('Y-m-d H:i:s')]);
        return redirect()->url('/')->with('success', '验证通过')->go();
    }
    /**
     * 个人中心.
     */
    #[GetMapping('/user')]
    public function user()
    {
        return redirect()->url('/users/' . auth()->id() . '.html')->go();
    }
    // 个人通知
    #[GetMapping('/user/notice')]
    public function notice() : ResponseInterface
    {
        return redirect()->url('/notice')->go();
    }
    // 个人收藏
    #[GetMapping('/user/collections')]
    public function collections()
    {
        return redirect()->url('/users/collections/' . auth()->id())->go();
    }
}