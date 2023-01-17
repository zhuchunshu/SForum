<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Controller\Admin;

use App\Middleware\AdminMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\Paginator\LengthAwarePaginator;
use Hyperf\Utils\Collection;
use Hyperf\Utils\Str;
use Swoole\Coroutine\System;
use Symfony\Component\Finder\Finder;

#[Middleware(AdminMiddleware::class)]
#[Controller(prefix: '/admin/server/backup')]
class BackupController
{
    #[GetMapping(path: '')]
    public function index()
    {
        if (! is_dir(BASE_PATH . '/runtime/backup')) {
            System::exec('cd ' . BASE_PATH . '/runtime' . '&& mkdir ' . 'backup');
        }
        return view('admin.server.backup', ['page' => $this->page()]);
    }

    #[GetMapping(path: 'download')]
    public function download()
    {
        $path = request()->input('path', BASE_PATH . '/runtime/backup/backup.zip');
        if (! file_exists($path)) {
            return redirect()->back()->with('danger', '文件不存在')->go();
        }
        return response()->download($path);
    }

    #[GetMapping(path: 'delete')]
    public function delete()
    {
        $filename = request()->input('filename', );
        if (! $filename) {
            return redirect()->back()->with('danger', 'filename不能为空')->go();
        }
        $path = BASE_PATH . '/runtime/backup/' . $filename;
        if (! file_exists($path)) {
            return redirect()->back()->with('danger', '文件不存在')->go();
        }
        go(function () use ($path) {
            System::exec('rm -rf ' . $path);
        });
        return redirect()->back()->with('success', '删除任务已创建')->go();
    }

    #[GetMapping(path: 'create')]
    public function create()
    {
        go(function () {
            backup('backup-' . date('y-m-d') . '-' . Str::random());
        });
        return redirect()->back()->with('success', '任务已创建')->go();
    }

    private function page(): LengthAwarePaginator
    {
        $currentPage = (int) request()->input('page', 1);
        $perPage = (int) request()->input('per_page', 15);

        // 这里根据 $currentPage 和 $perPage 进行数据查询，以下使用 Collection 代替
        $collection = new Collection($this->getAllBackup());

        $data = array_values($collection->forPage($currentPage, $perPage)->toArray());
        return new LengthAwarePaginator($data, count($this->getAllBackup()), $perPage, $currentPage);
    }

    private function getAllBackup(): array
    {
        $path = BASE_PATH . '/runtime/backup';
        $files = [];
        $result = Finder::create()->in($path)->files()->name('*.zip');
        foreach ($result as $item) {
            $files[$item->getCTime()] = [
                'size' => $item->getSize(),
                'date' => date('Y-m-d H:i:s', $item->getCTime()),
                'filename' => $item->getFilename(),
                'path' => $item->getRealPath(),
            ];
        }
        krsort($files);
        return array_values($files);
    }
}
