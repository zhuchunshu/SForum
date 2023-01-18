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
use App\Plugins\Core\src\Handler\UploadHandler;
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
    #[GetMapping(path: 'user')]
    public function user_file()
    {
        $page = UserUpload::query()->orderBy('id', 'desc')->with('user')->paginate(30);

        return view('User::Admin.Files.user', ['page' => $page]);
    }

    #[GetMapping(path: '')]
    public function index()
    {

        return view('User::Admin.Files.index', ['page' => $this->page()]);
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
            return redirect()->url('/admin/files/upload?url=' . $data['path'])->with('success', '上传成功!')->go();
        }

        return redirect()->url('/admin/files/upload')->with('danger', '上传失败!')->go();
    }

    private function get_all_files()
    {
        $path = public_path('/upload');
        $files = [];
        $result = Finder::create()->in($path)->files();
        $id=1;
        foreach ($result as $item) {
            $files[] = [
                'id' => $id++,
                'extension' => $item->getExtension(),
                'size' => $item->getSize(),
                'date' => date('Y-m-d H:i:s', $item->getCTime()),
                'filename' => $item->getFilename(),
                'url' => url('/upload/' . $item->getRelativePath() . '/' . $item->getFilename()),
                'path' => $item->getRealPath(),
            ];
        }
        return $files;
    }

    private function page()
    {
        $currentPage = (int) request()->input('page', 1);
        $perPage = (int) request()->input('per_page', 15);

        // 这里根据 $currentPage 和 $perPage 进行数据查询，以下使用 Collection 代替
        $collection = new Collection($this->get_all_files());

        $data = array_values($collection->forPage($currentPage, $perPage)->toArray());
        return new LengthAwarePaginator($data, count($this->get_all_files()), $perPage, $currentPage);
    }
}
