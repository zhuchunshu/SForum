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

class Redirect
{
    private string $url;
    public function url($url) : Redirect
    {
        $this->url = $url;
        return $this;
    }
    public function back() : Redirect
    {
        $this->url = request()->getHeader('referer')[0] ?: '/';
        return $this;
    }
    public function with($key, $value) : Redirect
    {
        session()->flash($key, $value);
        return $this;
    }
    public function go() : \Psr\Http\Message\ResponseInterface
    {
        return response()->redirect($this->url);
    }
}