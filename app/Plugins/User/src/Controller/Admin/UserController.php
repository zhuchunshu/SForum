<?php

namespace App\Plugins\User\src\Controller\Admin;

use App\Plugins\User\src\Models\User;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;

#[Controller]
#[Middleware(\App\Middleware\AdminMiddleware::class)]
class UserController
{
    #[GetMapping(path:"/admin/users")]
    public function index(){
        $page = User::query()->with("class")->paginate(15);
        return view("plugins.User.Admin.Users.index",['page' => $page]);
    }

    #[PostMapping(path:"/admin/users/update/username")]
    public function update_username(){
        $user_id = request()->input("user_id");
        $username = request()->input("username");
        if(!$user_id){
            return Json_Api(403,false,['msg' => '请求的用户id不能为空']);
        }
        if(!$username){
            return Json_Api(403,false,['msg' => '请求的用户名不能为空']);
        }
        if(!User::query()->where("id",$user_id)->exists()){
            return Json_Api(403,false,['msg' => 'ID为:'.$user_id."的用户不存在"]);
        }
        if(User::query()->where("username",$username)->exists()){
            return Json_Api(403,false,['msg' => '此用户名已被使用']);
        }
        User::query()->where("id",$user_id)->update([
           "username" => $username
        ]);
        return Json_Api(200,true,['msg' => '修改成功!']);
    }

    #[PostMapping(path:"/admin/users/update/email")]
    public function update_email(){
        $user_id = request()->input("user_id");
        $email = request()->input("email");
        if(!$user_id){
            return Json_Api(403,false,['msg' => '请求的用户id不能为空']);
        }
        if(!$email){
            return Json_Api(403,false,['msg' => '请求的邮箱不能为空']);
        }
        if(!User::query()->where("id",$user_id)->exists()){
            return Json_Api(403,false,['msg' => 'ID为:'.$user_id."的用户不存在"]);
        }
        if(User::query()->where("email",$email)->exists()){
            return Json_Api(403,false,['msg' => '此邮箱已被使用']);
        }
        User::query()->where("id",$user_id)->update([
            "email" => $email,
            "email_ver_time" => null
        ]);
        return Json_Api(200,true,['msg' => '修改成功! 用户重新登陆并验证邮箱后生效']);
    }
}