<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Search\src\Controller;

use App\Plugins\Core\src\Models\Post;
use App\Plugins\Topic\src\Models\Topic;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\Paginator\LengthAwarePaginator;
use Hyperf\Utils\Collection;
#[Controller]
class SearchController
{
    #[GetMapping('/search')]
    public function index()
    {
        $q = request()->input('q');
        if (!$q) {
            return redirect()->back()->with('danger', '搜索内容不能为空')->go();
        }
        // 中文转义搜索内容
        $q = urldecode($q);
        $topics = [];
        $topic = Topic::where('title', 'like', '%' . $q . '%')->with('user', 'post', 'tag')->get(['title', 'user_id', 'post_id', 'id', 'created_at', 'tag_id']);
        foreach ($topic as $item) {
            $topics[] = ['user' => ['name' => $item->user->username, 'url' => '/users/' . $item->user->id . '.html'], 'tag' => ['name' => $item->tag->name, 'url' => '/tags/' . $item->tag->id . '.html'], 'created_at' => $item->created_at, 'title' => $item->title, 'content' => str_limit(remove_bbCode(strip_tags($item->post->content) ?: '暂无内容') ?: '暂无内容', 100), 'url' => '/' . $item->id . '.html'];
        }
        $topics2 = [];
        $_topic = Post::where('topic_id', '!=', 'null')->where('topic_id', '!=', '0')->where('content', 'like', '%' . $q . '%')->with('user', 'topic', 'topic.tag')->get(['user_id', 'topic_id', 'content', 'created_at', 'id']);
        foreach ($_topic as $topic) {
            if (in_array($item, $topics2) || !Topic::where('id', $topic->topic_id)->exists()) {
                continue;
            }
            $item = ['user' => ['name' => $topic->user->username, 'url' => '/users/' . $topic->user->id . '.html'], 'tag' => ['name' => $topic->topic->tag->name, 'url' => '/tags/' . $topic->topic->tag->id . '.html'], 'created_at' => $topic->created_at, 'title' => @$topic->topic->title ?: '暂无标题', 'content' => @str_limit(remove_bbCode(strip_tags($topic->content) ?: '暂无内容') ?: '暂无内容', 100), 'url' => '/' . $topic->topic_id . '.html'];
            $topics2[] = $item;
        }
        $data = array_merge($topics, $topics2);
        $data = $this->unique_key_array($data, 'content');
        // 倒序
        array_multisort(array_column($data, 'created_at'), SORT_DESC, $data);
        $page = $this->page($data);
        return view('Search::data', ['page' => $page, 'q' => $q]);
    }
    private function unique_key_array($inputArray, $key) : array
    {
        $uniqueKeys = [];
        $outputArray = [];
        foreach ($inputArray as $item) {
            if (!in_array($item[$key], $uniqueKeys)) {
                $uniqueKeys[] = $item[$key];
                $outputArray[] = $item;
            }
        }
        return $outputArray;
    }
    private function page($result)
    {
        $currentPage = (int) request()->input('page', 1);
        $perPage = (int) request()->input('per_page', 15);
        // 这里根据 $currentPage 和 $perPage 进行数据查询，以下使用 Collection 代替
        $collection = new Collection($result);
        $data = array_values($collection->forPage($currentPage, $perPage)->toArray());
        return new LengthAwarePaginator($data, count($result), $perPage, $currentPage);
    }
}