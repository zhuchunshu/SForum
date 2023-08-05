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
use Hyperf\Contract\ConfigInterface;
use Hyperf\Contract\StdoutLoggerInterface;
use Hyperf\Engine\Coroutine;
use Hyperf\Server\ServerFactory;
use Psr\Container\ContainerInterface;
use Psr\EventDispatcher\EventDispatcherInterface;
use Swoole\Coroutine\System;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Input\InputOption;
use Symfony\Component\Console\Output\OutputInterface;

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
            $install = make(DockerInstall::class, ['output' => $this->output, 'command' => $this]);
            $install->run();
        }
    }

    protected function execute(InputInterface $input, OutputInterface $output)
    {
        shell_exec('composer du');
        $this->checkEnvironment($output);
        $serverFactory = $this->container->get(ServerFactory::class);
        $serverFactory->setEventDispatcher($this->container->get(EventDispatcherInterface::class));
        $serverFactory->setLogger($this->container->get(StdoutLoggerInterface::class));
        $serverConfig = $this->container->get(ConfigInterface::class)->get('server', []);
        if (! $serverConfig) {
            throw new \InvalidArgumentException('At least one se$rver should be defined.');
        }
        $serverFactory->configure($serverConfig);
        Coroutine::set(['hook_flags' => \Hyperf\Support\swoole_hook_flags()]);
        $serverFactory->start();
        return 0;
    }

    private function checkEnvironment(OutputInterface $output)
    {
        if (! extension_loaded('swoole')) {
            return;
        }
        $useShortname = ini_get_all('swoole')['swoole.use_shortname']['local_value'];
        $useShortname = strtolower(trim(str_replace('0', '', $useShortname)));
        if (! in_array($useShortname, ['', 'off', 'false'], true)) {
            $output->writeln("<error>ERROR</error> Swoole short function names must be disabled before the server starts, please set swoole.use_shortname='Off' in your php.ini.");
            exit(SIGTERM);
        }
    }
}
