<?php


namespace App\Plugins\User\src\Controller;

use App\Plugins\User\src\Models\User;
use Hyperf\HttpMessage\Stream\SwooleStream;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Multiavatar;
use Psr\Http\Message\ResponseInterface;
use App\Plugins\User\src\Models\UserClass as UserClassModel;

#[Controller]
class UserController
{
    /**
     * 用户列表
     */
    #[GetMapping(path:"/users")]
    public function list(){
        $count = User::query()->count();
        $page = User::query()->paginate(30);
        return view("plugins.User.list",['page' => $page,'count'=>$count]);
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

    #[GetMapping(path:"/users/group/{id}.html")]
    public function group_data($id): ResponseInterface
    {
        if(!UserClassModel::query()->where("id",$id)->count()){
            return admin_abort("页面不存在","404");
        }
        $userCount = User::query()->where("class_id",$id)->count();
        $data = UserClassModel::query()->where("id",$id)->first();
        $user = User::query()->where("class_id",$id)->paginate(30);
        return view("plugins.User.group_data",['userCount' => $userCount,'data' => $data,'user'=>$user]);
    }

    #[GetMapping(path:"/user/multiavatar/{user_id}")]
    public function user_multi_avatar($user_id){
        $ud = \App\Plugins\User\src\Models\User::query()->where("id",$user_id)->first();
        $img = new Multiavatar();
        $img = $img($ud->username, null, null);
        return ResponseObj()->withBody(SwooleStream($img))->withHeader("content-type","image/svg+xml; charset=utf-8");
    }
}