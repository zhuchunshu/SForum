<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Command;

use Hyperf\Command\Annotation\Command;
use Hyperf\Command\Command as HyperfCommand;
use Hyperf\Contract\ConfigInterface;
use Hyperf\ViewEngine\Compiler\CompilerInterface;
use Psr\Container\ContainerInterface;
use Symfony\Component\Finder\Finder;

#[Command]
class ViewEngineCache extends HyperfCommand
{
    /**
     * @var ContainerInterface
     */
    protected $container;
    
    public function __construct(ContainerInterface $container)
    {
        $this->container = $container;
        parent::__construct('CodeFec:view-engine-cache');
    }
    
    public function configure()
    {
        parent::configure();
        $this->setDescription('view-engine-cache');
    }
    
    public function handle()
    {
        $plugins = BASE_PATH . '/app/Plugins/';
        $all = [];
        foreach (plugins()->getEnPlugins() as $plugin) {
            if (is_dir($plugins . $plugin)) {
                $all[] = $plugins . $plugin . '/';
            }
        }
        $all[] = BASE_PATH . '/app/Themes/';
        $all[] = $this->container->get(ConfigInterface::class)->get('view.config.view_path');
        foreach ($all as $item) {
            $this->make($item);
        }
    }
    /**
     * 生成缓存.
     * @param $dir
     * @throws \Psr\Container\ContainerExceptionInterface
     * @throws \Psr\Container\NotFoundExceptionInterface
     */
    private function make($dir)
    {
        $finder = Finder::create()->in($dir)->files()->name('*.blade.php');
        $compiler = $this->container->get(CompilerInterface::class);
        foreach ($finder as $item) {
            $compiler->compile($item->getRealPath());
            $this->info(sprintf('File `%s` cache generation completed', $item->getRealPath()));
        }
    }
}