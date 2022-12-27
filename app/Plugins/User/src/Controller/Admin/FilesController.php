<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Controller\Admin;

use App\Middleware\AdminMiddleware;
use App\Plugins\Core\src\Handler\UploadHandler;
use App\Plugins\User\src\Models\UserUpload;
use App\Plugins\User\src\Request\UploadFile;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;

#[Controller(prefix: '/admin/users/files')]
#[Middleware(AdminMiddleware::class)]
class FilesController
{
    #[GetMapping(path: '')]
    public function index()
    {
        $page = UserUpload::query()->orderBy('id', 'desc')->with('user')->paginate(30);
        return view('User::Admin.Files.index', ['page' => $page]);
    }

    #[GetMapping(path: 'upload')]
    public function upload()
    {
        return view('User::Admin.Files.upload');
    }

    #[PostMapping(path: 'upload')]
    public function upload_submit(UploadFile $request, UploadHandler $uploader)
    {
        $file = $request->file('file');
        $data = $uploader->save($file, 'admin', admin_auth()->id());
        if ($data['success'] === true) {
            return redirect()->url('/admin/users/files/upload?url=' . $data['path'])->with('success', '上传成功!')->go();
        }

        return redirect()->url('/admin/users/files/upload')->with('danger', '上传失败!')->go();
    }
}
