<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Handler\Topic\Middleware\Create;

use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;
use App\Plugins\Topic\src\Models\TopicTag;
use App\Plugins\User\src\Models\UserClass;

#[\App\Plugins\Topic\src\Annotation\Topic\CreateFirstMiddleware]
class CreateFirstMiddleware implements MiddlewareInterface
{
    public function handler($data, \Closure $next)
    {
        //return cache()->get('topic_create_time_' . auth()->id());
        if (cache()->has('topic_create_time_' . auth()->id())) {
            $time = cache()->get('topic_create_time_' . auth()->id()) - time();
            unset($data['basis']['content']);
            return redirect()->with('danger', '发帖过于频繁,请 ' . $time . ' 秒后再试')->url('topic/create?' . http_build_query($data))->go();
        }
        $class_name = UserClass::query()->where('id', auth()->data()->class_id)->first()->name;
        $tag_value = TopicTag::query()->where('id', $data['basis']['tag'])->first();
        if (! user_TopicTagQuanxianCheck($tag_value, $class_name)) {
            unset($data['basis']['content']);
            return redirect()->with('danger', '无权使用此标签')->url('topic/create?' . http_build_query($data))->go();
        }
        return $next($data);
    }
}
