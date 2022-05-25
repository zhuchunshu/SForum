<?php

namespace App\Plugins\Search\src\Controller;

use App\Plugins\Core\src\Models\Post;
use App\Plugins\Topic\src\Models\Topic;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

#[Controller]
class SearchController
{
    #[GetMapping(path:"/search")]
    public function index(){
        $q = request()->input('q');
        if(!$q){
            return admin_abort("搜索内容不能为空",403);
        }
        $page = Post::where('content','like','%'.$q.'%')
            ->paginate(15);
        return view("Search::data",['page' => $page,'q' => $q]);
    }
}