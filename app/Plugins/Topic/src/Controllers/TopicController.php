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

use App\Plugins\Core\src\Models\Post;
use App\Plugins\Topic\src\Handler\Topic\CreateTopic;
use App\Plugins\Topic\src\Handler\Topic\CreateTopicView;
use App\Plugins\Topic\src\Handler\Topic\EditTopic;
use App\Plugins\Topic\src\Handler\Topic\EditTopicView;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\RateLimit\Annotation\RateLimit;

#[Controller(prefix: '/topic')]
#[Middleware(\App\Plugins\User\src\Middleware\AuthMiddleware::class)]
#[Middleware(LoginMiddleware::class)]
class TopicController
{
    #[GetMapping(path: 'create')]
    #[Middleware(LoginMiddleware::class)]
    public function create()
    {
        if (! Authority()->check('topic_create')) {
            return admin_abort('无发帖权限');
        }
        return (new CreateTopicView())->handler();
    }

    #[PostMapping(path: 'create')]
    #[Middleware(LoginMiddleware::class)]
    public function create_post()
    {
        if (! Authority()->check('topic_create')) {
            return Json_Api(419, false, ['无发帖权限']);
        }
        return (new CreateTopic())->handler(request());
    }

    #[PostMapping(path: 'create/upload')]
    #[Middleware(LoginMiddleware::class)]
    public function create_upload()
    {
        if (! Authority()->check('topic_create')) {
            return Json_Api(419, false, ['无发帖权限']);
        }
        return (new CreateTopic())->handler(request());
    }

    #[PostMapping(path: 'create/preview')]
    #[Middleware(LoginMiddleware::class)]
    public function create_preview()
    {
        $content = request()->input('content', '无内容');
        $content = xss()->clean($content);
        return view('Topic::create.preview', ['content' => $content]);
    }

    #[GetMapping(path: '/topic/{topic_id}/edit')]
    public function edit($topic_id)
    {
        if (! Topic::query()->where('id', $topic_id)->exists()) {
            return admin_abort('帖子不存在', 404);
        }
        $data = Topic::query()->find($topic_id);
        $quanxian = false;
        if (Authority()->check('admin_topic_edit') && curd()->GetUserClass(auth()->data()->class_id)['permission-value'] > curd()->GetUserClass($data->user->class_id)['permission-value']) {
            $quanxian = true;
        } elseif (Authority()->check('topic_edit') && auth()->id() === $data->user->id) {
            $quanxian = true;
        }
//        elseif (\App\Plugins\Topic\src\Models\Moderator::query()->where('tag_id', $data->tag_id)->where('user_id', auth()->id())->exists()) {
//            $quanxian = true;
//        }
        if ($quanxian === true) {
            return (new EditTopicView())->handler($data);
        }
        return admin_abort('无权限', 419);
    }

    #[PostMapping(path: '/topic/update')]
    #[RateLimit(create: 1, capacity: 1, consume: 1)]
    public function edit_post()
    {
        $data = Topic::query()->find(request()->input('basis.tag'));
        $quanxian = false;
        if (Authority()->check('admin_topic_edit') && curd()->GetUserClass(auth()->data()->class_id)['permission-value'] > curd()->GetUserClass($data->user->class_id)['permission-value']) {
            $quanxian = true;
        } elseif (Authority()->check('topic_edit') && auth()->id() === $data->user->id) {
            $quanxian = true;
        }
//        elseif (\App\Plugins\Topic\src\Models\Moderator::query()->where('tag_id', $data->tag_id)->where('user_id', auth()->id())->exists()) {
//            $quanxian = true;
//        }
        if ($quanxian === true) {
            return (new EditTopic())->handler();
        }
        return admin_abort('无权限', 419);
    }

    #[PostMapping(path: '/topic/{id}/topic.trashed.restore')]
    public function topic_trashed_restore($id)
    {
        if (! Topic::onlyTrashed()->where('id', $id)->exists()) {
            return redirect()->back()->with('danger', '此主题不在回收站中')->go();
        }
        // 主题信息
        $data = Topic::onlyTrashed()->find($id);
        // 判断权限
        $quanxian = false;
        if (auth()->id() === (int) $data->id && Authority()->check('topic_recover')) {
            $quanxian = true;
        } elseif (Authority()->check('admin_topic_recover')) {
            $quanxian = true;
        } elseif (\App\Plugins\Topic\src\Models\Moderator::query()->where('tag_id', $data->tag_id)->where('user_id', auth()->id())->exists()) {
            $quanxian = true;
        }
        if (! $quanxian) {
            return redirect()->back()->with('danger', '无权限')->go();
        }
        // 恢复主题
        $data->restore();
        $post = Post::withTrashed()->find($data->post_id);
        $post->restore();
        // 发送通知
        if (auth()->id() !== $data->user_id) {
            user_notice()->send($data->user_id, '你有一条主题被恢复', '您的主题《<a href="/' . $data->id . '.html">' . $data->title . '</a>》已被恢复，管理员【<a href="/users/' . auth()->id() . '">' . auth()->data()->username . '</a>】', '/' . $data->id . '.html', true, 'system');
        }
        return redirect()->with('success', '操作成功')->back()->go();
    }
}
