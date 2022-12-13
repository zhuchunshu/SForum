<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Controllers;

use App\Model\AdminUser;
use App\Plugins\Core\src\Handler\AvatarUpload;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\Topic\src\Models\TopicTag;
use App\Plugins\Topic\src\Requests\CreateTagRequest;
use App\Plugins\Topic\src\Requests\EditTagRequest;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\PostMapping;

#[Controller]
class TagsController
{
    #[GetMapping(path: '/tags')]
    public function index(): \Psr\Http\Message\ResponseInterface
    {
        $page = TopicTag::query()->where('status', '=', null)->paginate(15, ['*'], 'TagPage');
        return view('Topic::Tags.index', ['page' => $page]);
    }

    #[GetMapping(path: '/tags/{id}.html')]
    public function data($id)
    {
        if (! TopicTag::query()->where('status', '=', null)->where('id', $id)->exists()) {
            return admin_abort('页面不存在', 404);
        }
        $page = Topic::query()
            ->where('tag_id', $id)
            ->with('tag', 'user')
            ->orderBy('id', 'desc')
            ->paginate(get_options('topic_home_num', 15));
        $data = TopicTag::query()->where('id', $id)->first();
        return view('Topic::Tags.data', ['data' => $data, 'page' => $page]);
    }

    #[GetMapping(path: '/tags/create')]
    public function create()
    {
        if (! auth()->check() || ! Authority()->check('topic_tag_create')) {
            return admin_abort('权限不足', 401);
        }
        $userClass = \App\Plugins\User\src\Models\UserClass::query()->get();
        return view('Topic::Tags.create', ['userClass' => $userClass]);
    }

    #[PostMapping(path: '/tags/create')]
    public function create_store(CreateTagRequest $request)
    {
        if (! auth()->check() || ! Authority()->check('topic_tag_create')) {
            return Json_Api(401, false, ['msg' => '无权限']);
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
        $create_status = null;
        $created_msg = '创建成功!';
        if (get_options('topic_create_tag_ex', 'false') === 'true') {
            $create_status = '待审核';
            $created_msg = '创建成功请求已发出,请等待管理员审核';
        }
        TopicTag::create([
            'name' => $name,
            'color' => $color,
            'description' => $description,
            'icon' => $icon,
            'userClass' => $userClass,
            'status' => $create_status,
            'user_id' => auth()->id(),
        ]);
        if (get_options('topic_create_tag_ex', 'false') === 'true') {
            $mail = Email();
            $url = url('/admin/topic/tag/jobs');
            go(function () use ($mail, $url) {
                foreach (AdminUser::query()->get() as $user) {
                    $mail->addAddress($user->email);
                    $mail->Subject = '【' . get_options('web_name') . '】 有用户申请创建了标签，需要你审核';
                    $mail->Body = <<<HTML
<h3>标题: 有用户申请创建了标签，需要你审核</h3>
<p>链接: <a href="{$url}">{$url}</a></p>
HTML;
                    $mail->send();
                }
            });
        }

        return redirect()->url('/tags/create')->with('success', $created_msg)->go();
    }

    #[GetMapping(path: '/tags/{id}/edit')]
    public function edit($id)
    {
        if (! auth()->check() || ! Authority()->check('topic_tag_create')) {
            return admin_abort('权限不足', 401);
        }
        if (! TopicTag::query()->where('status', '=', null)->where('id', $id)->count()) {
            return admin_abort('id为' . $id . '的标签不存在', 403);
        }
        $data = TopicTag::query()->find($id);
        if ((int) $data->user_id !== (int) auth()->id()) {
            return admin_abort('您无权限修改', 401);
        }
        $userClass = \App\Plugins\User\src\Models\UserClass::query()->get();
        return view('Topic::Tags.edit', ['data' => $data, 'userClass' => $userClass]);
    }

    #[PostMapping(path: '/tags/edit')]
    public function edit_post(EditTagRequest $request, AvatarUpload $upload)
    {
        if (! auth()->check() || ! Authority()->check('topic_tag_create')) {
            return admin_abort('权限不足', 401);
        }
        $id = $request->input('id');
        $name = $request->input('name');
        $description = $request->input('description');
        $color = $request->input('color');
        $userClass = $request->input('userClass');
        if ($userClass) {
            $userClass = json_encode($userClass, JSON_THROW_ON_ERROR, JSON_UNESCAPED_UNICODE);
        } else {
            $userClass = null;
        }

        $data = TopicTag::query()->where('status', '=', null)->find($id);
        if ((int) $data->user_id !== (int) auth()->id()) {
            return admin_abort('您无权限修改', 401);
        }

        TopicTag::query()->where('id', $id)->update([
            'name' => $name,
            'description' => $description,
            'color' => $color,
            'icon' => request()->input('icon'),
            'userClass' => $userClass,
        ]);
        return redirect()->url('/tags/' . $id . '/edit')->with('success', '修改成功!')->go();
    }
}
