<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Controllers\Admin;

use App\Middleware\AdminMiddleware;
use App\Plugins\Topic\src\Models\TopicKeyword;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\DeleteMapping;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;

#[Controller(prefix: '/admin/topic/keywords')]
#[Middleware(AdminMiddleware::class)]
class KeywordController
{
    #[GetMapping(path: '')]
    public function index()
    {
        $q = request()->input('q', '');
        $page = TopicKeyword::with('user')->where('name', 'like', '%' . $q . '%')->paginate(20);
        return view('Topic::KeyWords.admin', ['page' => $page]);
    }

    #[DeleteMapping(path: '{id}')]
    public function remove($id)
    {
        if (empty($id) || ! (int) $id) {
            return json_api(400, false, 'Bad request');
        }
        if (TopicKeyword::where('id', $id)->exists()) {
            TopicKeyword::where('id', $id)->delete();
        }
        return json_api(200, false, ['msg' => '删除成功']);
    }
}
