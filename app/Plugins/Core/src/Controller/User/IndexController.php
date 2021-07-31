<?php


namespace App\Plugins\Core\src\Controller\User;


use App\Plugins\Core\src\Request\User\Mydata\JibenRequest;
use App\Plugins\User\src\Mail\RePwd;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UserRepwd;
use Exception;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\HttpServer\Annotation\RequestMapping;
use HyperfExt\Hashing\Hash;
use HyperfExt\Mail\Mail;
use Illuminate\Support\Str;
use Psr\Http\Message\ResponseInterface;

#[Controller]
#[Middleware(LoginMiddleware::class)]
class IndexController
{
    /**
     * @throws Exception
     */
    #[GetMapping(path: "/test")]
    public function test()
    {
        return request()->fullUrl();
    }
}