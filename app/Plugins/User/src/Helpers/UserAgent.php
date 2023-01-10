<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Helpers;

class UserAgent
{
    public static function getBrowser($agent): string
    {
        if (preg_match('/MSIE\s([^\s|;]+)/i', $agent, $regs)) {
            $outputer = '<i class="ua-icon icon-ie"></i>Internet Explore';
        } elseif (preg_match('/FireFox\/([^\s]+)/i', $agent, $regs)) {
            $str1 = explode('Firefox/', $regs[0]);
            $FireFox_vern = explode('.', $str1[1]);
            $outputer = '<i class="ua-icon icon-firefox"></i>FireFox';
        } elseif (preg_match('/Maxthon([\d]*)\/([^\s]+)/i', $agent, $regs)) {
            $str1 = explode('Maxthon/', $agent);
            $Maxthon_vern = explode('.', $str1[1]);
            $outputer = '<i class="ua-icon icon-edge"></i>MicroSoft Edge';
        } elseif (preg_match('#360([a-zA-Z0-9.]+)#i', $agent, $regs)) {
            $outputer = '<i class="ua-icon icon-360"></i>360极速浏览器';
        } elseif (preg_match('/Edge([\d]*)\/([^\s]+)/i', $agent, $regs)) {
            $str1 = explode('Edge/', $regs[0]);
            $Edge_vern = explode('.', $str1[1]);
            $outputer = '<i class="ua-icon icon-edge"></i>MicroSoft Edge';
        } elseif (stripos($agent, 'UC') !== false) {
            $str1 = explode('rowser/', $agent);
            $UCBrowser_vern = explode('.', $str1[1]);
            $outputer = '<i class="ua-icon icon-uc"></i>UC浏览器';
        } elseif (preg_match('/QQ/i', $agent, $regs) || preg_match('/QQBrowser\/([^\s]+)/i', $agent, $regs)) {
            $str1 = explode('rowser/', $agent);
            $QQ_vern = explode('.', $str1[1]);
            $outputer = '<i class= "ua-icon icon-qq"></i>QQ浏览器';
        } elseif (preg_match('/UBrowser/i', $agent, $regs)) {
            $str1 = explode('rowser/', $agent);
            $UCBrowser_vern = explode('.', $str1[1]);
            $outputer = '<i class="ua-icon icon-uc"></i>UC浏览器';
        } elseif (preg_match('/Opera[\s|\/]([^\s]+)/i', $agent, $regs)) {
            $outputer = '<i class= "ua-icon icon-opera"></i>Opera';
        } elseif (preg_match('/Chrome([\d]*)\/([^\s]+)/i', $agent, $regs)) {
            $str1 = explode('Chrome/', $agent);
            $chrome_vern = explode('.', $str1[1]);
            $outputer = '<i class="ua-icon icon-chrome""></i>Google Chrome';
        } elseif (preg_match('/safari\/([^\s]+)/i', $agent, $regs)) {
            $str1 = explode('Version/', $agent);
            $safari_vern = explode('.', $str1[1]);
            $outputer = '<i class="ua-icon icon-safari"></i>Safari';
        } else {
            $outputer = '<i class="ua-icon icon-chrome"></i>Google Chrome';
        }
        return $outputer;
    }

    // 获取操作系统信息
    public static function getOs($agent): string
    {
        $os = false;

        if (stripos($agent, 'win') !== false) {
            if (preg_match('/nt 6.0/i', $agent)) {
                $os = '<i class= "ua-icon icon-win1"></i>Windows Vista';
            } elseif (preg_match('/nt 6.1/i', $agent)) {
                $os = '<i class= "ua-icon icon-win1"></i>Windows 7';
            } elseif (preg_match('/nt 6.2/i', $agent)) {
                $os = '<i class="ua-icon icon-win2"></i>Windows 8';
            } elseif (preg_match('/nt 6.3/i', $agent)) {
                $os = '<i class= "ua-icon icon-win2"></i>Windows 8.1';
            } elseif (preg_match('/nt 5.1/i', $agent)) {
                $os = '<i class="ua-icon icon-win1"></i>Windows XP';
            } elseif (preg_match('/nt 10.0/i', $agent)) {
                $os = '<i class="ua-icon icon-win2"></i>Windows 10';
            } else {
                $os = '<i class="ua-icon icon-win2"></i>Windows X64';
            }
        } elseif (stripos($agent, 'android') !== false) {
            if (preg_match('/android 9/i', $agent)) {
                $os = '<i class="ua-icon icon-android"></i>Android Pie';
            } elseif (preg_match('/android 8/i', $agent)) {
                $os = '<i class="ua-icon icon-android"></i>Android Oreo';
            } else {
                $os = '<i class="ua-icon icon-android"></i>Android';
            }
        } elseif (stripos($agent, 'ubuntu') !== false) {
            $os = '<i class="ua-icon icon-ubuntu"></i>Ubuntu';
        } elseif (stripos($agent, 'linux') !== false) {
            $os = '<i class= "ua-icon icon-linux"></i>Linux';
        } elseif (stripos($agent, 'iPhone') !== false) {
            $os = '<i class="ua-icon icon-apple"></i>iPhone';
        } elseif (stripos($agent, 'mac') !== false) {
            $os = '<i class="ua-icon icon-mac"></i>MacOS';
        } elseif (stripos($agent, 'fusion') !== false) {
            $os = '<i class="ua-icon icon-android"></i>Android';
        } else {
            $os = '<i class="ua-icon icon-linux"></i>Linux';
        }
        return $os;
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
