<?php

namespace App\Plugins\Mail\src\Controller;

use App\Middleware\AdminMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Utils\Str;
use PHPMailer\PHPMailer\Exception;

#[Middleware(AdminMiddleware::class)]
#[Controller(prefix:"/admin/mail")]
class AdminController
{
	#[GetMapping(path:"")]
	public function index(){
		return view("Mail::admin");
	}
	
	#[PostMapping(path:"")]
	public function submit(){
		$email = request()->input('email');
		if(!$email){
			return redirect()->back()->with("danger","请求参数不足!")->go();
		}

		$mail = Email();
		$mail->addAddress($email);
		$random = Str::random();
		$mail->Subject = "【".get_options("web_name")."】 查看你的验证码!";
		$mail->Body    = <<<HTML
当你收到此条消息,说明邮件发送成功! {$random}
HTML;
		if($mail->send()){
			return redirect()->back()->with("success","发送成功!")->go();
		}
		return redirect()->back()->with("danger","发送失败!")->go();
	}
}