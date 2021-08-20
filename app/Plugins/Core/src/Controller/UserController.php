<?php
namespace App\Plugins\Core\src\Controller;

use App\Plugins\Core\src\Request\LoginRequest;
use App\Plugins\Core\src\Request\RegisterRequest;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UsersOption;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use HyperfExt\Hashing\Hash;
use Illuminate\Support\Str;

#[Controller()]
#[Middleware(\App\Plugins\User\src\Middleware\AuthMiddleware::class)]
class UserController
{

    #[GetMapping(path:"/login")]
    public function login(): \Psr\Http\Message\ResponseInterface
    {

        return view("plugins.Core.user.sign",['title' => "登陆","view" => "plugins.Core.user.login"]);
    }

    #[GetMapping(path:"/register")]
    public function register(): \Psr\Http\Message\ResponseInterface
    {
        return view("plugins.Core.user.sign",['title' => "注册","view" => "plugins.Core.user.register"]);
    }

    #[PostMapping(path: "/register")]
    public function register_post(RegisterRequest $request): array
    {
        $data = $request->validated();
        if($data['password'] !== $data['cfpassword']){
            return Json_Api(403,false,['msg' => "The two passwords are inconsistent 两次输入密码不一致"]);
        }
        if(!plugins_core_captcha()->validate($data['captcha'])){
            return Json_Api(403,false,['msg' => "Verification failed, calculation result is wrong 验证失败，计算结果错误"]);
        }
        $userOption = UsersOption::query()->create(["qianming" => "这个人没有签名"]);
        User::query()->create([
            "username" => $data['username'],
            "email" => $data['email'],
            "password" => Hash::make($data['password']),
            "class_id" => get_options("plugins_core_user_reg_defuc",1),
            "_token" => Str::random(),
            "options_id" => $userOption->id
        ]);
        return Json_Api(200,true,['msg' => '注册成功!']);
    }

    #[PostMapping(path: "/login")]
    public function login_post(LoginRequest $request): ?array
    {
        $data = $request->validated();
        $email = $data['email'];
        $password = $data['password'];
        if(auth()->SignIn($email,$password)){
            return Json_Api(200,true,['msg' => '登陆成功!']);
        }

        return Json_Api(403,false,['msg' => '登陆失败,账号或密码错误']);
    }

    #[PostMapping(path: "/logout")]
    public function logout(): array
    {
        session()->remove("auth");
        session()->remove("auth_data");
        return Json_Api(200,true,['msg' => '退出登陆成功!','url' => '/login']);
    }


}