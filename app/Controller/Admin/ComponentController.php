<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Controller\Admin;

use App\Middleware\AdminMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\Paginator\LengthAwarePaginator;
use Hyperf\Utils\Collection;
use Hyperf\Utils\Str;
use Symfony\Component\Finder\Finder;

#[Controller(prefix: '/admin/setting/components')]
#[Middleware(AdminMiddleware::class)]
class ComponentController
{
    #[GetMapping(path: '')]
    public function index()
    {
        $page = $this->page();
        return view('admin.setting.components', ['page' => $page]);
    }

    private function getAllComponents(): array
    {
        $path = BASE_PATH . '/resources/views/customize/component/';
        $files = [];
        $result = Finder::create()->in($path)->files()->name('*.blade.php');
        foreach ($result as $item) {
            $Filename = $item->getFilename();
            $viewName = Str::before($item->getFilename(), '.');
            $RelativePath = $item->getRelativePath();
            if ($RelativePath) {
                $RelativePath = explode('/', $RelativePath);
                $viewName = implode('.', $RelativePath) . '.' . $viewName;
            }
            $files[] = [
                'id' => Str::random(10),
                'file_name' => $Filename,
                'view' => 'customize.component.' . $viewName,
                'import' => $viewName,
                'path' => $item->getRealPath(),
            ];
        }
        return array_values($files);
    }

    private function page(): LengthAwarePaginator
    {
        $currentPage = (int) request()->input('page', 1);
        $perPage = (int) request()->input('per_page', 15);

        // 这里根据 $currentPage 和 $perPage 进行数据查询，以下使用 Collection 代替
        $collection = new Collection($this->getAllComponents());

        $data = array_values($collection->forPage($currentPage, $perPage)->toArray());
        return new LengthAwarePaginator($data, count($this->getAllComponents()), $perPage, $currentPage);
    }
}
