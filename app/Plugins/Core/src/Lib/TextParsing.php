<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Lib;

use App\Plugins\User\src\Models\User;
use Hyperf\Stringable\Str;

class TextParsing
{
    public function keywords($keywords): string
    {
        return <<<HTML
<a href="/keywords/{$keywords}.html">#{$keywords}</a>
HTML;
    }

    public function at($username): string
    {
        $username = Str::after($username, '@');
        $username = trim($username);
        $username = strip_tags($username);
        if (User::query()->where('username', $username)->exists()) {
            $uid = User::query()->where('username', $username)->first()->id;
            return <<<HTML
<a href="/users/{$uid}.html">@{$username}</a>
HTML;
        }
        return '@' . $username;
    }
}
