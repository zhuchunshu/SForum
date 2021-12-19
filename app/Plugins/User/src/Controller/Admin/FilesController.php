<?php

namespace App\Plugins\User\src\Controller\Admin;

use App\Middleware\AdminMiddleware;
use App\Plugins\User\src\Models\UserUpload;
use Hyperf\DbConnection\Db;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;

#[Controller(prefix:"/admin/users/files")]
#[Middleware(AdminMiddleware::class)]
class FilesController
{
    #[GetMapping(path:"")]
    public function index(){
        $page = UserUpload::query()->with("user")->paginate(30);
        return view("User::Admin.Files.index",['page' => $page]);
    }
}