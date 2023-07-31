<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\ShortCode;

use App\CodeFec\Annotation\ShortCode\ShortCodeR;
use Hyperf\View\RenderInterface;
class Single
{
    #[ShortCodeR(name: 'friend_links')]
    public function friend_links()
    {
        $container = \Hyperf\Utils\ApplicationContext::getContainer();
        return $container->get(RenderInterface::class)->render('App::widget.shortCode.friend_links');
    }
}