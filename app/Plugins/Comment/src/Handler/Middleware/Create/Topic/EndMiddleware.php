<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Comment\src\Handler\Middleware\Create\Topic;

use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;

class EndMiddleware implements MiddlewareInterface
{
    public function handler($data, \Closure $next)
    {
        return redirect()->url('/' . $data['topic_id'] . '.html/' . $data['comment']['id'] . '?page=' . get_topic_comment_page($data['comment']['id'])."&clean_topic_comment_content_cache=create_topic_comment_".$data['topic_id'])->with('success', '发表成功!')->go();
    }
}
