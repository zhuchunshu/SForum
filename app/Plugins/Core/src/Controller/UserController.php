<?php
namespace App\Plugins\Core\src\Controller;

use App\Plugins\Core\src\Request\LoginRequest;
use App\Plugins\Core\src\Request\RegisterRequest;
use App\Plugins\User\src\Auth;
use App\Plugins\User\src\Models\User;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\PostMapping;
use HyperfExt\Hashing\Hash;
use Illuminate\Support\Str;

/**
 * Class UserController
 * @Controller
 * @package App\Plugins\Core\src\Controller
 */
class UserController
{
    /**
     * @GetMapping(path="/login")
     */
    public function login(){
        return view("plugins.Core.user.sign",['title' => "登陆","view" => "plugins.Core.user.login"]);
    }

    /**
     * @GetMapping(path="/register")
     */
    public function register(){
        return view("plugins.Core.user.sign",['title' => "注册","view" => "plugins.Core.user.register"]);
    }

    /**
     * @PostMapping(path="/register")
     */
    public function register_post(RegisterRequest $request){
        $data = $request->validated();
        if($data['password'] != $data['cfpassword']){
            return Json_Api(403,false,['msg' => "The two passwords are inconsistent 两次输入密码不一致"]);
        }
        if(!plugins_core_captcha()->validate($data['captcha'])){
            return Json_Api(403,false,['msg' => "Verification failed, calculation result is wrong 验证失败，计算结果错误"]);
        }
        User::query()->create([
            "username" => $data['username'],
            "email" => $data['email'],
            "password" => Hash::make($data['password']),
            "class_id" => get_options("plugins_core_user_reg_defuc",1),
            "_token" => Str::random()
        ]);
        return Json_Api(200,true,['msg' => '注册成功!']);
    }

    /**
     * @PostMapping(path="/login")
     */
    public function login_post(LoginRequest $request){
        $data = $request->validated();
        $email = $data['email'];
        $password = $data['password'];
        if(Auth::SignIn($email,$password)){
            return Json_Api(200,true,['msg' => '登陆成功!']);
        }else{
            return Json_Api(403,false,['msg' => '登陆失败,账号或密码错误']);
        }
    }
}