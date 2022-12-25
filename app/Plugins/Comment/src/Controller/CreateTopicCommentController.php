<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Comment\src\Controller;

use App\Plugins\Comment\src\Handler\CreateTopicComment;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;

#[Controller(prefix: '/topic/create/comment')]
#[Middleware(LoginMiddleware::class)]
class CreateTopicCommentController
{
    #[GetMapping(path: '{topic_id}')]
    public function index($topic_id)
    {
        if (! Authority()->check('comment_create')) {
            return redirect()->back()->with('danger', '无评论权限')->go();
        }
        if (! Topic::query()->where(['id' => $topic_id, 'status' => 'publish'])->exists()) {
            return redirect()->back()->with('danger', '帖子不存在')->go();
        }
        $topic = Topic::query()->find($topic_id);
        if (@$topic->post->options->disable_comment) {
            return redirect()->back()->with('danger', '此帖子关闭了评论功能')->go();
        }
        return view('Comment::topic.create', ['topic' => $topic]);
    }

    #[PostMapping(path: '{topic_id}')]
    public function store($topic_id)
    {
        return (new CreateTopicComment())->handler($topic_id);
    }
}
