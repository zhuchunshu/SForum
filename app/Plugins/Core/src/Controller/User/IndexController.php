<?php


namespace App\Plugins\Core\src\Controller\User;


use App\Plugins\Topic\src\Models\Topic;
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
     * @var \Psr\Log\LoggerInterface
     */
    public \Psr\Log\LoggerInterface $logger;

    /**
     * 强制验证邮箱
     */
    #[GetMapping(path: "/user/ver_email")]
    public function user_ver_email()
    {
        if(auth()->data()->email_ver_time){
            return redirect()->url("/")->with("info","你已验证邮箱,无需重复操作")->go();
        }
        return view("Core::user.ver_email");
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

    /**
     * 个人中心
     */
    #[GetMapping(path: "/user")]
    public function user(){
        return redirect()->url('/users/'.auth()->data()->username.".html")->go();
    }

    // 草稿
    #[GetMapping("/user/draft")]
    public function draft(){
        $title = "我的草稿";
        $page = Topic::query()
            ->where(['user_id' => auth()->id(),'status' => 'draft'])
            ->with("tag","user")
            ->orderBy("topping","desc")
            ->orderBy("id","desc")
            ->paginate(get_options("topic_home_num",15));

        return view("User::draft",["page" => $page]);
    }

    // 草稿
    #[GetMapping("/draft/{id}")]
    public function draft_show($id){
        if(!Topic::query()->where('id',$id)->exists()) {
            return admin_abort("页面不存在",404);
        }
        $data = Topic::query()
            ->where('id', $id)
            ->with("tag","user","topic_updated","update_user")
            ->first();
        $quanxian = false;
        if(auth()->id()==$data->user_id){
            $quanxian = true;
        }
        if(Authority()->check("admin_view_draft_topic")){
            $quanxian = true;
        }
        if($quanxian===false){
            return admin_abort("无权预览此草稿",401);
        }
        $shang = Topic::query()->where([['id','<',$id],['status','publish']])->select('title','id')->orderBy('id','desc')->first();
        $xia = Topic::query()->where([['id','>',$id],['status','publish']])->select('title','id')->orderBy('id','asc')->first();
        $sx = ['shang' => $shang,'xia' => $xia];
        return view('Core::topic.show.draft',['data' => $data,'get_topic' => $sx]);
    }
}