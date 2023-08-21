<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Controllers\Admin;

use App\Middleware\AdminMiddleware;
use App\Plugins\Core\src\Handler\AvatarUpload;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\Topic\src\Models\TopicTag;
use App\Plugins\Topic\src\Requests\CreateTagRequest;
use App\Plugins\Topic\src\Requests\EditTagRequest;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
#[Middleware(AdminMiddleware::class)]
#[Controller]
class TagController
{
    #[GetMapping('/admin/topic/tag/create')]
    public function create()
    {
        $userClass = \App\Plugins\User\src\Models\UserClass::query()->get();
        return view('Topic::Tag.create', ['userClass' => $userClass]);
    }
    #[PostMapping('/admin/topic/tag/create')]
    public function create_store(CreateTagRequest $request)
    {
        if (!admin_auth()->check()) {
            return Json_Api(419, false, ['msg' => '无权限']);
        }
        $name = $request->input('name');
        $color = $request->input('color');
        $description = $request->input('description');
        $icon = $request->input('icon');
        $userClass = $request->input('userClass');
        if ($userClass) {
            $userClass = json_encode($userClass, JSON_THROW_ON_ERROR, JSON_UNESCAPED_UNICODE);
        } else {
            $userClass = null;
        }
        TopicTag::create(['name' => $name, 'color' => $color, 'description' => $description, 'icon' => $icon, 'userClass' => $userClass]);
        return redirect()->url('/admin/topic/tag')->with('success', '创建成功!')->go();
    }
    #[GetMapping('/admin/topic/tag')]
    public function index() : \Psr\Http\Message\ResponseInterface
    {
        $page = TopicTag::query()->paginate(15);
        return view('Topic::Tag.index', ['page' => $page]);
    }
    #[GetMapping('/admin/topic/tag/edit/{id}')]
    public function edit($id)
    {
        if (!TopicTag::query()->where('id', $id)->count()) {
            return admin_abort('id为' . $id . '的板块不存在', 403);
        }
        $data = TopicTag::query()->where('id', $id)->first();
        $userClass = \App\Plugins\User\src\Models\UserClass::query()->get();
        return view('Topic::Tag.edit', ['data' => $data, 'userClass' => $userClass]);
    }
    #[PostMapping('/admin/topic/tag/edit')]
    public function edit_post(EditTagRequest $request, AvatarUpload $upload)
    {
        if (!admin_auth()->check()) {
            return Json_Api(419, false, ['msg' => '无权限']);
        }
        $id = $request->input('id');
        $icon = $request->input('icon');
        $name = $request->input('name');
        $description = $request->input('description');
        $color = $request->input('color');
        $userClass = $request->input('userClass');
        if ($userClass) {
            $userClass = json_encode($userClass, JSON_THROW_ON_ERROR, JSON_UNESCAPED_UNICODE);
        } else {
            $userClass = null;
        }
        TopicTag::query()->where('id', $id)->update(['name' => $name, 'description' => $description, 'color' => $color, 'icon' => $icon, 'userClass' => $userClass]);
        return redirect()->url(request()->input('Redirect'))->with('success', '修改成功!')->go();
    }
    #[PostMapping('/admin/topic/tag/remove')]
    public function remove()
    {
        if (!admin_auth()->check()) {
            return Json_Api(419, false, ['msg' => '无权限']);
        }
        $id = request()->input('id');
        if (!$id) {
            return Json_Api(403, false, ['msg' => '请求id不能为空']);
        }
        if ($id === 1) {
            return Json_Api(403, false, ['msg' => '安全起见,你不能删除id为1的板块,因为这属于是帖子的默认分类']);
        }
        if (!TopicTag::query()->where('id', $id)->count()) {
            return Json_Api(403, false, ['msg' => 'id为' . $id . '的板块不存在']);
        }
        // 迁移工作
        go(function () use($id) {
            Topic::query()->where('tag_id', $id)->update(['tag_id' => 1]);
        });
        TopicTag::query()->where('id', $id)->delete();
        return Json_Api(200, true, ['msg' => '删除成功!']);
    }
    #[GetMapping('/admin/topic/tag/jobs')]
    public function jobs()
    {
        $jobs = TopicTag::query()->where('status', '待审核')->paginate(15);
        return view('Topic::Tag.jobs', ['page' => $jobs]);
    }
    #[PostMapping('/admin/topic/tag/job/approval')]
    public function job_approval()
    {
        if (!admin_auth()->check()) {
            return Json_Api(419, false, ['msg' => '无权限']);
        }
        $id = request()->input('id');
        if (!$id) {
            return Json_Api(403, false, ['msg' => '请求id不能为空']);
        }
        if (!TopicTag::query()->where('id', $id)->count()) {
            return Json_Api(403, false, ['msg' => 'id为' . $id . '的板块不存在']);
        }
        TopicTag::where('status', '待审核')->where('id', $id)->update(['status' => null]);
        $user = TopicTag::query()->find($id)->user;
        $url = url('/tags/' . $id . '.html');
        // 判断用户是否愿意接收通知
        go(function () use($url, $user) {
            $Subject = '【' . get_options('web_name') . '】 你的板块创建申请已审核通过';
            $Body = <<<HTML
<h3>标题: 你的板块创建申请已审核通过,现在可以使用啦</h3>
<p>链接: <a href="{$url}">{$url}</a></p>
HTML;
            Email()->send($user->email, $Subject, $Body);
        });
        return Json_Api(200, true, ['msg' => '修改成功!']);
    }
    #[PostMapping('/admin/topic/tag/job/reject')]
    public function job_reject()
    {
        if (!admin_auth()->check()) {
            return Json_Api(419, false, ['msg' => '无权限']);
        }
        $id = request()->input('id');
        if (!$id) {
            return Json_Api(403, false, ['msg' => '请求id不能为空']);
        }
        if (!TopicTag::query()->where('status', '待审核')->where('id', $id)->count()) {
            return Json_Api(403, false, ['msg' => 'id为' . $id . '的板块不存在']);
        }
        $user = TopicTag::query()->where('status', '待审核')->find($id)->user;
        $url = url('/tags');
        // 判断用户是否愿意接收通知
        $content = request()->input('content', '无理由');
        go(function () use($url, $user, $content) {
            $Subject = '【' . get_options('web_name') . '】 你的板块创建申请已被驳回';
            $Body = <<<HTML
<h3>标题: 你的板块创建申请已被驳回</h3>
<p>驳回理由:{$content}</p>
<p>链接: <a href="{$url}">{$url}</a></p>
HTML;
            Email()->send($user->email, $Subject, $Body);
        });
        TopicTag::query()->where('status', '待审核')->where('id', $id)->delete();
        return Json_Api(200, true, ['msg' => '修改成功!']);
    }
}