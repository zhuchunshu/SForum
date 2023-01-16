<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */

namespace App\Controller\Admin\Hook;

use App\Middleware\AdminMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\Paginator\LengthAwarePaginator;
use Hyperf\Utils\Collection;
use Hyperf\Utils\Str;
use Hyperf\ViewEngine\Contract\FactoryInterface;
use Swoole\Coroutine\System;
use Symfony\Component\Finder\Finder;

#[Controller(prefix: '/admin/hook/components')]
#[Middleware(AdminMiddleware::class)]
class ComponentController
{
    #[GetMapping(path: '')]
    public function index()
    {
        if (!is_dir(BASE_PATH . '/resources/views/customize/component')) {
            System::exec('cd ' . BASE_PATH . '/resources/views' . '&& mkdir ' . 'customize && cd customize && mkdir component');
        }
        $page = $this->page();
        return view('admin.setting.hook.components', ['page' => $page]);
    }

    private function getAllComponents(): array
    {
        $path = BASE_PATH . '/resources/views/customize/component/';
        $files = [];
        $result = Finder::create()->in($path)->files()->name('*.blade.php');
        $id = 1;
        foreach ($result as $item) {
            $Filename = $item->getRelativePath();
            $viewName = Str::before($item->getFilename(), '.');
            $RelativePath = $item->getRelativePath();
            if ($RelativePath) {
                $RelativePath = explode('/', $RelativePath);
                $viewName = implode('.', $RelativePath) . '.' . $viewName;
            }
            $files[] = [
                'id' => $id++,
                'file_name' => $Filename,
                'view' => 'customize.component.' . $viewName,
                'import' => $viewName,
                'path' => $item->getRealPath(),
                'remark' => $this->get_remark($item->getRealPath()) ?: "暂无",
            ];
        }
        return array_values($files);
    }

    private function page(): LengthAwarePaginator
    {
        $currentPage = (int)request()->input('page', 1);
        $perPage = (int)request()->input('per_page', 15);

        // 这里根据 $currentPage 和 $perPage 进行数据查询，以下使用 Collection 代替
        $collection = new Collection($this->getAllComponents());

        $data = array_values($collection->forPage($currentPage, $perPage)->toArray());
        return new LengthAwarePaginator($data, count($this->getAllComponents()), $perPage, $currentPage);
    }

    /**
     * get remark
     * @param $file
     * @return string|null
     */
    private function get_remark($file): string|null
    {
        $content = fopen($file, 'rb');
        $content = fgets($content);
        $result = [];
        preg_match_all('/(?:\\{{--)(.*)(?:\\--}})/i', $content, $result);
        return @$result[1][0] ?: null;
    }

    #[GetMapping(path:"preview")]
    public function preview(){
        $component = request()->input('component');
        $component = 'customize.component.'.$component;
        $container = \Hyperf\Utils\ApplicationContext::getContainer();
        $factory = $container->get(FactoryInterface::class);
        if(!$factory->exists($component)){
            return admin_abort('小部件不存在',403);
        }
        return view('admin.setting.hook.preview', ['view' => $component]);
    }
}
