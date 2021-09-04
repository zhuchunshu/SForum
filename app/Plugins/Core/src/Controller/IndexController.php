<?php

namespace App\Plugins\Core\src\Controller;

use App\Plugins\Topic\src\Handler\Topic\ShowTopic;
use App\Plugins\Topic\src\Models\Topic;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\PostMapping;

#[Controller]
class IndexController
{
    #[GetMapping(path:"/")]
    public function index(): \Psr\Http\Message\ResponseInterface
    {
        $page = Topic::query()
            ->with("tag","user")
            ->orderBy("id","desc")
            ->paginate(get_options("topic_home_num",15));
        return view("plugins.Core.index",["page" => $page]);
    }

    #[GetMapping(path:"/{id}.html")]
    public function show($id){
        if(!Topic::query()->where('id',$id)->exists()) {
            return admin_abort("页面不存在",404);
        }
        return (new ShowTopic())->handle($id);
    }

}