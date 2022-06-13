<?php

namespace App\Plugins\Core\src\Controller\User;

use App\Plugins\User\src\Middleware\LoginMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;

#[Middleware(LoginMiddleware::class)]
#[Controller(prefix:"/notice")]
class NoticeController
{
	#[GetMapping(path:"")]
	public function index()
	{
		return view("App::notice.index");
	}
}