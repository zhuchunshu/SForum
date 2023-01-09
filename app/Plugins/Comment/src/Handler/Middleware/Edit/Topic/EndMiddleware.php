<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Comment\src\Handler\Middleware\Edit\Topic;

use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;

class EndMiddleware implements MiddlewareInterface
{
    public function handler($data, \Closure $next)
    {
        $topic_id = TopicComment::query()->find($data['comment_id'])->topic_id;
        return redirect()->url('/' . $topic_id . '.html/' . $data['comment_id'] . '?page=' . get_topic_comment_page((int)$data['comment_id']))->with('success', '更新成功!')->go();
    }
}
