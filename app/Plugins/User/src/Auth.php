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

use App\Plugins\User\src\Event\AfterLogin;
use App\Plugins\User\src\Event\Logout;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UserClass;
use App\Plugins\User\src\Models\UsersAuth;
use App\Plugins\User\src\Models\UsersOption;
use HyperfExt\Hashing\Hash;

class Auth
{
    public function SignIn(string $email, string $password): bool
    {
        if (! User::query()->where('email', $email)->count()) {
            return false;
        }
        // 数据库里的密码
        $user_id = User::query()->where('email', $email)->first()->id;
        $user = User::find($user_id);

        if (Hash::check($password, $user->password)) {
            if (! authManager()->login($user)) {
                return false;
            }
            EventDispatcher()->dispatch(new AfterLogin($user));
            return true;
        }
        return false;
    }

    // 刷新登陆
    public function refresh(int $id): bool
    {
        if (! User::query()->where('id', $id)->count()) {
            return false;
        }
        // 数据库里的密码
        $user = User::find($id);
        authManager()->login($user);
        EventDispatcher()->dispatch(new AfterLogin($user));
        return true;
    }

    public function SignInUsername(string $username, string $password): bool
    {
        if (! User::query()->where('username', $username)->count()) {
            return false;
        }
        // 数据库里的密码
        $user = User::query()->where('username', $username)->first();
        if (Hash::check($password, $user->password)) {
            $user_id = $user->id;
            if (! authManager()->login(User::find($user_id))) {
                return false;
            }
            EventDispatcher()->dispatch(new AfterLogin($user));
            return true;
        }
        return false;
    }

    public function token()
    {
        if ($this->check() === true) {
            return session()->get('AUTH_TOKEN', null);
        }
        return null;
    }

    public function logout(): bool
    {
        EventDispatcher()->dispatch(new Logout($this->id()));
        authManager()->logout();
        return true;
    }

    public function data(): \Hyperf\Database\Model\Collection | \Hyperf\Database\Model\Model | array | \Hyperf\Database\Model\Builder | null
    {
        return authManager()->user();
    }

    public function Class(): \Hyperf\Database\Model\Model | \Hyperf\Database\Model\Builder | null
    {
        return UserClass::query()->where('id', auth()->data()->class_id)->first();
    }

    public function Options(): \Hyperf\Database\Model\Model | \Hyperf\Database\Model\Builder | null
    {
        return UsersOption::query()->where('id', auth()->data()->options_id)->first();
    }

    /**
     * get user id.
     * @return int
     */
    public function id()
    {
        if ($this->check()) {
            return (int) authManager()->id();
        }
        return null;
    }

    /**
     * check is login.
     * @param null|mixed $token
     */
    public function check(string $token = null): bool
    {
        if ($token === null) {
            return authManager()->check($token);
        }
        return UsersAuth::query()->where([
            'user_id' => authManager()->user()->getId(),
            'token' => $token,
            'user_agent' => get_user_agent(),
        ])->exists();
    }
}
