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

use App\Middleware\AdminMiddleware;
use App\Plugins\Topic\src\Models\Moderator;
use App\Plugins\Topic\src\Models\TopicTag;
use App\Plugins\User\src\Models\User;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\DeleteMapping;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
#[Controller(prefix: '/admin/users/moderator')]
#[Middleware(AdminMiddleware::class)]
class ModeratorController
{
    #[GetMapping('')]
    public function index()
    {
        $page = Moderator::paginate(15);
        return view('User::Admin.Users.moderator.index', compact('page'));
    }
    #[GetMapping('create')]
    public function create()
    {
        return view('User::Admin.Users.moderator.create');
    }
    #[PostMapping("create")]
    public function store()
    {
        $user_id = request()->input('user_id');
        $tag_id = request()->input('tag_id');
        if (empty($user_id) || empty($tag_id)) {
            return redirect()->back()->with('error', '用户或标签不能为空!')->go();
        }
        if ($user_id === "0") {
            return redirect()->back()->with('error', '用户不能为空!')->go();
        }
        if ($tag_id === "0") {
            return redirect()->back()->with('error', '标签不能为空!')->go();
        }
        if (!User::where('id', $user_id)->exists()) {
            return redirect()->back()->with('error', '用户不存在!')->go();
        }
        if (!TopicTag::where('id', $tag_id)->exists()) {
            return redirect()->back()->with('error', '标签不存在!')->go();
        }
        if (Moderator::where('user_id', $user_id)->where('tag_id', $tag_id)->exists()) {
            return redirect()->back()->with('error', '该用户已经是该标签的版主了!')->go();
        }
        Moderator::create(['user_id' => $user_id, 'tag_id' => $tag_id]);
        return redirect()->url('/admin/users/moderator')->with('success', '版主添加成功!')->go();
    }
    #[DeleteMapping('{id}')]
    public function delete($id) : array
    {
        Moderator::destroy($id);
        return json_api(200, true, ['message' => '删除成功!']);
    }
}