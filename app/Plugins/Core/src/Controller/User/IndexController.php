<?php


namespace App\Plugins\Core\src\Controller\User;


use App\Plugins\Core\src\Request\User\UpdateMydataRequest;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\HttpServer\Annotation\RequestMapping;

#[Controller]
#[Middleware(LoginMiddleware::class)]
class IndexController
{
    #[GetMapping(path: "/user/setting")]
    public function user_setting(): \Psr\Http\Message\ResponseInterface
    {
        return view("plugins.Core.user.setting");
    }

    #[PostMapping(path: "/user/data")]
    public function my_data()
    {
        return auth()->data();
    }

    // 更新个人信息
    #[RequestMapping(method: "POST,HEAD",path: "/user/myUpdate")]
    public function myUpdate(UpdateMydataRequest $request): array|\Hyperf\HttpMessage\Upload\UploadedFile|null
    {
        $file = $request->file('avatar');
        $file->getPath();
        return $file;
    }

    #[GetMapping(path: "/test")]
    public function test()
    {
        return view("index");
    }
}