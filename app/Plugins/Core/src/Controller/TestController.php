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
        return core_http_build_query(core_get_page('http://127.0.0.1:9501/?page=2'),request()->all());
    }

    public function build(string $url){
        $data = explode("=",parse_url($url)['query']);
        return [$data[0] => $data[1]];
    }
}