<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Controller\Api;

use App\Plugins\User\src\Middleware\LoginMiddleware;
use App\Plugins\User\src\Models\UsersAuth;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;

#[Middleware(LoginMiddleware::class)]
#[Controller(prefix: '/api/user/setting/authOffline')]
class AuthOfflineApiController
{
    // 设备下线

    #[PostMapping(path: 'check')]
    public function check()
    {
        $token = request()->input('token');
        if (! $token || ! UsersAuth::query()->where(['user_id' => auth()->id(), 'token' => $token])->exists()) {
            return Json_Api(419, false, ['msg' => '无权限']);
        }
        if (cache()->has('plugins.User.auth.offline.ver_code.' . $token)) {
            return Json_Api(200, true, ['msg' => '请输入验证码']);
        }
        $this->send_code($token);
        return Json_Api(200, true, ['msg' => '6位数验证码已发送至你的邮箱,请输入:']);
    }

    #[PostMapping(path: 'verify')]
    public function verify()
    {
        $token = request()->input('token');
        $code = request()->input('code');
        if (! $token || ! UsersAuth::query()->where(['user_id' => auth()->id(), 'token' => $token])->exists()) {
            return Json_Api(419, false, ['msg' => '无权限']);
        }
        if (! $code || (int) $this->ver_code($token) !== (int) $code) {
            return Json_Api(419, false, ['msg' => '验证码错误']);
        }
        UsersAuth::query()->where(['user_id' => auth()->id(), 'token' => $token])->delete();
        return Json_Api(200, true, ['msg' => '操作成功!']);
    }

    // 验证码
    private function ver_code($token)
    {
        if (! cache()->has('plugins.User.auth.offline.ver_code.' . $token)) {
            cache()->set('plugins.User.auth.offline.ver_code.' . $token, random_int(100000, 999999), 600);
        }
        return cache()->get('plugins.User.auth.offline.ver_code.' . $token);
    }

    private function send_code($token)
    {
        $username = auth()->data()->username;
        $email = auth()->data()->email;
        $captcha = $this->ver_code($token);
        $mail = Email();
        go(function () use ($mail, $email, $captcha, $username, $token) {
            $mail->addAddress($email);
            $mail->Subject = '【' . get_options('web_name') . '】请查看你的验证码';
            $mail->Body = <<<HTML
你好 {$username},<br>
Token:{$token},<br>
验证码:{$captcha}<br><br>10分钟内有效
HTML;
            $mail->send();
        });
    }
}
