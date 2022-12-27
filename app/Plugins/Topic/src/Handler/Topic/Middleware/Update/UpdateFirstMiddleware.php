<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Handler\Topic\Middleware\Update;

use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;
use App\Plugins\Topic\src\Models\TopicTag;
use App\Plugins\User\src\Models\UserClass;

#[\App\Plugins\Topic\src\Annotation\Topic\UpdateFirstMiddleware]
class UpdateFirstMiddleware implements MiddlewareInterface
{
    public function handler($data, \Closure $next)
    {
        $class_name = UserClass::query()->where('id', auth()->data()->class_id)->first()->name;
        $tag_value = TopicTag::query()->where('id', $data['basis']['tag'])->first();
        if (! user_TopicTagQuanxianCheck($tag_value, $class_name)) {
            return redirect()->with('danger', '无权使用此标签')->back()->go();
        }
        return $next($data);
    }
}
