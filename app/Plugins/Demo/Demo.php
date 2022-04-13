<?php

namespace App\Plugins\Demo;

use App\CodeFec\Annotation\RouteRewrite;

class Demo
{
	#[RouteRewrite(route:"/",callback:"index")]
	public function index(){
		return 1;
	}
}