<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
use Hyperf\Watcher\Driver\ScanFileDriver;

return [
    'driver' => ScanFileDriver::class,
    'bin' => env('WATCHER_BIN', 'php'),
    'watch' => [
        'dir' => ['app'],
        'file' => ['.env', 'build-info.php'],
        'scan_interval' => 2000,
    ],
];
