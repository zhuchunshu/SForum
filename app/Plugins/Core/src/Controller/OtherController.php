<?php

namespace App\Plugins\Core\src\Controller;

use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use function Hyperf\ViewEngine\view;
/**
 * Class OtherController
 * @package App\Plugins\Core\src\Controller
 */
#[Controller]
class OtherController
{
    #[GetMapping('/help/core/viewRender/{view}.html')]
    public function renderView($view)
    {
        $view = "App::" . $view;
        if (!view()->exists($view)) {
            return admin_abort(['msg' => '视图不存在'], 404);
        }
        return view($view);
    }
}