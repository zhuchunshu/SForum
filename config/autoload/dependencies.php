<?php

declare(strict_types=1);

use App\CodeFec\Ui\Ui;
use App\CodeFec\Menu\Menu;
use App\CodeFec\View\Render;
use App\CodeFec\Header\Header;
use App\CodeFec\Ui\UiInterface;
use Hyperf\View\RenderInterface;
use App\CodeFec\Menu\MenuInterface;
use App\CodeFec\Header\HeaderInterface;

/**
 * CodeFec - Hyperf
 *
 * @link     https://github.com/zhuchunshu
 * @document https://codefec.com
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/CodeFecHF/blob/master/LICENSE
 */
return [
    Hyperf\HttpServer\CoreMiddleware::class => App\Middleware\CoreMiddleware::class,
    MenuInterface::class => Menu::class,
    HeaderInterface::class => Header::class,
    UiInterface::class => Ui::class,
    RenderInterface::class => Render::class
];
