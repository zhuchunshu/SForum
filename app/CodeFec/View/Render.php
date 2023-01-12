<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\CodeFec\View;

use Hyperf\Contract\ConfigInterface;
use Hyperf\HttpMessage\Stream\SwooleStream;
use Hyperf\Utils\Context;
use Hyperf\View\Engine\EngineInterface;
use Hyperf\View\Engine\NoneEngine;
use Hyperf\View\Exception\EngineNotFindException;
use Hyperf\View\Exception\RenderException;
use Hyperf\View\Mode;
use Hyperf\View\RenderInterface;
use Psr\Container\ContainerInterface;
use Psr\Http\Message\ResponseInterface;

class Render implements RenderInterface
{
    /**
     * @var ContainerInterface
     */
    protected $container;

    /**
     * @var string
     */
    protected $engine;

    /**
     * @var string
     */
    protected $mode;

    /**
     * @var array
     */
    protected $config;

    public function __construct(ContainerInterface $container, ConfigInterface $config)
    {
        $engine = $config->get('view.engine', NoneEngine::class);
        if (! $container->has($engine)) {
            throw new EngineNotFindException("{$engine} engine is not found.");
        }

        $this->engine = $engine;
        $this->mode = $config->get('view.mode', Mode::TASK);
        $this->config = $config->get('view.config', []);
        $this->container = $container;
    }

    public function render(string $template, array $data = [], int $code = 200): ResponseInterface
    {
        $content = $this->getContents($template, $data);
        return $this->response()
            ->withStatus($code)
            ->withAddedHeader('content-type', $this->getContentType())
            ->withBody(new SwooleStream($content));
    }

    public function renderR(string $result, int $code = 200): ResponseInterface
    {
        return $this->response()
            ->withStatus($code)
            ->withAddedHeader('content-type', $this->getContentType())
            ->withBody(new SwooleStream($result));
    }

    public function getContents(string $template, array $data = []): string
    {
        try {
            switch ($this->mode) {
                case Mode::SYNC:
                    /** @var EngineInterface $engine */
                    $engine = $this->container->get($this->engine);
                    $result = $engine->render($template, $data, $this->config);
                    break;
                case Mode::TASK:
                default:
                    $executor = $this->container->get(TaskExecutor::class);
                    $result = $executor->execute(new Task([$this->engine, 'render'], [$template, $data, $this->config]));
                    break;
            }

            return $result;
        } catch (\Throwable $throwable) {
            throw new RenderException($throwable->getMessage(), $throwable->getCode(), $throwable);
        }
    }

    public function getContentType(): string
    {
        $charset = ! empty($this->config['charset']) ? '; charset=' . $this->config['charset'] : '';

        return 'text/html' . $charset;
    }

    protected function response(): ResponseInterface
    {
        return Context::get(ResponseInterface::class);
    }
}
