<?php

use Hyperf\HttpServer\Router\Router;

Router::addServer('websocket', function () {
	// 用户私信
	Router::addGroup('/User/', function(){
		Router::get('Pm',\App\Plugins\User\src\WebSocket\Pm::class);
	});
});