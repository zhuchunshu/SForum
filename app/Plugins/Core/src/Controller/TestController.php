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
        $id =1;
        go(static function() use($id){
            for ($i=0; $i < 1000000000; $i++) {
                $id++;
            }
        });
        return $id;
    }
}