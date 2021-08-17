<?php


namespace App\Plugins\User\src\Controller;

use App\Plugins\User\src\Models\User;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

#[Controller]
class UserController
{
    /**
     * 用户列表
     */
    #[GetMapping(path:"/users")]
    public function list(){
        $page = User::query()->paginate(30);
        return view("plugins.User.list",['page' => $page]);
    }
}