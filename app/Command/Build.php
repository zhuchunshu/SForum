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
class Build extends HyperfCommand
{
    /**
     * @var ContainerInterface
     */
    protected ContainerInterface $container;

    public function __construct(ContainerInterface $container)
    {
        $this->container = $container;

        parent::__construct('Build');
    }

    public function configure()
    {
        parent::configure();
        $this->setDescription('Build');
    }

    public function handle()
    {
        $version = $this->ask("版本","v1.0.0");
	    $author = $this->ask("作者","zhuchunshu");
	    $link = $this->ask("链接","https://forum.runpod.cn");
	    $content = $this->replace($version, $author, $link);
	    file_put_contents(BASE_PATH."/build-info.php",$content);
		$this->info('Successfully');
    }
	
	private function replace(mixed $version, mixed $author, mixed $link): string
	{
		$content = file_get_contents(BASE_PATH."/app/Command/build-info.stub");
		return str_replace(array('author', '1.0', 'http://forum.runpod.cn'), array($author, $version, $link), $content);
	}
}
