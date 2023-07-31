<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Controller\Admin;

use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Core\src\Models\PayAmountRecord;
use App\Plugins\Core\src\Models\PayOrder;
use App\Plugins\Core\src\Models\Post;
use App\Plugins\Core\src\Models\Report;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\User\src\Lib\UserAuth;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UserClass as Uc;
use App\Plugins\User\src\Models\UserFans;
use App\Plugins\User\src\Models\UsersAuth;
use App\Plugins\User\src\Models\UsersCollection;
use App\Plugins\User\src\Models\UsersNotice;
use App\Plugins\User\src\Models\UsersOption;
use App\Plugins\User\src\Models\UsersPm;
use App\Plugins\User\src\Models\UsersSetting;
use App\Plugins\User\src\Models\UserUpload;
use App\Plugins\User\src\Service\Middleware\Oauth2Master;
use App\Plugins\User\src\Service\UserManagement;
use Hyperf\DbConnection\Db;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Utils\Str;
use HyperfExt\Hashing\Hash;
#[Controller]
#[Middleware(\App\Middleware\AdminMiddleware::class)]
class UserController
{
    #[GetMapping('/admin/users')]
    public function index()
    {
        $page = User::query()->with('class')->paginate(15);
        return view('User::Admin.Users.index', ['page' => $page]);
    }
    #[GetMapping('/admin/users/search')]
    public function search()
    {
        $q = request()->input('q');
        $page = User::query()->where('username', 'like', '%' . $q . '%')->with('class')->paginate(15);
        return view('User::Admin.Users.index', ['page' => $page]);
    }
    #[PostMapping('/admin/users/update/username')]
    public function update_username()
    {
        if (!admin_auth()->check()) {
            return Json_Api(419, false, ['msg' => '无权限']);
        }
        $user_id = request()->input('user_id');
        $username = request()->input('username');
        if (!$user_id) {
            return Json_Api(403, false, ['msg' => '请求的用户id不能为空']);
        }
        if (!$username) {
            return Json_Api(403, false, ['msg' => '请求的用户名不能为空']);
        }
        if (!User::query()->where('id', $user_id)->exists()) {
            return Json_Api(403, false, ['msg' => 'ID为:' . $user_id . '的用户不存在']);
        }
        if (User::query()->where('username', $username)->exists()) {
            return Json_Api(403, false, ['msg' => '此用户名已被使用']);
        }
        User::query()->where('id', $user_id)->update(['username' => $username]);
        (new UserAuth())->destroy($user_id);
        return Json_Api(200, true, ['msg' => '修改成功!']);
    }
    #[PostMapping('/admin/users/update/email')]
    public function update_email()
    {
        if (!admin_auth()->check()) {
            return Json_Api(419, false, ['msg' => '无权限']);
        }
        $user_id = request()->input('user_id');
        $email = request()->input('email');
        if (!$user_id) {
            return Json_Api(403, false, ['msg' => '请求的用户id不能为空']);
        }
        if (!$email) {
            return Json_Api(403, false, ['msg' => '请求的邮箱不能为空']);
        }
        if (!User::query()->where('id', $user_id)->exists()) {
            return Json_Api(403, false, ['msg' => 'ID为:' . $user_id . '的用户不存在']);
        }
        if (User::query()->where('email', $email)->exists()) {
            return Json_Api(403, false, ['msg' => '此邮箱已被使用']);
        }
        User::query()->where('id', $user_id)->update(['email' => $email, 'email_ver_time' => null]);
        (new UserAuth())->destroy($user_id);
        return Json_Api(200, true, ['msg' => '修改成功! 用户重新登陆并验证邮箱后生效']);
    }
    #[GetMapping('/admin/users/update/{id}/UserClass')]
    public function update_UserClass_view($id)
    {
        if (!User::query()->where('id', $id)->exists()) {
            return admin_abort('页面不存在', 404);
        }
        $data = User::query()->where('id', $id)->with('Class')->first();
        $class = UC::query()->get();
        return view('User::Admin.Users.update_UserClass', ['data' => $data, 'class' => $class]);
    }
    #[PostMapping('/admin/users/update/UserClass')]
    public function update_UserClass()
    {
        if (!admin_auth()->check()) {
            return Json_Api(419, false, ['msg' => '无权限']);
        }
        $user_id = request()->input('user_id');
        $class_id = request()->input('class_id');
        if (!$user_id || !$class_id) {
            return redirect()->back()->with('danger', '请求参数不完整')->go();
        }
        User::query()->where('id', $user_id)->update(['class_id' => $class_id]);
        (new UserAuth())->destroy($user_id);
        return redirect()->url('/admin/users')->with('success', '修改成功!')->go();
    }
    #[PostMapping('/admin/users/update/token')]
    public function update_Token()
    {
        if (!admin_auth()->check()) {
            return Json_Api(419, false, ['msg' => '无权限']);
        }
        $user_id = request()->input('user_id');
        if (!$user_id) {
            return Json_Api(403, false, ['msg' => '请求参数不完整']);
        }
        User::query()->where('id', $user_id)->update(['_token' => Str::random()]);
        (new UserAuth())->destroy($user_id);
        return Json_Api(200, true, ['msg' => '更新成功!']);
    }
    #[PostMapping('/admin/users/update/password')]
    public function update_password()
    {
        if (!admin_auth()->check()) {
            return Json_Api(419, false, ['msg' => '无权限']);
        }
        $user_id = request()->input('user_id');
        $password = request()->input('password');
        if (!$user_id || !$password) {
            return Json_Api(403, false, ['msg' => '请求参数不完整']);
        }
        User::query()->where('id', $user_id)->update(['password' => Hash::make($password)]);
        (new UserAuth())->destroy($user_id);
        return Json_Api(200, true, ['msg' => '更新成功!']);
    }
    /**
     * 删除用户.
     */
    #[PostMapping('/admin/users/remove')]
    public function remove_user()
    {
        if (!admin_auth()->check()) {
            return Json_Api(419, false, ['msg' => '无权限']);
        }
        $user_id = request()->input('user_id');
        if (!$user_id) {
            return Json_Api(403, false, ['msg' => '请求参数不完整']);
        }
        go(function () use($user_id) {
            // 清理用户帖子数据
            Db::table('topic')->where('user_id', $user_id)->delete();
            // 清理用户评论数据
            Db::table('topic_comment')->where('user_id', $user_id)->delete();
            // 清理用户Posts数据
            Db::table('posts')->where('user_id', $user_id)->delete();
            //清理用户粉丝数据
            UserFans::where('user_id', $user_id)->delete();
            UserFans::where('fans_id', $user_id)->delete();
            // 清理用户设置数据
            UsersSetting::where('user_id', $user_id)->delete();
            // 清理用户通知数据
            UsersNotice::where('user_id', $user_id)->delete();
            // 清理用户收藏数据
            UsersCollection::where('user_id', $user_id)->delete();
            // 清理用户消息数据
            UsersPm::where('from_id', $user_id)->orWhere('to_id', $user_id)->delete();
            // 清理用户options数据
            $options_id = User::find($user_id)->options_id;
            UsersOption::where('id', $options_id)->delete();
            // 清理用户auth数据
            UsersAuth::query()->where('user_id', $user_id)->delete();
            // 删除用户上传的文件
            UserUpload::query()->where('user_id', $user_id)->delete();
            // 删除用户举报
            Report::query()->where('user_id', $user_id)->delete();
            // 删除用户订单
            PayOrder::query()->where('user_id', $user_id)->delete();
            // 删除用户财富记录
            PayAmountRecord::where('user_id', $user_id)->delete();
        });
        User::query()->where('id', $user_id)->delete();
        (new UserAuth())->destroy($user_id);
        return Json_Api(200, true, ['msg' => '已删除!']);
    }
    #[GetMapping('/admin/users/{id}/show')]
    public function show($id)
    {
        if (!User::query()->where('id', $id)->exists()) {
            return redirect()->url('/admin/users')->with('danger', '用户不存在')->go();
        }
        $user = $this->get_user_data($id);
        return view('User::Admin.Users.show', ['user' => $user]);
    }
    #[GetMapping('/admin/users/{id}/edit')]
    public function edit($id)
    {
        if (!User::query()->where('id', $id)->exists()) {
            return redirect()->url('/admin/users')->with('danger', '用户不存在')->go();
        }
        $user = $this->get_user_data($id);
        return view('User::Admin.Users.edit', ['user' => $user]);
    }
    #[PostMapping('/admin/users/{id}/edit')]
    public function edit_submit($id)
    {
        $request = request()->all();
        $request['id'] = $id;
        $handler = function ($request) {
            return redirect()->back()->with('success', '更新成功!')->go();
        };
        // 通过中间件
        $run = $this->throughMiddleware($handler, $this->middlewares());
        return $run($request);
    }
    public function get_user_data($user_id) : \Hyperf\Database\Model\Model|\Hyperf\Database\Model\Builder|null
    {
        return User::query()->with(['Options', 'Class'])->where('id', $user_id)->first();
    }
    /**
     * 通过中间件 through the middleware.
     * @param $handler
     * @param $stack
     * @return \Closure|mixed
     */
    protected function throughMiddleware($handler, $stack) : mixed
    {
        // 闭包实现中间件功能 closures implement middleware functions
        foreach ($stack as $middleware) {
            $handler = function ($request) use($handler, $middleware) {
                if ($middleware instanceof \Closure) {
                    return call_user_func($middleware, $request, $handler);
                }
                return call_user_func([new $middleware(), 'handler'], $request, $handler);
            };
        }
        return $handler;
    }
    private function middlewares() : array
    {
        $_[] = Oauth2Master::class;
        $middlewares = array_merge($_, (new UserManagement())->get_all_handler());
        return array_reverse($middlewares);
    }
}