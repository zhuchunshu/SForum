<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Listener;

use App\CodeFec\CodeFec;
use Hyperf\Event\Annotation\Listener;
use Hyperf\Event\Contract\ListenerInterface;
use Hyperf\Framework\Event\BootApplication;
use Psr\Container\ContainerInterface;

#[Listener]
class Boot implements ListenerInterface
{
    /**
     * @var ContainerInterface
     */
    private ContainerInterface $container;

    public function __construct(ContainerInterface $container)
    {
        $this->container = $container;
    }

    public function listen(): array
    {
        return [
            BootApplication::class,
        ];
    }

    public function process(object $event): void
    {
        if (! file_exists(BASE_PATH . '/app/CodeFec/storage/update.lock')) {
            if (file_exists(BASE_PATH . '/app/CodeFec/storage/install.lock') || $this->get_step() >= 5) {
                (new CodeFec())->handle();
            }
        }
    }

    private function get_step()
    {
        if (! is_dir(BASE_PATH . '/app/CodeFec/storage')) {
            mkdir(BASE_PATH . '/app/CodeFec/storage');
        }
        // 创建文件
        if (! file_exists(BASE_PATH . '/app/CodeFec/storage/install.step.lock')) {
            file_put_contents(BASE_PATH . '/app/CodeFec/storage/install.step.lock', 1);
        }
        if (! @file_get_contents(BASE_PATH . '/app/CodeFec/storage/install.step.lock')) {
            $step = 1;
        } else {
            $step = (int) file_get_contents(BASE_PATH . '/app/CodeFec/storage/install.step.lock');
        }
        return $step;
    }
}
