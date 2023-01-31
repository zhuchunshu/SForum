<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
use App\Plugins\Mail\src\Mail;
use App\Plugins\Mail\src\Service\SendService;

if (! function_exists('Email')) {
    function Email(): SendService
    {
        return new SendService();
    }
}

if (! function_exists('EmailData')) {
    function EmailData(): Mail
    {
        return (new Mail())->init()->data();
    }
}
