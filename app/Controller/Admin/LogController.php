<?php

namespace App\Controller\Admin;

use App\Middleware\AdminMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\Paginator\LengthAwarePaginator;
use Hyperf\Paginator\Paginator;
use Hyperf\Utils\Collection;

#[Middleware(AdminMiddleware::class)]
#[Controller(prefix:"/admin/server/logger")]
class LogController
{
    #[GetMapping(path:"")]
    public function index(){
        $currentPage = (int) request()->input('page', 1);
        $perPage = (int) request()->input('per_page', 15);

        $data = admin_log()->get();
        // 这里根据 $currentPage 和 $perPage 进行数据查询，以下使用 Collection 代替
        $collection = new Collection($data);

        $users = array_values($collection->forPage($currentPage, $perPage)->toArray());

        $page = new LengthAwarePaginator($users, count($data),$perPage, $currentPage);
        return view('admin.server.logger',['page' => $page]);
    }

    #[GetMapping(path:"{id}.html")]
    public function data($id){
        $data = admin_log()->db()->findById($id);
        $_token = $data['_token'];
        return view('admin.server.logger_data',['data' => $data,'_token' => $_token]);
    }
}