<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Handler\Topic\Middleware\Create;

use App\Plugins\Core\src\Models\PostsOption;
use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;

#[\App\Plugins\Topic\src\Annotation\Topic\CreateLastMiddleware]
class SetDisableCommentMiddleware implements MiddlewareInterface
{
    public function handler($data, \Closure $next)
    {
        $disable_comment = (bool) request()->input('options.disable_comment', false);
        $posts_options_id = $data['posts_options']['id'];
        PostsOption::query()->where('id', $posts_options_id)->update([
            'disable_comment' => $disable_comment,
        ]);
        return $next($data);
    }
}
