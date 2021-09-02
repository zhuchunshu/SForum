<?php


namespace App\Plugins\Core\src\Controller;

use App\Plugins\Core\src\Lib\ShortCode\Test;
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
        ShortCode()->add("a","App\Plugins\Core\src\Lib\ShortCode\Defaults@a");
        $content = 'a @Inkedus @Inkedus @Nb a';
        return replace_all_at($content);
    }
}