<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Controller;

use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UserClass as UserClassModel;
use App\Plugins\User\src\Models\UserFans;
use App\Plugins\User\src\Models\UsersCollection;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Psr\Http\Message\ResponseInterface;
#[Controller]
class UserController
{
    /**
     * 用户列表.
     */
    #[GetMapping('/users')]
    public function list()
    {
        if (auth()->check()) {
            $count = User::query()->count();
            $page = User::query()->paginate(20);
            return view('User::list', ['page' => $page, 'count' => $count]);
        }
        return admin_abort('登陆后可见', 401, '/login');
    }
    /**
     * 用户信息.
     * @param $username
     * @return ResponseInterface
     */
    #[GetMapping('/users/{id}.html')]
    public function data($id)
    {
        if (!User::query()->where('id', $id)->count()) {
            return admin_abort('页面不存在', 404);
        }
        $user = User::query()->find($id);
        return view('User::data', ['user' => $user]);
    }
    /**
     * 用户信息.
     * @param $username
     * @return ResponseInterface
     */
    #[GetMapping('/users/{username}.username')]
    public function username($username)
    {
        $username = urldecode($username);
        if (!User::query()->where('username', $username)->count()) {
            return admin_abort('页面不存在', 404);
        }
        $user = User::query()->where('username', $username)->first();
        return redirect()->url('/users/' . $user->id . '.html')->go();
    }
    #[GetMapping('/users/group/{id}.html')]
    public function group_data($id) : ResponseInterface
    {
        if (!UserClassModel::query()->where('id', $id)->count()) {
            return admin_abort('页面不存在', 404);
        }
        $userCount = User::query()->where('class_id', $id)->count();
        $data = UserClassModel::query()->where('id', $id)->first();
        $user = User::query()->where('class_id', $id)->paginate(30);
        return view('User::group_data', ['userCount' => $userCount, 'data' => $data, 'user' => $user]);
    }
    // 用户帖子
    #[GetMapping('/users/topic/{username}.html')]
    public function topic($username)
    {
        $username = urldecode($username);
        if (!User::query()->where('username', $username)->count()) {
            return admin_abort('用户名为:' . $username . '的用户不存在');
        }
        $user = User::query()->where('username', $username)->first();
        $page = Topic::query()->where(['user_id' => $user->id])->orderBy('created_at', 'desc')->paginate(15);
        return view('User::topic', ['page' => $page, 'user' => $user]);
    }
    // 用户粉丝
    #[GetMapping('/users/fans/{username}.html')]
    public function fans($username)
    {
        $username = urldecode($username);
        if (!User::query()->where('username', $username)->count()) {
            return admin_abort('用户名为:' . $username . '的用户不存在');
        }
        $user = User::query()->where('username', $username)->first();
        $page = UserFans::query()->where('user_id', $user->id)->with('fans')->paginate(15);
        return view('User::fans', ['page' => $page, 'user' => $user]);
    }
    // 用户收藏
    #[GetMapping('/users/collections/{id}')]
    public function collections($id)
    {
        if (!User::query()->where('id', $id)->exists()) {
            return admin_abort('用户不存在', 404);
        }
        $quanxian = false;
        if (auth()->id() == $id) {
            $quanxian = true;
        }
        $user = User::query()->where('id', $id)->first();
        $page = UsersCollection::query()->where('user_id', $id)->orderBy('id', 'desc')->paginate(15);
        return view('User::Collections', ['page' => $page, 'quanxian' => $quanxian, 'user' => $user]);
    }
}