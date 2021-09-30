<?php

declare(strict_types=1);
/**
 * This file is part of Hyperf.
 *
 * @link     https://www.hyperf.io
 * @document https://hyperf.wiki
 * @contact  group@hyperf.io
 * @license  https://github.com/hyperf/hyperf/blob/master/LICENSE
 */
namespace App\CodeFec\View;

use App\Model\AdminPlugin;
use Hyperf\Utils\ApplicationContext;
use Hyperf\View\Engine\EngineInterface;
use Hyperf\ViewEngine\Contract\FactoryInterface;

class HyperfViewEngine implements EngineInterface
{
    public function render($template, $data, $config): string
    {
        /** @var FactoryInterface $factory */
        $factory = ApplicationContext::getContainer()->get(FactoryInterface::class);
        $array = AdminPlugin::query()->where("status",1)->get();
        $result = [];
        foreach ($array as $value) {
            $result[]=$value->name;
        }
        foreach ($result as $value) {
            $factory->addNamespace($value,plugin_path($value."/resources/views"));
        }
        return $factory->make($template, $data)->render();
    }


}
