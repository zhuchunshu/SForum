<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Exception\Handler;

use Hyperf\ExceptionHandler\ExceptionHandler;
use Hyperf\HttpMessage\Stream\SwooleStream;
use Hyperf\Validation\ValidationException;
use Psr\Http\Message\ResponseInterface;
use Throwable;

class ValidationExceptionHandler extends ExceptionHandler
{
    public function handle(Throwable $throwable, ResponseInterface $response): ResponseInterface
    {
        if (request()->input('Redirect', null)) {
            $this->stopPropagation();
            /** @var ValidationException $throwable */
            $body = $throwable->validator->errors()->all();
            session()->set('errors', $body);
            return response()->redirect(request()->input('Redirect', null));
        }

        $this->stopPropagation();
        /** @var ValidationException $throwable */
        $body = $throwable->validator->errors()->all();
//        if (! $response->hasHeader('content-type')) {
//            $response = $response->withAddedHeader('content-type', 'text/plain; charset=utf-8');
//        }
        $container = \Hyperf\Context\ApplicationContext::getContainer();
        $responses = $container->get(\Hyperf\HttpServer\Contract\ResponseInterface::class);

        return $responses->json(json_api($throwable->status, false, $body));
        //return $response->withStatus($throwable->status)->withBody(new SwooleStream($body));
    }

    public function isValid(Throwable $throwable): bool
    {
        return $throwable instanceof ValidationException;
    }
}
