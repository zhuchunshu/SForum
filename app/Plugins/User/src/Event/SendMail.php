<?php

namespace App\Plugins\User\src\Event;

class SendMail
{
	public int|string $user_id;
	
	public string $title;
	
	public null|string $action;
	
	public function __construct($user_id,$title,$action)
	{
		$this->user_id = $user_id;
		
		$this->title = $title;
		
		$this->action = $action;
	}
}