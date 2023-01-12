<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Helpers\Middlewares\UserNameRender;

use App\Plugins\User\src\Annotation\UserNameRenderAnnotation;
use App\Plugins\User\src\Helpers\Middlewares\MiddlewareInterface;

#[UserNameRenderAnnotation]
class LinkName implements MiddlewareInterface
{
    public function handler($user, array $data, \Closure $next)
    {
        if (data_get($data, 'link', true) !== false) {
            $user->usernameRender = '<a href="/users/' . $user->id . '.html" style="' . $this->getStyle($data) . '" class="' . $this->getClass($data) . '">' . $user->usernameRender . '</a>';
        } else {
            $user->usernameRender = '<span style="' . $this->getStyle($data) . '" class="' . $this->getClass($data) . '">' . $user->usernameRender . '</span>';
        }
        //return json_encode($data['link']);
        return $next($user, $data);
    }

    private function getClass($data)
    {
        $classes = data_get($data, 'class', ['text-reset']);
        if (is_array($classes)) {
            $classes = implode(' ', $classes);
        }
        return $classes;
    }

    private function getStyle($data)
    {
        $style = data_get($data, 'style', []);
        if (is_array($style)) {
            $style = implode(' ', $style);
        }
        return $style;
    }
}
