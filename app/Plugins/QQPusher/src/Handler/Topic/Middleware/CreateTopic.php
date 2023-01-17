<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\QQPusher\src\Handler\Topic\Middleware;

use App\Plugins\QQPusher\src\Jobs\Service\SendMessageService;
use App\Plugins\Topic\src\Annotation\Topic\CreateLastMiddleware;
use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;
use Hyperf\Di\Annotation\Inject;

#[CreateLastMiddleware]
class CreateTopic implements MiddlewareInterface
{
    /**
     * 推送
     * @Inject
     */
    protected SendMessageService $SendMessageService;

    public function handler($data, \Closure $next)
    {
        $topic_id = $data['topic_id'];
        $this->SendMessageService->push($topic_id);
        return $next($data);
    }
}
