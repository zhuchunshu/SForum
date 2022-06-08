<?php

namespace App\Plugins\Core\src\Controller\PrivateMessage;

use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

class IndexController
{
	#[GetMapping(path:"notice")]
	public function index(){
		return 1;
	}
}