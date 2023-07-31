<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Lib;

use App\Plugins\User\src\Models\User;
use Hyperf\Utils\Str;
class TextParsing
{
    public function keywords($keywords) : string
    {
        return <<<HTML
<a href="/keywords/{$keywords}.html">#{$keywords}</a>
HTML;
    }
    public function at($username) : string
    {
        $username = Str::after($username, '@');
        $username = trim($username);
        $username = strip_tags($username);
        if (User::query()->where('username', $username)->exists()) {
            return <<<HTML
<a href="/users/{$username}.username">@{$username}</a>
HTML;
        }
        return '@' . $username;
    }
}