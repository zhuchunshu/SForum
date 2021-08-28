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
    /**
     * @GetMapping(path="/")
     */
    public function index(){
        return view("plugins.Core.test");
    }

    #[GetMapping(path: "/test")]
    public function test()
    {
        $content = '$[Inkedus] $[Inkedus]';
        preg_match_all("/(?<=\\$\\[)[^]]+/u", $content, $arrMatches);
        return $arrMatches[0];
    }
}