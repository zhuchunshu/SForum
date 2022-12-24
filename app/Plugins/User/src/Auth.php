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
use App\Plugins\User\src\Lib\UserAuth;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UserClass;
use App\Plugins\User\src\Models\UsersAuth;
use App\Plugins\User\src\Models\UsersOption;
use Hyperf\Utils\Str;
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
        $user = User::query()->find($user_id);

        if (Hash::check($password, $user->password)) {
            $token = Str::random(17);
            session()->set('auth', $token);
            if (! (new UserAuth())->create($user->id, $token)) {
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
        $user = User::query()->find($id);
        $token = Str::random(17);
        session()->set('auth', $token);
        (new UserAuth())->create($user->id, $token);
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
            $token = Str::random(17);
            session()->set('auth', $token);
            (new UserAuth())->create($user->id, $token);
            EventDispatcher()->dispatch(new AfterLogin($user));
            return true;
        }
        return false;
    }

    public function token()
    {
        if ($this->check() === true) {
            return session()->get('auth', null);
        }
        return null;
    }

    public function logout(): bool
    {
        EventDispatcher()->dispatch(new Logout($this->id()));
        (new UserAuth())->destroy_token(session()->get('auth'));
        session()->remove('auth');
        return true;
    }

    public function data(): \Hyperf\Database\Model\Collection|\Hyperf\Database\Model\Model|array|\Hyperf\Database\Model\Builder|null
    {
        return User::query()->find($this->id());
    }

    public function Class(): \Hyperf\Database\Model\Model|\Hyperf\Database\Model\Builder|null
    {
        return UserClass::query()->where('id', auth()->data()->class_id)->first();
    }

    public function Options(): \Hyperf\Database\Model\Model|\Hyperf\Database\Model\Builder|null
    {
        return UsersOption::query()->where('id', auth()->data()->options_id)->first();
    }


    /**
     * get user id.
     * @return int
     */
    public function id()
    {
        return (int) @UsersAuth::query()->where('token', session()->get('auth'))->first('user_id')->user_id;
    }

    /**
     * check is login.
     */
    public function check(): bool
    {
        if (! session()->has('auth')) {
            return false;
        }
        if (User::query()->where('id', $this->id())->count()) {
            return true;
        }

        return false;
    }
}
