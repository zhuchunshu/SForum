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

class UpdateEndMiddleware implements MiddlewareInterface
{
    public function handler($data, \Closure $next)
    {
        return redirect()->url('/' . $data['topic_id'] . '.html')->with('success', '更新成功!')->go();
    }
}
