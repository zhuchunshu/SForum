<?php

namespace App\Plugins\Core\src\Lib;

class TextParsing
{
    public function keywords($keywords):string{
        return <<<HTML
<a href="">#{$keywords}</a>
HTML;

    }

    public function at($username): string
    {
        return <<<HTML
<a href="">{$username}</a>
HTML;
    }
}