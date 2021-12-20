<?php

declare(strict_types=1);

namespace App\Command\CodeFec\Plugins;

use App\CodeFec\Plugins;
use Hyperf\Command\Command as HyperfCommand;
use Hyperf\Command\Annotation\Command;
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
        (new Plugins())->composerInstall();
        $this->line('Successfully installed', 'info');
    }
}
