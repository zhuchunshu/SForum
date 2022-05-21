<?php

declare(strict_types=1);

namespace App\Middleware;

use Hyperf\Contract\TranslatorInterface;
use Psr\Container\ContainerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\RequestHandlerInterface;

class Translator implements MiddlewareInterface
{
    /**
     * @var ContainerInterface
     */
    protected ContainerInterface $container;
	
	/**
	 * @var TranslatorInterface
	 */
	protected TranslatorInterface $translator;

	
    public function __construct(ContainerInterface $container ,TranslatorInterface $translator)
    {
        $this->container = $container;
		$this->translator = $translator;
    }

    public function process(ServerRequestInterface $request, RequestHandlerInterface $handler): ResponseInterface
    {
		$this->translator->setLocale(session()->get('lang',get_options('language','zh_CN')));
        return $handler->handle($request);
    }
}