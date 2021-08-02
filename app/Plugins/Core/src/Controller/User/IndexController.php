<?php


namespace App\Plugins\Core\src\Controller\User;


use App\Plugins\User\src\Middleware\LoginMiddleware;
use App\Plugins\User\src\Models\User;
use Exception;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use HyperfExt\Mail\Mail;
use Psr\Http\Message\ResponseInterface;

#[Controller]
#[Middleware(LoginMiddleware::class)]
class IndexController
{
    /**
     * @throws Exception
     */
    #[GetMapping(path: "/test")]
    public function test()
    {
        return request()->fullUrl();
    }

    /**
     * 强制验证邮箱
     */
    #[GetMapping(path: "/user/ver_email")]
    public function user_ver_email()
    {
        if(auth()->data()->email_ver_time){
            return redirect()->url("/")->with("info","你已验证邮箱,无需重复操作")->go();
        }
        return view("plugins.Core.user.ver_email");
    }

    #[PostMapping(path: "/user/ver_email")]
    public function user_ver_email_post(){
        $send = request()->input('send',null);
        $captcha = request()->input('captcha',null);
        if($send==="send"){
            if(!core_user_ver_email()->ifsend()){
                return redirect()->back()->with("danger","冷却期间,请".core_user_ver_email()->sendTime()."秒后再试")->go();
            }
            core_user_ver_email()->send(auth()->data()->email);
            return redirect()->back()->with("success","验证码邮件已发送")->go();
        }
        if(!$captcha){
            return redirect()->back()->with("danger","请填写验证码")->go();
        }
        if(!core_user_ver_email()->check($captcha)){
            return redirect()->back()->with("danger","验证码错误")->go();
        }
        User::query()->where("id",auth()->data()->id)->update([
           "email_ver_time" => date("Y-m-d H:i:s")
        ]);
        session()->set("auth_data",User::query()->where("id",session()->get('auth'))->first());
        return redirect()->url("/")->with("success","验证通过!")->go();
    }
}