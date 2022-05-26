<?php

declare(strict_types=1);

namespace App\Command;

use App\CodeFec\Install;
use Hyperf\Command\Annotation\Command;
use Hyperf\Command\Command as HyperfCommand;
use Hyperf\Utils\Arr;
use Hyperf\Utils\Str;
use Hyperf\Watcher\{Option,Watcher};
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
        if(file_exists(BASE_PATH."/app/CodeFec/storage/install.lock") || (new Install($this->output,$this))->getStep()>=5){
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
	
			// 复制文件
			$this->copyFile();
			
	        $watcher->run();
        }else{
	        $install = make(Install::class, [
		        'output' => $this->output,
		        'command' => $this
	        ]);
	
	        $install->run();
        }
    }
	
	private function copyFile(): void
	{
		foreach(Itf()->get('copyFile') as $file){
			if(Arr::has($file, 'from') && Arr::has($file, 'to') && is_dir($file['from']) && is_dir($file['to'])) {
				copy($file['from'],$file['to']);
			}
		}
		$this->info('Copy File Successfully!');
	}
}
