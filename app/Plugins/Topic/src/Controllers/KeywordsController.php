<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Controllers;

use App\Plugins\Topic\src\Models\TopicKeyword;
use App\Plugins\Topic\src\Models\TopicKeywordsWith;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

#[Controller]
class KeywordsController
{
//    #[GetMapping(path: '/keywords')]
    public function index()
    {
        $color = [
            '#35bdc7',
            '#fca61e',
            '#e65a4f',
            '#f29c9f',
            '#76cba2',
            '#8f82bc',
        ];
        $page = TopicKeyword::query()->with('kw')->paginate(90);
        return view('Topic::KeyWords.index', ['page' => $page, 'color' => $color]);
    }

//    #[GetMapping(path: '/keywords/{name}.html')]
    public function data($name)
    {
        $name = urldecode($name);
        if (! TopicKeyword::query()->where('name', $name)->exists()) {
            return admin_abort('标签:' . $name . '不存在', 404);
        }
        $data = TopicKeyword::query()->where('name', $name)->first();
        $page = TopicKeywordsWith::query()->with('topic')->where('with_id', $data->id)->paginate(15);
        return view('Topic::KeyWords.data', ['data' => $data, 'page' => $page]);
    }
}
