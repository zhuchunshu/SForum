<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\CodeFec\Itf\Itf;

use Hyperf\Collection\Arr;

class Itf implements ItfInterface
{
    public array $list = [];

    public function add($class, $id, $data): bool
    {
        Arr::set($this->list, $class . '.' . $class . '_' . $id, $data);
        return true;
    }

    public function re($class, $id, $data): bool
    {
        $this->list[$class][$class . '_' . $id] = $data;
        return true;
    }

    public function del($class, $id): bool
    {
        Arr::forget($this->list, $class . '.' . $class . '_' . $id);
        return true;
    }

    public function get($class): array
    {
        if (Arr::has($this->list, $class)) {
            $array = $this->list[$class];
            if ($array && is_array($array)) {
                ksort($array);
            } else {
                $array = [];
            }
            return $array;
        }
        return [];
    }

    public function all(): array
    {
        return $this->list;
    }

    public function _del($class)
    {
        $this->list[$class] = [];
    }
}
