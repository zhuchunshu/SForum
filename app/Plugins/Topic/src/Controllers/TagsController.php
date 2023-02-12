<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
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
        $page = Topic::query(true)
            ->where('tag_id', $id)
            ->with('tag', 'user')
            ->orderBy('topping', 'desc')
            ->orderBy('updated_at', 'desc')
            ->paginate((int)get_options('topic_home_num', 15));
        if (request()->input('query') === 'hot') {
            $page = Topic::query()
                ->where('tag_id', $id)
                ->with('tag', 'user')
                ->orderBy('view', 'desc')
                ->orderBy('id', 'desc')
                ->paginate((int)get_options('topic_home_num', 15));
        }
        if (request()->input('query') === 'publish') {
            $page = Topic::query()
                ->where('tag_id', $id)
                ->with('tag', 'user')
                ->orderBy('id', 'desc')
                ->paginate((int)get_options('topic_home_num', 15));
        }
        if (request()->input('query') === 'essence') {
            $page = Topic::query()
                ->where('tag_id', $id)
                ->where([['essence', '>', 0],])
                ->with('tag', 'user')
                ->orderBy('updated_at', 'desc')
                ->paginate((int)get_options('topic_home_num', 15));
        }
        if (request()->input('query') === 'topping') {
            $page = Topic::query()
                ->where('tag_id', $id)
                ->where([['topping', '>', 0],])
                ->with('tag', 'user')
                ->orderBy('updated_at', 'desc')
                ->paginate((int) get_options('topic_home_num', 15));
        }
        $data = TopicTag::query()->where('id', $id)->first();
        $topic_menu = [
            [
                'name' => '最新发布',
                'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-news" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M16 6h3a1 1 0 0 1 1 1v11a2 2 0 0 1 -4 0v-13a1 1 0 0 0 -1 -1h-10a1 1 0 0 0 -1 1v12a3 3 0 0 0 3 3h11"></path>
   <line x1="8" y1="8" x2="12" y2="8"></line>
   <line x1="8" y1="12" x2="12" y2="12"></line>
   <line x1="8" y1="16" x2="12" y2="16"></line>
</svg>',
                'url' => '/tags/' . $data->id . '.html?' . core_http_build_query(['query' => 'publish'], ['page' => request()->input('page', 1)]),
                'parameter' => 'query=publish',
            ],
            [
                'name' => __('app.essence'),
                'icon' => '<svg width="24" height="24" viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg" class="icon w-3 h-3 me-1 d-none d-md-block"><g stroke-width="3" fill-rule="evenodd"><path fill="#fff" fill-opacity=".01" d="M0 0h48v48H0z"/><g stroke="currentColor" fill="none"><path d="M10.636 5h26.728L45 18.3 24 43 3 18.3z"/><path d="M10.636 5L24 43 37.364 5M3 18.3h42"/><path d="M15.41 18.3L24 5l8.59 13.3"/></g></g></svg>',
                'url' => '/tags/' . $data->id . '.html?' . core_http_build_query(['query' => 'essence'], ['page' => request()->input('page', 1)]),
                'parameter' => 'query=essence',
            ],
            [
                'name' => __('app.hot'),
                'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon me-1 d-none d-md-block" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path d="M0 0h24v24H0z" stroke="none"/><path d="M12 12c2-2.96 0-7-1-8 0 3.038-1.773 4.741-3 6-1.226 1.26-2 3.24-2 5a6 6 0 1 0 12 0c0-1.532-1.056-3.94-2-5-1.786 3-2.791 3-4 2z"/></svg>',
                'url' => '/tags/' . $data->id . '.html?' . core_http_build_query(['query' => 'hot'], ['page' => request()->input('page', 1)]),
                'parameter' => 'query=hot',
            ],
        ];
        return view('Topic::Tags.data', ['data' => $data, 'page' => $page, 'topic_menu' => $topic_menu]);
    }

    #[GetMapping(path: '/tags/create')]
    public function create()
    {
        if (! auth()->check() || ! Authority()->check('topic_tag_create')) {
            return admin_abort('权限不足', 419);
        }
        $userClass = \App\Plugins\User\src\Models\UserClass::query()->get();
        return view('Topic::Tags.create', ['userClass' => $userClass]);
    }

    #[PostMapping(path: '/tags/create')]
    public function create_store(CreateTagRequest $request)
    {
        if (! auth()->check() || ! Authority()->check('topic_tag_create')) {
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
            $url = url('/admin/topic/tag/jobs');
            go(function () use ($url) {
                foreach (AdminUser::query()->get() as $user) {
                    $Subject = '【' . get_options('web_name') . '】 有用户申请创建了标签，需要你审核';
                    $Body = <<<HTML
<h3>标题: 有用户申请创建了标签，需要你审核</h3>
<p>链接: <a href="{$url}">{$url}</a></p>
HTML;
                    Email()->send($user->email,$Subject,$Body);
                }
            });
        }

        return redirect()->url('/tags/create')->with('success', $created_msg)->go();
    }

    #[GetMapping(path: '/tags/{id}/edit')]
    public function edit($id)
    {
        if (! auth()->check() || ! Authority()->check('topic_tag_create')) {
            return admin_abort('权限不足', 419);
        }
        if (! TopicTag::query()->where('status', '=', null)->where('id', $id)->count()) {
            return admin_abort('id为' . $id . '的标签不存在', 403);
        }
        $data = TopicTag::query()->find($id);
        if ((int) $data->user_id !== (int) auth()->id()) {
            return admin_abort('您无权限修改', 419);
        }
        $userClass = \App\Plugins\User\src\Models\UserClass::query()->get();
        return view('Topic::Tags.edit', ['data' => $data, 'userClass' => $userClass]);
    }

    #[PostMapping(path: '/tags/edit')]
    public function edit_post(EditTagRequest $request, AvatarUpload $upload)
    {
        if (! auth()->check() || ! Authority()->check('topic_tag_create')) {
            return admin_abort('权限不足', 419);
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
            return admin_abort('您无权限修改', 419);
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
