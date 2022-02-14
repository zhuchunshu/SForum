<?php

namespace App\Plugins\User\src\Event;

// 注册成功
class AfterRegister
{
	/**
	 * 用户信息
	 * @var object|array
	 */
	public object|array $data;
	public function __construct($data){
		$this->data = $data;
	}
}