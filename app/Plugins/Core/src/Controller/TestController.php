<?php


namespace App\Plugins\Core\src\Controller;

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
        return view("Core::test");
    }

    public function build(string $url){
        $data = explode("=",parse_url($url)['query']);
        return [$data[0] => $data[1]];
    }
}