<?php

use Hyperf\HttpServer\Router\Router;

Router::addServer('websocket', function () {
    Router::get('/core', \App\Plugins\Core\src\Controller\WebSocket\CoreController::class);
});