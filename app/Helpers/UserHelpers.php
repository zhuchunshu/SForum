<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
if (! function_exists('u_username')) {
    function u_username($user, array $data = [])
    {
        return (new \App\Plugins\User\src\Helpers\UserNameRender())->render($user, $data);
    }
}
