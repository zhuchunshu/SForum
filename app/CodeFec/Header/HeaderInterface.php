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

interface HeaderInterface
{
    /**
     * 获取页头菜单数组.
     */
    public function get(): array;

    /**
     * 新增页头钩子.
     *
     * @param int $id 唯一id
     * @param int $type 类型(0:左,1:右,2:页头按钮) $type
     * @param string $view 视图名称 $view
     */
    public function add(int $id, int $type, string $view): bool;
}
