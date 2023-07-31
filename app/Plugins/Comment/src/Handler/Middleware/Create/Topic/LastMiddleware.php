<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Comment\src\Handler\Middleware\Create\Topic;

use App\Plugins\Comment\src\Annotation\Topic\CreateLastMiddleware;
use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;
use App\Plugins\Topic\src\Models\Topic;
#[CreateLastMiddleware]
class LastMiddleware implements MiddlewareInterface
{
    public function handler($data, \Closure $next)
    {
        Topic::query()->where('id', $data['topic_id'])->update(['last_time' => time()]);
        return $next($data);
    }
}