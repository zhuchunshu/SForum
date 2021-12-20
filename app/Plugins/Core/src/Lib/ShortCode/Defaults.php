<?php


namespace App\Plugins\Core\src\Lib\ShortCode;


class Defaults
{
    public static function a($match){
        return "<a>$match[1]</a>";
    }
}