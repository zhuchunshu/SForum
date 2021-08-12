<?php

declare(strict_types=1);

namespace App\Controller;

use App\CodeFec\Admin\Ui\Card;
use App\CodeFec\Admin\Ui\Row;
use App\CodeFec\Admin\Ui;
use Hyperf\HttpServer\Annotation\AutoController;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Psr\Http\Message\ResponseInterface;

/**
 * Class PluginsController
 * @AutoController(prefix="/admin/plugins")
 * @Middleware(\App\Middleware\AdminMiddleware::class)
 * @package App\Controller
 */
class PluginsController
{
    /**
     * @GetMapping(path="/")
     * @param Ui $ui
     * @param Row $row
     * @param Card $card
     * @return ResponseInterface
     */
    public function index(Ui $ui,Row $row,Card $card): ResponseInterface
    {
        return $ui
            ->title("插件管理")
            ->body($row->row("col-md-12")
                ->content($card
                    ->title("插件列表")
                    ->titleType(1)
                    ->id("vue-plugin-table")
                    ->content(view("admin.plugins"))
                    ->render()
                )
                ->render())
            ->render();
    }
}
