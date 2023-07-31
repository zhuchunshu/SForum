<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Service\interfaces;

interface UserManagementInterface
{
    // 处理器
    public function handler() : string;
    // 编辑视图
    public function edit_view() : string;
    // 查看视图
    public function show_view() : string;
}