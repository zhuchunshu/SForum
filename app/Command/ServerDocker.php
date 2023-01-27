<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Command;

use App\CodeFec\DockerInstall;
use Hyperf\Command\Annotation\Command;
use Hyperf\Command\Command as HyperfCommand;
use Hyperf\Watcher\Option;
use Hyperf\Watcher\Watcher;
use Psr\Container\ContainerInterface;
use Swoole\Coroutine\System;
use Symfony\Component\Console\Input\InputOption;

/**
 * @Command
 */
#[Command]
class ServerDocker extends HyperfCommand
{
    /**
     * @var ContainerInterface
     */
    protected $container;

    public function __construct(ContainerInterface $container)
    {
        $this->container = $container;

        parent::__construct('server:docker');
        $this->container = $container;
        $this->addOption('file', 'F', InputOption::VALUE_OPTIONAL | InputOption::VALUE_IS_ARRAY, '', []);
        $this->addOption('dir', 'D', InputOption::VALUE_OPTIONAL | InputOption::VALUE_IS_ARRAY, '', []);
        $this->addOption('no-restart', 'N', InputOption::VALUE_NONE, 'Whether no need to restart server');
    }

    public function configure()
    {
        parent::configure();
        $this->setDescription('start docker server');
    }

    public function handle()
    {
        if (! file_exists(BASE_PATH . '/app/CodeFec/storage/install.lock')) {
            if (! is_dir(BASE_PATH . '/app/CodeFec/storage')) {
                System::exec('cd ' . BASE_PATH . '/app/CodeFec && mkdir storage');
            }
            $myfile = fopen(BASE_PATH . '/app/CodeFec/storage/install.step.lock', 'wb') or exit('Unable to open file!');
            fwrite($myfile, "5");
            fclose($myfile);
            $install = make(DockerInstall::class, [
                'output' => $this->output,
                'command' => $this,
            ]);

            $install->run();
        }
        go(function () {
            system_clear_cache();
        });
        $option = make(Option::class, [
            'dir' => $this->input->getOption('dir'),
            'file' => $this->input->getOption('file'),
            'restart' => ! $this->input->getOption('no-restart'),
        ]);

        $watcher = make(Watcher::class, [
            'option' => $option,
            'output' => $this->output,
        ]);

        $watcher->run();
    }
}
