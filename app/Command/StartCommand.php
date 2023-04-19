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

use App\CodeFec\Install;
use Hyperf\Command\Annotation\Command;
use Symfony\Component\Console\Command\Command as HyperfCommand;
use Hyperf\Contract\ConfigInterface;
use Hyperf\Contract\StdoutLoggerInterface;
use Hyperf\Engine\Coroutine;
use Hyperf\Server\ServerFactory;
use Hyperf\Watcher\Option;
use Hyperf\Watcher\Watcher;
use Psr\Container\ContainerInterface;
use Psr\EventDispatcher\EventDispatcherInterface;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Input\InputOption;
use Symfony\Component\Console\Output\OutputInterface;
use InvalidArgumentException;


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

    protected function execute(InputInterface $input, OutputInterface $output)
    {
        $this->removeFiles(BASE_PATH . '/runtime/container', BASE_PATH . '/runtime/view');
        exec('php CodeFec');

        if (file_exists(BASE_PATH . '/app/CodeFec/storage/install.lock') || (new Install($output, $this))->getStep() >= 5) {

            $this->checkEnvironment($output);

            $serverFactory = $this->container->get(ServerFactory::class)
                ->setEventDispatcher($this->container->get(EventDispatcherInterface::class))
                ->setLogger($this->container->get(StdoutLoggerInterface::class));

            $serverConfig = $this->container->get(ConfigInterface::class)->get('server', []);
            if (!$serverConfig) {
                throw new InvalidArgumentException('At least one server should be defined.');
            }

            $serverFactory->configure($serverConfig);

            Coroutine::set(['hook_flags' => swoole_hook_flags()]);
            $serverFactory->start();
        } else {
            $install = make(Install::class, [
                'output' => $output,
                'command' => $this,
            ]);

            $install->run();
        }

        return 0;
    }

    private function removeFiles(...$values): void
    {
        foreach ($values as $value) {
            exec('rm -rf "' . $value . '"');
        }
    }

    private function checkEnvironment(OutputInterface $output)
    {
        if (!extension_loaded('swoole')) {
            return;
        }
        /**
         * swoole.use_shortname = true       => string(1) "1"     => enabled
         * swoole.use_shortname = "true"     => string(1) "1"     => enabled
         * swoole.use_shortname = on         => string(1) "1"     => enabled
         * swoole.use_shortname = On         => string(1) "1"     => enabled
         * swoole.use_shortname = "On"       => string(2) "On"    => enabled
         * swoole.use_shortname = "on"       => string(2) "on"    => enabled
         * swoole.use_shortname = 1          => string(1) "1"     => enabled
         * swoole.use_shortname = "1"        => string(1) "1"     => enabled
         * swoole.use_shortname = 2          => string(1) "1"     => enabled
         * swoole.use_shortname = false      => string(0) ""      => disabled
         * swoole.use_shortname = "false"    => string(5) "false" => disabled
         * swoole.use_shortname = off        => string(0) ""      => disabled
         * swoole.use_shortname = Off        => string(0) ""      => disabled
         * swoole.use_shortname = "off"      => string(3) "off"   => disabled
         * swoole.use_shortname = "Off"      => string(3) "Off"   => disabled
         * swoole.use_shortname = 0          => string(1) "0"     => disabled
         * swoole.use_shortname = "0"        => string(1) "0"     => disabled
         * swoole.use_shortname = 00         => string(2) "00"    => disabled
         * swoole.use_shortname = "00"       => string(2) "00"    => disabled
         * swoole.use_shortname = ""         => string(0) ""      => disabled
         * swoole.use_shortname = " "        => string(1) " "     => disabled.
         */
        $useShortname = ini_get_all('swoole')['swoole.use_shortname']['local_value'];
        $useShortname = strtolower(trim(str_replace('0', '', $useShortname)));
        if (!in_array($useShortname, ['', 'off', 'false'], true)) {
            $output->writeln("<error>ERROR</error> Swoole short function names must be disabled before the server starts, please set swoole.use_shortname='Off' in your php.ini.");
            exit(SIGTERM);
        }
    }
}
