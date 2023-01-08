<?php

namespace App\Plugins\Core\src\Lib;

use App\Plugins\User\src\Models\User;
use Hyperf\Utils\Str;

class TextParsing
{
    public function keywords($keywords):string{
        return <<<HTML
<a href="/keywords/{$keywords}.html">#{$keywords}</a>
HTML;

    }

    public function at($username): string
    {
        $username = Str::after($username,"@");
		$username = str_replace(" ","",$username);
		$username = urldecode($username);
        if(User::query()->where("username",$username)->exists()) {
            return <<<HTML
<a href="/users/{$username}.username">@{$username}</a>
HTML;
        }

        return "@".$username;
    }
}