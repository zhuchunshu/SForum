<?php
namespace App\Plugins\User\src\Controller;

use App\Plugins\User\src\Request\Create;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;

/**
 * Class UserClass
 * @Controller
 * @Middleware(\App\Middleware\AdminMiddleware::class)
 * @package App\Plugins\User\src\Controller
 */
class UserClass
{
    /**
     * @GetMapping(path="/admin/userClass")
     */
    public function index(){
        $page = \App\Plugins\User\src\Models\UserClass::query()->paginate(15);
        return view("plugins.User.Class.index",['page' => $page]);
    }

    /**
     * @GetMapping(path="/admin/userClass/create")
     */
    public function create(){
        return view("plugins.User.Class.create");
    }

    /**
     * @PostMapping(path="/admin/userClass/create")
     */
    public function create_post(Create $request): array
    {
        \App\Plugins\User\src\Models\UserClass::query()->create($request->validated());
        return Json_Api(200,true,['msg' => '用户组创建成功!']);
    }
}