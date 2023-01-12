<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\User\src;

use App\Plugins\User\src\Models\UsersAuth;
use Hyperf\Context\Context;
use Hyperf\Utils\Str;
use Qbhy\HyperfAuth\Authenticatable;
use Qbhy\HyperfAuth\Exception\AuthException;
use Qbhy\HyperfAuth\Guard\SessionGuard;

class AuthGuard extends SessionGuard
{
    public function login(Authenticatable $user): bool
    {
        $this->session->put($this->sessionKey(), $user->getId());

        $token = Str::random(17);
        UsersAuth::query()->create([
            'user_id' => $user->getId(),
            'token' => $token,
            'user_ip' => get_client_ip(),
            'user_agent' => get_user_agent(),
        ]);
        $this->deleteAuth($user->getId());
        session()->set('AUTH_TOKEN', $token);
        Context::set($this->resultKey(), $user);
        return true;
    }

    public function logout(): bool
    {
        \Hyperf\Context\Context::set($this->resultKey(), null);
        UsersAuth::query()->where([
            'user_id' => $this->user()->getId(),
            'token' => session()->get('AUTH_TOKEN'),
        ])->delete();
        return (bool) $this->session->remove($this->sessionKey()) && $this->session->remove('AUTH_TOKEN');
    }

    public function check(): bool
    {
        try {
            return $this->user() instanceof Authenticatable && call_user_func(function () {
                return UsersAuth::query()->where([
                    'user_id' => $this->user()->getId(),
                    'token' => session()->get('AUTH_TOKEN'),
                    'user_agent' => get_user_agent(),
                ])->exists();
            }) === true;
        } catch (AuthException $exception) {
            return false;
        }
    }

    private function deleteAuth($user_id)
    {
        $protecteds = UsersAuth::query()->where('user_id', $user_id)->orderByDesc('id')->take(get_options('core_user_session_num', 10))->get(['id']);
        $_protected = [];
        foreach ($protecteds as $protected) {
            $_protected[] = ['id', '!=', $protected->id];
        }
        UsersAuth::query()->where('user_id', $user_id)->where($_protected)->delete();
    }

    public function guest(): bool
    {
        return ! $this->check();
    }

}
