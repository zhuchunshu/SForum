<?php

declare(strict_types=1);

namespace App\Command;

use Hyperf\Command\Command as HyperfCommand;
use Hyperf\Command\Annotation\Command;
use Hyperf\Utils\Str;
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
	    if(Str::is('Linux',system_name())){
		    \Swoole\Coroutine\System::exec('yes | composer dump-autoload -o');
		    \Swoole\Coroutine\System::exec('yes| php CodeFec');
	    }else{
		    \Swoole\Coroutine\System::exec('composer dump-autoload -o');
		    \Swoole\Coroutine\System::exec('php CodeFec');
	    }
    }
}
