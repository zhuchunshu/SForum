<?php


namespace App\Plugins\Topic\src\Controllers;

use App\Plugins\Topic\src\Models\TopicTag;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

#[Controller]
class TagsController
{
    #[GetMapping(path: "/tags")]
    public function index(): \Psr\Http\Message\ResponseInterface
    {
        $page = TopicTag::query()->paginate(15,['*'],"TagPage");
        return view("plugins.Topic.Tags.index",['page' => $page]);
    }
}