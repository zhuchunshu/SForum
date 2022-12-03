<?php

declare(strict_types=1);
/**
 * This file is part of Hyperf.
 * @link     https://www.hyperf.io
 * @document https://hyperf.wiki
 * @contact  group@hyperf.io
 * @license  https://github.com/hyperf/hyperf/blob/master/LICENSE
 */
namespace App\Command;

use App\CodeFec\Install;
use Hyperf\Command\Annotation\Command;
use Hyperf\Command\Command as HyperfCommand;
use Hyperf\Watcher\Option;
use Hyperf\Watcher\Watcher;
use Psr\Container\ContainerInterface;
use Symfony\Component\Console\Input\InputOption;

/**
 * @Command
 */
class StartCommand extends HyperfCommand
{
    protected ContainerInterface $container;

    public function __construct(ContainerInterface $container)
    {
        parent::__construct('CodeFec');

        $this->container = $container;
        $this->setDescription('CodeFec Start Server');
        $this->addOption('file', 'F', InputOption::VALUE_OPTIONAL | InputOption::VALUE_IS_ARRAY, '', []);
        $this->addOption('dir', 'D', InputOption::VALUE_OPTIONAL | InputOption::VALUE_IS_ARRAY, '', []);
        $this->addOption('no-restart', 'N', InputOption::VALUE_NONE, 'Whether no need to restart server');
    }

    public function handle()
    {
        if (file_exists(BASE_PATH . '/app/CodeFec/storage/install.lock') || (new Install($this->output, $this))->getStep() >= 5) {
            
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
        } else {
            $install = make(Install::class, [
                'output' => $this->output,
                'command' => $this,
            ]);

            $install->run();
        }
    }
}
