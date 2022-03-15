<?php

namespace App\Plugins\User\src\Jobs;

use App\Plugins\User\src\Models\User;
use Hyperf\AsyncQueue\Annotation\AsyncQueueMessage;

class SendMail
{
	#[AsyncQueueMessage]
	public function handler($user_id,$title,$action){
		/**
		 * 发送邮件通知
		 */
		// 接受者邮箱
		$email = User::query()->where('id', $user_id)->first()->email;
		$mail = Email();
		$url =url($action);
		
		// 判断用户是否愿意接收通知
		if(user_notice()->check("email",$user_id)===false){
			// 执行发送
			go(function() use ($title,$email,$mail,$url){
				$mail->addAddress($email);
				$mail->Subject = "【".get_options("web_name")."】 你有一条新通知!";
				$mail->Body    = <<<HTML
<h3>标题: {$title}</h3>
<p>链接: <a href="{$url}">{$url}</a></p>
HTML;
				$mail->send();
			});
		}
	}
}