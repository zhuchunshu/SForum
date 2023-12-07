<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Middleware;

use Hyperf\Stringable\Str;
use Psr\Container\ContainerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface;

class InstallMiddleware implements MiddlewareInterface
{
    /**
     * @var ContainerInterface
     */
    protected $container;

    public function __construct(ContainerInterface $container)
    {
        $this->container = $container;
    }

    public function process(ServerRequestInterface $request, RequestHandlerInterface $handler): ResponseInterface
    {
        if (!file_exists(BASE_PATH . '/app/CodeFec/storage/install.lock') && request()->path() !== 'install' && !Str::is($this->clean_str('install/*'), $this->clean_str(request()->path()))) {
            return response()->redirect('/install');
        }
        if (file_exists(BASE_PATH . '/app/CodeFec/storage/update.lock') && request()->path() !== 'admin' && !Str::is($this->clean_str('admin/*'), $this->clean_str(request()->path()))) {
            return admin_abort('系统升级中..', 200);
        }
        if ((request()->path() === 'install') && file_exists(BASE_PATH . '/app/CodeFec/storage/install.lock')) {
            return admin_abort(['msg' => '页面不存在'], 404);
        }
        return $handler->handle($request);
    }

    public function clean_str($str): array | string
    {
        return str_replace('/', '_', $str);
    }
}
