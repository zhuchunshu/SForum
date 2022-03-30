<?php

declare(strict_types=1);

namespace App\Command\CodeFec;

use App\CodeFec\Install;
use Hyperf\Command\Command as HyperfCommand;
use Hyperf\Command\Annotation\Command;
use Psr\Container\ContainerInterface;

/**
 * @Command
 */
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
    }

    public function configure()
    {
        parent::configure();
        $this->setDescription('系统升级');
    }

    public function handle()
    {
	    $install = make(\App\CodeFec\Upgrading::class, [
		    'output' => $this->output,
		    'command' => $this
	    ]);
	
	    $install->run();
    }
}
