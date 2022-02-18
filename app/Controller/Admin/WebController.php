<?php

namespace App\Controller\Admin;

use App\Middleware\AdminMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;

#[Controller(prefix: "/admin")]
#[Middleware(AdminMiddleware::class)]
class WebController
{
	#[GetMapping(path:"Releases/{id}")]
	public function Releases($id){
		return view("admin.panel.Releases",['id' => $id]);
	}
}