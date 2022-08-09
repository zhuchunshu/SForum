<?php

declare(strict_types=1);

namespace App\Command;

use Hyperf\Command\Command as HyperfCommand;
use Hyperf\Command\Annotation\Command;
use Hyperf\Watcher\Option;
use Hyperf\Watcher\Watcher;
use Psr\Container\ContainerInterface;
use Symfony\Component\Console\Application;
use Symfony\Component\Console\Input\ArrayInput;
use Symfony\Component\Console\Input\InputOption;
use Symfony\Component\Console\Output\NullOutput;

/**
 * @Command
 */
class Docker extends HyperfCommand
{
	protected ContainerInterface $container;
	
	public function __construct(ContainerInterface $container)
	{
		parent::__construct('Docker');
		
		$this->container = $container;
		$this->setDescription('CodeFec Docker Start Server');
		$this->addOption('file', 'F', InputOption::VALUE_OPTIONAL | InputOption::VALUE_IS_ARRAY, '', []);
		$this->addOption('dir', 'D', InputOption::VALUE_OPTIONAL | InputOption::VALUE_IS_ARRAY, '', []);
		$this->addOption('no-restart', 'N', InputOption::VALUE_NONE, 'Whether no need to restart server');
	}
	
	public function handle()
	{
		if(file_exists(BASE_PATH."/app/CodeFec/storage/install.lock") || $this->getStep()!=4){
			if(stripos(system_name(), "Linux") !== false){
				\Swoole\Coroutine\System::exec("yes yes | composer du");
			}else{
				\Swoole\Coroutine\System::exec("composer du");
			}
			
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
		}else{
			
			if(!file_exists(BASE_PATH."/.env")){
				copy(BASE_PATH."/.env.example",BASE_PATH."/.env");
			}
			
			$command = 'migrate';
			
			$params = ["command" => $command];
			
			$input = new ArrayInput($params);
			$output = new NullOutput();
			
			$container = \Hyperf\Utils\ApplicationContext::getContainer();
			
			/** @var Application $application */
			$application = $container->get(\Hyperf\Contract\ApplicationInterface::class);
			$application->setAutoExit(false);
			
			$exitCode = $application->run($input, $output);
			
			
			file_put_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock",6);
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
	public function getStep()
	{
		if (!is_dir(BASE_PATH . "/app/CodeFec/storage")) {
			\Swoole\Coroutine\System::exec("mkdir " . BASE_PATH . "/app/CodeFec/storage");
		}
		// 创建文件
		if(!file_exists(BASE_PATH."/app/CodeFec/storage/install.step.lock")){
			file_put_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock",1);
		}
		if(!@file_get_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock")){
			$step = 1;
		}else{
			$step = (int)file_get_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock");
		}
		return $step;
	}
	
}
