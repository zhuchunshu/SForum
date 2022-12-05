<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
use App\CodeFec\Header\Header;
use App\CodeFec\Header\HeaderInterface;
use App\CodeFec\Itf\Setting\Setting;
use App\CodeFec\Itf\Setting\SettingInterface;
use App\CodeFec\Menu\Menu;
use App\CodeFec\Menu\MenuInterface;
use App\CodeFec\Ui\Ui;
use App\CodeFec\Ui\UiInterface;
use App\CodeFec\View\Render;
use Hyperf\View\RenderInterface;

return [
    Hyperf\HttpServer\CoreMiddleware::class => App\Middleware\CoreMiddleware::class,
    MenuInterface::class => Menu::class,
    HeaderInterface::class => Header::class,
    UiInterface::class => Ui::class,
    RenderInterface::class => Render::class,
    SettingInterface::class => Setting::class,
    \App\CodeFec\Itf\Route\RouteInterface::class => \App\CodeFec\Itf\Route\Route::class,
    \App\CodeFec\Itf\Itf\ItfInterface::class => \App\CodeFec\Itf\Itf\Itf::class,
    \App\CodeFec\Itf\Theme\ThemeInterface::class => \App\CodeFec\Itf\Theme\Theme::class,
];
