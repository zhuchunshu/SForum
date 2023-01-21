<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
use App\Server\CodeFecServer;
use Hyperf\Server\Event;
use Hyperf\Server\ServerInterface;
use Swoole\Constant;

return [
    'mode' => SWOOLE_PROCESS,
    'servers' => [
        [
            'name' => 'http',
            'type' => ServerInterface::SERVER_HTTP,
            'host' => (string) env('SERVER_WEB_DOMAIN', '0.0.0.0'),
            'port' => (int) env('SERVER_WEB_PORT', 9501),
            'sock_type' => SWOOLE_SOCK_TCP,
            'callbacks' => [
                Event::ON_REQUEST => [CodeFecServer::class, 'onRequest'],
            ],
        ],
        //        [
        //            'name' => 'socket-io',
        //            'type' => ServerInterface::SERVER_WEBSOCKET,
        //            'host' => (string) env('SERVER_WEB_DOMAIN', '0.0.0.0'),
        //            'port' => (int) env('SERVER_WS_PORT', 9502),
        //            'sock_type' => SWOOLE_SOCK_TCP,
        //            'callbacks' => [
        //                Event::ON_HAND_SHAKE => [Hyperf\WebSocketServer\Server::class, 'onHandShake'],
        //                Event::ON_MESSAGE => [Hyperf\WebSocketServer\Server::class, 'onMessage'],
        //                Event::ON_CLOSE => [Hyperf\WebSocketServer\Server::class, 'onClose'],
        //            ],
        //            'settings' => [
        //                'open_websocket_protocol' => false,
        //            ],
        //        ],
    ],
    'settings' => [
        Constant::OPTION_ENABLE_COROUTINE => true,
        Constant::OPTION_WORKER_NUM => swoole_cpu_num(),
        Constant::OPTION_PID_FILE => BASE_PATH . '/runtime/hyperf.pid',
        Constant::OPTION_OPEN_TCP_NODELAY => true,
        Constant::OPTION_MAX_COROUTINE => 100000,
        Constant::OPTION_OPEN_HTTP2_PROTOCOL => true,
        Constant::OPTION_MAX_REQUEST => 100000,
        Constant::OPTION_SOCKET_BUFFER_SIZE => 2 * 1024 * 1024,
        Constant::OPTION_BUFFER_OUTPUT_SIZE => 2 * 1024 * 1024,
        Constant::OPTION_PACKAGE_MAX_LENGTH => 100 * 1024 * 1024,
        // 静态资源
        'document_root' => BASE_PATH . '/public',
        'enable_static_handler' => env('ENABLE_STATIC_HANDLER', true),
    ],
    'callbacks' => [
        Event::ON_WORKER_START => [Hyperf\Framework\Bootstrap\WorkerStartCallback::class, 'onWorkerStart'],
        Event::ON_PIPE_MESSAGE => [Hyperf\Framework\Bootstrap\PipeMessageCallback::class, 'onPipeMessage'],
        Event::ON_WORKER_EXIT => [Hyperf\Framework\Bootstrap\WorkerExitCallback::class, 'onWorkerExit'],
    ],
];
