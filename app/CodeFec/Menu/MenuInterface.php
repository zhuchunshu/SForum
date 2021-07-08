<?php

declare(strict_types=1);
/**
 * CodeFec - Hyperf
 *
 * @link     https://github.com/zhuchunshu
 * @document https://codefec.com
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/CodeFecHF/blob/master/LICENSE
 */
namespace App\CodeFec\Menu;

interface MenuInterface
{
    /**
     * 获取菜单数组.
     */
    public function get(): array;

    /**
     * 新增菜单.
     *
     * @param int 菜单id $id
     * @param array 菜单内容 $arr
     * @return bool
     */
    public function add(int $id, array $arr): bool;
}
