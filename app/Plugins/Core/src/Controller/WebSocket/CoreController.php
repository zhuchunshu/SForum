<?php
declare(strict_types=1);

namespace App\Plugins\Core\src\Controller\WebSocket;

use App\Plugins\User\src\Models\UsersAuth;
use App\Plugins\User\src\Models\UsersNotice;
use Hyperf\Contract\OnCloseInterface;
use Hyperf\Contract\OnMessageInterface;
use Hyperf\Contract\OnOpenInterface;
use Hyperf\WebSocketServer\Context;
use Swoole\Http\Request;
use Swoole\Server;
use Swoole\Timer;
use Swoole\Websocket\Frame;
use Swoole\WebSocket\Server as WebSocketServer;

class CoreController implements OnMessageInterface, OnOpenInterface, OnCloseInterface
{
    public function onMessage($server, Frame $frame): void
    {
        $token = Context::get('login-token');
//        if($token && UsersAuth::query()->where("token",$token)->exists()){
//            $server->push($frame->fd, json_encode([
//                "user" => [
//                    "online" => redis()->sMembers("user_online")
//                ]
//            ]));
//        }

    }

    public function onClose($server, int $fd, int $reactorId): void
    {
        $token = Context::get('login-token');

//        var_dump('closed');
    }

    public function onOpen($server, Request $request): void
    {
        $token = $request->get['login-token'];
        Context::set('login-token', $token);

    }
}
