<?php

namespace App\Plugins\User\src\Event;

// 退出登陆
class Logout
{
	/**
	 * 用户id
	 * @var int|string
	 */
	public $user_id;
	
	public function __construct($user_id){
		$this->user_id = $user_id;
	}
}