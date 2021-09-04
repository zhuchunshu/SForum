<?php


namespace App\Plugins\Core\src\Controller;

use App\Plugins\Core\src\Lib\ShortCode\Test;
use App\Plugins\Topic\src\Models\Topic;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

/**
 * @Controller
 * @package App\Plugins\Core\src\Controller
 */
class TestController
{

    #[GetMapping(path: "/test")]
    public function test()
    {
        //ShortCodeR()->add("a","App\Plugins\Core\src\Lib\ShortCode\Defaults@a");
        $content = "[reply]看你怎么隐藏[/reply] 2222 [reply]看你怎么隐藏[/reply]";
        return response()->raw(remove_bbCode(($content)));
    }
}