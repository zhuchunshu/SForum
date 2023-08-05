<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\CodeFec\Header;

class functions
{
    public static function header(): HeaderInterface
    {
        $container = \Hyperf\Context\ApplicationContext::getContainer();
        return $container->get(HeaderInterface::class);
    }

    public static function left(): array
    {
        $arr = [];
        foreach (\App\CodeFec\Header\functions::header()->get() as $value) {
            if ($value['type'] == 0) {
                $arr[] = $value;
            }
        }
        return $arr;
    }

    public static function right(): array
    {
        $arr = [];
        foreach (\App\CodeFec\Header\functions::header()->get() as $value) {
            if ($value['type'] == 1) {
                $arr[] = $value;
            }
        }
        return $arr;
    }

    public static function headerBtn(): array
    {
        $arr = [];
        foreach (\App\CodeFec\Header\functions::header()->get() as $value) {
            if ($value['type'] == 2) {
                $arr[] = $value;
            }
        }
        return $arr;
    }
}
