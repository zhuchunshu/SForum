<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Lib;

use App\Plugins\User\src\Models\User;

class UserVerEmail
{
    public function make()
    {
        $id = auth()->data()->id;
        cache()->set('core.user.ver.email.' . $id, random_int(100000, 999999), 1800);
        cache()->set('core.user.ver.email.time.' . $id, time() + 60);
        return cache()->get('core.user.ver.email.' . $id);
    }

    public function check($captcha): bool
    {
        $id = auth()->data()->id;
        if (! cache()->has('core.user.ver.email.' . $id)) {
            return false;
        }
        $data = cache()->get('core.user.ver.email.' . $id);
        return $captcha === $data;
    }

    public function ifsend(): bool
    {
        $id = auth()->id();
        if (! cache()->has('core.user.ver.email.time.' . $id)) {
            return true;
        }
        if (cache()->get('core.user.ver.email.time.' . $id, 0) - time() <= 0) {
            return true;
        }
        return false;
    }

    public function sendTime(): int
    {
        $id = auth()->id();
        return cache()->get('core.user.ver.email.time.' . $id, 0) - time();
    }

    public function send($id)
    {
        $data = User::query()->where('id', $id)->first();
        $username = $data->username;
        $email = $data->email;
        $captcha = $this->make();
        $mail = Email();
        go(function () use ($mail, $email, $captcha, $username) {
            $mail->addAddress($email);
            $mail->Subject = '【' . get_options('web_name') . '】请查看你的邮箱验证码';
            $mail->Body = <<<HTML
你好 {$username},<br>
你的邮箱验证码是:{$captcha}
HTML;
            $mail->send();
        });
        return $captcha;
    }
}
