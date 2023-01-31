<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Mail\src;

interface SendServiceInterface
{
    // 获取服务名称
    public function get_service_name(): string;

    // 发送邮件
    public function send($email, $subject, $body);

    // 后台设置处理器
    public function handler(): string;

    //后台设置视图
    public function view(): string;
}
