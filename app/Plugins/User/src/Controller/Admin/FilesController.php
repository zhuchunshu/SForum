<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Controller\Admin;

use App\Middleware\AdminMiddleware;
use App\Plugins\Core\src\Handler\FileUpload;
use App\Plugins\Core\src\Handler\Store\Handler\MasterHandler;
use App\Plugins\Core\src\Service\FileStoreService;
use App\Plugins\User\src\Models\UserUpload;
use App\Plugins\User\src\Request\UploadFile;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Paginator\LengthAwarePaginator;
use Hyperf\Utils\Collection;
use Symfony\Component\Finder\Finder;

#[Controller(prefix: '/admin/files')]
#[Middleware(AdminMiddleware::class)]
class FilesController
{

    #[GetMapping(path: '')]
    public function index()
    {
        return view("User::Admin.Files.index");
    }

    #[PostMapping(path: '')]
    public function submit()
    {
        $request = request()->all();
        $handler = function ($request) {
            return redirect()->back()->with('success', '更新成功!')->go();
        };

        // 通过中间件
        $run = $this->throughMiddleware($handler, $this->middlewares());
        return $run($request);
    }

    /**
     * 通过中间件 through the middleware.
     * @param $handler
     * @param $stack
     * @return \Closure|mixed
     */
    protected function throughMiddleware($handler, $stack): mixed
    {
        // 闭包实现中间件功能 closures implement middleware functions
        foreach ($stack as $middleware) {
            $handler = function ($request) use ($handler, $middleware) {
                if ($middleware instanceof \Closure) {
                    return call_user_func($middleware, $request, $handler);
                }

                return call_user_func([new $middleware(), 'handler'], $request, $handler);
            };
        }
        return $handler;
    }

    private function middlewares(): array
    {
        $_[] = MasterHandler::class;
        $middlewares = array_merge($_, (new FileStoreService())->get_handlers());
        return array_reverse($middlewares);
    }

    #[GetMapping(path: 'user')]
    public function user_file()
    {
        $page = UserUpload::query()->orderBy('id', 'desc')->with('user')->paginate(30);

        return view('User::Admin.Files.user', ['page' => $page]);
    }

    #[GetMapping(path: 'upload')]
    public function upload()
    {
        return view('User::Admin.Files.upload');
    }

    #[PostMapping(path: 'upload')]
    public function upload_submit(UploadFile $request, FileUpload $uploader)
    {
        $file = $request->file('file');
        $data = $uploader->save($file, 'admin', admin_auth()->id());
        if ($data['success'] === true) {
            return redirect()->url('/admin/files/upload?url=' . $data['path'])->with('success', '上传成功!')->go();
        }

        return redirect()->url('/admin/files/upload')->with('danger', '上传失败!')->go();
    }

}
