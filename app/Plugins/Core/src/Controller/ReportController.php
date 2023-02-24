<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Controller;

use App\Plugins\Core\src\Models\Report;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

#[Controller(prefix: '/report')]
class ReportController
{
    #[GetMapping(path: '')]
    public function index()
    {
        if(!Authority()->check('admin_report')){
            return admin_abort('无权访问');
        }
        $page = Report::query()->orderBy('created_at', 'desc')->paginate(15);
        return view('App::report.index', ['page' => $page]);
    }

    #[GetMapping(path: '{id}.html')]
    public function data($id)
    {
        if(!Authority()->check('admin_report')){
            return admin_abort('无权访问');
        }

        if (! Report::query()->where('id', $id)->exists()) {
            return admin_abort('页面不存在', 404);
        }
        $data = Report::query()->where('id', $id)->first();
        return view('App::report.data', ['data' => $data]);
    }
}
