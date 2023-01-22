<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Service\interfaces;

interface Oauth2Interface
{
    /*
     * 后台设置视图
     */
    public function admin_view(): string;

    /*
     * 唯一标识,建议是英文
     */
    public function mark(): string;

    /**
     * 登陆方式名称.
     */
    public function name(): string;

    /*
     * 登陆方式视图名
     */
    public function view(): string;

    /**
     * 图标.
     */
    public function icon(): string;

    /**
     * 设置处理器.
     */
    public function setting_handler(): string;
}
