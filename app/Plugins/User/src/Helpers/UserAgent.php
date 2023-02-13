<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Helpers;

use WhichBrowser\Parser;

class UserAgent
{
    public static function getBrowser($agent): string
    {
        $parse = new Parser($agent);
        // 默认输出为null
        $echo = null;

        // chrome
        if ($parse->browser->name === 'Chrome') {
            $echo = '<i class="ua-icon icon-chrome"></i>Chrome';
        }

        //wechat
        if ($parse->browser->name === 'WeChat') {
            $echo = '<i class="ua-icon icon-wechat"></i>WeChat';
        }
        // Internet Explore
        if ($parse->browser->name === 'Internet Explorer') {
            $echo = '<i class="ua-icon icon-ie"></i>IE';
        }
        // FireFox
        if ($parse->browser->name === 'Firefox') {
            $echo = '<i class="ua-icon icon-firefox"></i>FireFox';
        }
        // Edge
        if ($parse->browser->name === 'Edge') {
            $echo = '<i class="ua-icon icon-edge"></i>Edge';
        }
        // 360极速浏览器
        if ($parse->browser->name === '360SE') {
            $echo = '<i class="ua-icon icon-360"></i>360极速浏览器';
        }
        // UC浏览器
        if ($parse->browser->name === 'UC Browser') {
            $echo = '<i class="ua-icon icon-uc"></i>UC浏览器';
        }
        // QQ浏览器
        if ($parse->browser->name === 'QQ Browser') {
            $echo = '<i class="ua-icon icon-qq"></i>QQ浏览器';
        }
        // Opera
        if ($parse->browser->name === 'Opera') {
            $echo = '<i class="ua-icon icon-opera"></i>Opera';
        }
        // Safari
        if ($parse->browser->name === 'Safari') {
            $echo = '<i class="ua-icon icon-safari"></i>Safari';
        }

        // 猎豹浏览器
        if ($parse->browser->name === 'LBBROWSER') {
            $echo = '<i class="ua-icon icon-liebao"></i>猎豹浏览器';
        }

        // 百度浏览器
        if ($parse->browser->name === 'Baidu') {
            $echo = '<i class="ua-icon icon-baidu"></i>百度浏览器';
        }

        // 小米浏览器
        if ($parse->browser->name === 'MIUI Browser') {
            $echo = '<i class="ua-icon icon-xiaomi"></i>小米浏览器';
        }


        if ($echo === null) {
            $echo = $parse->browser->name;
        }
        return $echo ?: $parse->toString();
    }

    // 获取操作系统信息
    public static function getOs($agent): string
    {
        $parse = new Parser($agent);
        // 默认输出为null
        $echo = null;

        // mac
        if ($parse->isOs('OS X')) {
            $echo = '<i class="ua-icon icon-mac"></i>MacOS ' . $parse->os->version->toString();
        }

        // win
        if ($parse->isOs('Windows')) {
            if ((int) $parse->os->version->toString() ?: 1 > 7) {
                $echo = '<i class="ua-icon icon-win2"></i>Windows ' . $parse->os->version->toString();
            } else {
                $echo = '<i class="ua-icon icon-win1"></i>Windows ' . $parse->os->version->toString();
            }
        }

        // ubuntu
        if ($parse->isOs('Ubuntu')) {
            $echo = '<i class="ua-icon icon-ubuntu"></i>Ubuntu';
        }

        //debian
        if ($parse->isOs('Debian')) {
            $echo = '<i class="ua-icon icon-debian"></i>Debian';
        }

        //linux
        if ($parse->isOs('Linux')) {
            $echo = '<i class="ua-icon icon-linux"></i>Linux';
        }

        //android
        if ($parse->isOs('Android')) {
            $echo = '<i class="ua-icon icon-android"></i>Android ' . $parse->os->version->toString();
        }

        //ios || iphone
        if ($parse->isOs('iOS') || $parse->isOs('iPhone')) {
            $echo = '<i class="ua-icon icon-apple"></i>iOS ' . $parse->os->version->toString();
        }

        if ($echo === null) {
            if ($parse->isType('desktop')) {
                $echo = '<i class="ua-icon icon-desktop"></i>' . $parse->os->toString();
            } elseif ($parse->isMobile()) {
                $echo = '<i class="ua-icon icon-mobile"></i>' . $parse->os->toString();
            } else {
                $echo = $parse->os->toString();
            }
        }
        return $echo ?: $parse->toString();
    }

    public static function cutIp(string $ip, int $length = 3, string $replace = '*')
    {
        $ip = explode('.', $ip);
        $result = array_splice($ip, 0, $length);
        $data = [];
        for ($i = 1; $i <= 4; ++$i) {
            $data[] = $replace;
        }
        $data = array_merge($result, $data);
        $_ip = array_splice($data, 0, 4);
        return implode('.', $_ip);
    }
}
