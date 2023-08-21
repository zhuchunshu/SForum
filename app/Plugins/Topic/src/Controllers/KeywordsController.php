<?php

declare (strict_types=1);
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
    #[GetMapping('/keywords')]
    public function index()
    {
        $color = ['#35bdc7', '#fca61e', '#e65a4f', '#f29c9f', '#76cba2', '#8f82bc'];
        $page = TopicKeyword::query()->with('kw')->paginate(90);
        return view('Topic::KeyWords.index', ['page' => $page, 'color' => $color]);
    }
    #[GetMapping('/keywords/{name}.html')]
    public function data($name)
    {
        $name = urldecode($name);
        if (!TopicKeyword::query()->where('name', $name)->exists()) {
            return admin_abort('板块:' . $name . '不存在', 404);
        }
        $data = TopicKeyword::query()->where('name', $name)->first();
        $topic_menu = [['name' => '最新发布', 'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-news" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M16 6h3a1 1 0 0 1 1 1v11a2 2 0 0 1 -4 0v-13a1 1 0 0 0 -1 -1h-10a1 1 0 0 0 -1 1v12a3 3 0 0 0 3 3h11"></path>
   <line x1="8" y1="8" x2="12" y2="8"></line>
   <line x1="8" y1="12" x2="12" y2="12"></line>
   <line x1="8" y1="16" x2="12" y2="16"></line>
</svg>', 'url' => '/keywords/' . $data->name . '.html?' . core_http_build_query(['query' => 'publish'], ['page' => request()->input('page', 1)]), 'parameter' => 'query=publish'], ['name' => __('app.essence'), 'icon' => '<svg width="24" height="24" viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg" class="icon w-3 h-3 me-1 d-none d-md-block"><g stroke-width="3" fill-rule="evenodd"><path fill="#fff" fill-opacity=".01" d="M0 0h48v48H0z"/><g stroke="currentColor" fill="none"><path d="M10.636 5h26.728L45 18.3 24 43 3 18.3z"/><path d="M10.636 5L24 43 37.364 5M3 18.3h42"/><path d="M15.41 18.3L24 5l8.59 13.3"/></g></g></svg>', 'url' => '/keywords/' . $data->name . '.html?' . core_http_build_query(['query' => 'essence'], ['page' => request()->input('page', 1)]), 'parameter' => 'query=essence'], ['name' => __('app.hot'), 'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon me-1 d-none d-md-block" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path d="M0 0h24v24H0z" stroke="none"/><path d="M12 12c2-2.96 0-7-1-8 0 3.038-1.773 4.741-3 6-1.226 1.26-2 3.24-2 5a6 6 0 1 0 12 0c0-1.532-1.056-3.94-2-5-1.786 3-2.791 3-4 2z"/></svg>', 'url' => '/keywords/' . $data->name . '.html?' . core_http_build_query(['query' => 'hot'], ['page' => request()->input('page', 1)]), 'parameter' => 'query=hot']];
        $page = TopicKeywordsWith::query()->with('topic')->where('with_id', $data->id)->paginate(15);
        return view('Topic::KeyWords.data', ['data' => $data, 'page' => $page, 'topic_menu' => $topic_menu]);
    }
}