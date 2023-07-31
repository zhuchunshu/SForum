<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Command\CodeFec;

use Hyperf\Command\Annotation\Command;
use Hyperf\Command\Command as HyperfCommand;
use Psr\Container\ContainerInterface;
use Symfony\Component\Console\Input\InputOption;

#[Command]
class Upgrading extends HyperfCommand
{
    /**
     * @var ContainerInterface
     */
    protected $container;
    
    public function __construct(ContainerInterface $container)
    {
        $this->container = $container;
        parent::__construct('CodeFec:Upgrading');
        // 强制更新
        $this->addOption('force', 'f', InputOption::VALUE_NONE, '强制更新');
        // 不备份
        $this->addOption('no-backup', 'nb', InputOption::VALUE_NONE, '不进行备份');
    }
    
    public function configure()
    {
        parent::configure();
        $this->setDescription('系统升级');
    }
    
    public function handle()
    {
        $install = make(\App\CodeFec\Upgrading::class, ['output' => $this->output, 'command' => $this, 'force' => $this->input->getOption('force'), 'no_backup' => $this->input->getOption('no-backup')]);
        $install->run();
    }
}