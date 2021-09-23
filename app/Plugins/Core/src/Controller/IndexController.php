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
        if(request()->input("query")==="hot"){
            $page = Topic::query()
                ->with("tag","user")
                ->orderBy("view","desc")
                ->paginate(get_options("topic_home_num",15));
        }
        if(request()->input("query")==="likes"){
            $page = Topic::query()
                ->with("tag","user")
                ->orderBy("like","desc")
                ->paginate(get_options("topic_home_num",15));
        }
        if(request()->input("query")==="updated_at"){
            $page = Topic::query()
                ->with("tag","user")
                ->orderBy("updated_at","desc")
                ->paginate(get_options("topic_home_num",15));
        }
        $topic_menu = [
            [
                "name" => "热度最高",
                "url"=> "/?".core_http_build_query(['query'=>'hot'],['page' => request()->input('page' , 1)]),
                "parameter" => "query=hot"
            ],
            [
                "name" => "最多点赞",
                "url"=> "/?".core_http_build_query(['query'=>'likes'],['page' => request()->input('page' , 1)]),
                "parameter" => "query=likes"
            ],
            [
                "name" => "最后更新",
                "url"=> "/?".core_http_build_query(['query'=>'updated_at'],['page' => request()->input('page' , 1)]),
                "parameter" => "query=updated_at"
            ],
        ];
        return view("plugins.Core.index",["page" => $page,"topic_menu"=>$topic_menu]);
    }

    #[GetMapping(path:"/{id}.html")]
    public function show($id){
        if(!Topic::query()->where('id',$id)->exists()) {
            return admin_abort("页面不存在",404);
        }
        return (new ShowTopic())->handle($id);
    }

}