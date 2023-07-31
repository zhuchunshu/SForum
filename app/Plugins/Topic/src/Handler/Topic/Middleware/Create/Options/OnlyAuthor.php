<?php

namespace App\Plugins\Topic\src\Handler\Topic\Middleware\Create\Options;

use App\Plugins\Core\src\Models\PostsOption;
use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;
class OnlyAuthor implements MiddlewareInterface
{
    public function handler($data, \Closure $next)
    {
        $disable_comment = (bool) request()->input('options.only_author', false);
        $posts_options_id = $data['posts_options']['id'];
        PostsOption::query()->where('id', $posts_options_id)->update(['only_author' => $disable_comment]);
        return $next($data);
    }
}