<?php


namespace App\Plugins\User\src\Controller;

use App\Plugins\User\src\Models\User;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Psr\Http\Message\ResponseInterface;

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

    /**
     * 用户信息
     * @param $username
     * @return ResponseInterface
     */
    #[GetMapping(path:"/users/{username}.html")]
    public function data($username){
        if(!User::query()->where("username",$username)->count()){
            return admin_abort("用户名为:".$username."的用户不存在");
        }
        $data = User::query()->with("Class","Options")->where("username",$username)->first();
        return view("plugins.User.data",['data'=>$data]);
    }
}