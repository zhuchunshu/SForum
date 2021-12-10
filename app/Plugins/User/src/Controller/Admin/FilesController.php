<?php

namespace App\Plugins\User\src\Controller\Admin;

use App\Plugins\User\src\Models\UserUpload;
use Hyperf\DbConnection\Db;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

#[Controller(prefix:"/admin/users/files")]
class FilesController
{
    #[GetMapping(path:"")]
    public function index(){
        $page = UserUpload::query()->with("user")->paginate(30);
        return view("User::Admin.Files.index",['page' => $page]);
    }
}