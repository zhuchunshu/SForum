<?php

declare(strict_types=1);

namespace App\Command;

use Hyperf\Command\Command as HyperfCommand;
use Hyperf\Command\Annotation\Command;
use Psr\Container\ContainerInterface;

/**
 * @Command
 */
#[Command]
class ClearCache extends HyperfCommand
{
    /**
     * @var ContainerInterface
     */
    protected $container;

    public function __construct(ContainerInterface $container)
    {
        $this->container = $container;

        parent::__construct('ClearCache');
    }

    public function configure()
    {
        parent::configure();
        $this->setDescription('清理缓存');
    }

    public function handle()
    {
	    \Swoole\Coroutine\System::exec('composer dump-autoload -o');
	    \Swoole\Coroutine\System::exec('php CodeFec');
    }
}
