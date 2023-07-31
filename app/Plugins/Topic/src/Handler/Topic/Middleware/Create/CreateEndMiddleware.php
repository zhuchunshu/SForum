<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Handler\Topic\Middleware\Create;

use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;
class CreateEndMiddleware implements MiddlewareInterface
{
    public function handler($data, \Closure $next)
    {
        // 帖子发表成功
        // 添加发表成功事件
        EventDispatcher()->dispatch(new \App\Plugins\User\src\Event\Task\Daily\CreateTopic($data['topic_id']));
        return redirect()->url('/?clean_topic_content_cache=true')->with('success', '发表成功!')->go();
    }
}