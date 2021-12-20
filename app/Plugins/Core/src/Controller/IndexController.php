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
        $title = null;
        $page = Topic::query(true)
            ->where("status",'publish')
            ->with("tag","user")
            ->orderBy("topping","desc")
            ->orderBy("id","desc")
            ->paginate(get_options("topic_home_num",15));
        if(request()->input("query")==="hot"){
            $page = Topic::query()
                ->where("status",'publish')
                ->with("tag","user")
                ->orderBy("view","desc")
                ->orderBy("id","desc")
                ->paginate(get_options("topic_home_num",15));
            $title = "热度最高的帖子";
        }
        if(request()->input("query")==="likes"){
            $page = Topic::query()
                ->where("status",'publish')
                ->with("tag","user")
                ->orderBy("like","desc")
                ->orderBy("id","desc")
                ->paginate(get_options("topic_home_num",15));
            $title = "点赞最多的帖子";
        }
        if(request()->input("query")==="updated_at"){
            $page = Topic::query()
                ->where("status",'publish')
                ->with("tag","user")
                ->orderBy("updated_at","desc")
                ->paginate(get_options("topic_home_num",15));
            $title = "最后更新";
        }
        if(request()->input("query")==="essence"){
            $page = Topic::query()
                ->where([["essence",">",0],["status",'publish']])
                ->with("tag","user")
                ->orderBy("updated_at","desc")
                ->paginate(get_options("topic_home_num",15));
            $title = "最后更新";
        }
        if(request()->input("query")==="topping"){
            $page = Topic::query()
                ->where([["topping",">",0],["status",'publish']])
                ->with("tag","user")
                ->orderBy("updated_at","desc")
                ->paginate(get_options("topic_home_num",15));
            $title = "最后更新";
        }
        $topic_menu = [
            [
                "name" => "置顶帖子",
                "url"=> "/?".core_http_build_query(['query'=>'topping'],['page' => request()->input('page' , 1)]),
                "parameter" => "query=topping"
            ],
            [
                "name" => "精华帖子",
                "url"=> "/?".core_http_build_query(['query'=>'essence'],['page' => request()->input('page' , 1)]),
                "parameter" => "query=essence"
            ],
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
        return view("Core::index",["page" => $page,"topic_menu"=>$topic_menu,'title'=>$title]);
    }

    #[GetMapping(path:"/{id}.html[/{comment}]")]
    public function show($id,$comment=null){
        if(!Topic::query()->where([['id',$id],['status','publish']])->exists()) {
            return admin_abort("页面不存在",404);
        }
        return (new ShowTopic())->handle($id,$comment);
    }

    #[GetMapping(path:"/{id}.md")]
    public function show_md($id){
        if(!Topic::query()->where([['id',$id],['status','publish']])->exists()) {
            return admin_abort("页面不存在",404);
        }
        $data = Topic::query()
            ->where([['id', $id],['status','publish']])
            ->select("markdown")
            ->first();
        return response()->raw($data->markdown);
    }

}