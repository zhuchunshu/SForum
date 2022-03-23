<?php

namespace App\Plugins\User\src\Listener;

use App\Plugins\User\src\Event\SendMail;
use App\Plugins\User\src\Models\User;
use Hyperf\Event\Annotation\Listener;
use Hyperf\Event\Contract\ListenerInterface;

#[Listener]
class SendMailListener implements ListenerInterface
{
	public function listen(): array
	{
		// 返回一个该监听器要监听的事件数组，可以同时监听多个事件
		return [
			SendMail::class,
		];
	}
	
	/**
	 * @param object $event
	 * @throws \PHPMailer\PHPMailer\Exception
	 */
	public function process(object $event)
	{
		$email = User::query()->where('id', $event->user_id)->first()->email;
		$mail = Email();
		$url =url($event->action);
		// 判断用户是否愿意接收通知
		if(user_notice()->check("email",$event->user_id)===true){
			// 执行发送
			$title = $event->title;
			go(function() use ($title,$mail,$url,$email){
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