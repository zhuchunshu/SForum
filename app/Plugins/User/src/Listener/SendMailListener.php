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
		
	}
}