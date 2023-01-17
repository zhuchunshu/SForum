<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Command\CodeFec\Plugins;

use Hyperf\Command\Annotation\Command;
use Hyperf\Command\Command as HyperfCommand;
use Psr\Container\ContainerInterface;

/**
 * @Command
 */
#[Command]
class ComposerInstall extends HyperfCommand
{
    /**
     * @var ContainerInterface
     */
    protected $container;

    public function __construct(ContainerInterface $container)
    {
        $this->container = $container;

        parent::__construct('CodeFec:PluginsComposerInstall');
    }

    public function configure()
    {
        parent::configure();
        $this->setDescription('CodeFec All Plugins Run Composer Install');
    }

    public function handle()
    {
        if (stripos(system_name(), 'Linux') !== false) {
            \Swoole\Coroutine\System::exec('yes | composer update');
        } else {
            \Swoole\Coroutine\System::exec('composer update');
        }
        $this->line('Successfully installed', 'info');
    }
}
