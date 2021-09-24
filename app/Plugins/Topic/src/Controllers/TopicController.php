<?php


namespace App\Plugins\Topic\src\Controllers;

use App\Plugins\Topic\src\Handler\Topic\CreateTopic;
use App\Plugins\Topic\src\Handler\Topic\CreateTopicView;
use App\Plugins\Topic\src\Handler\Topic\EditTopic;
use App\Plugins\Topic\src\Handler\Topic\EditTopicView;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\Topic\src\Requests\Topic\CreateTopicRequest;
use App\Plugins\Topic\src\Requests\Topic\UpdateTopicRequest;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Utils\Str;

#[Controller(prefix:"/topic")]
#[Middleware(\App\Plugins\User\src\Middleware\AuthMiddleware::class)]
#[Middleware(LoginMiddleware::class)]
class TopicController
{
    #[GetMapping(path: "create")]
    #[Middleware(LoginMiddleware::class)]
    public function create(){
        if(!Authority()->check("topic_create")){
            return admin_abort("无发帖权限");
        }
        return (new CreateTopicView())->handler();
    }

    #[PostMapping(path: "create")]
    #[Middleware(LoginMiddleware::class)]
    public function create_post(CreateTopicRequest $request){
        if(!Authority()->check("topic_create")){
            return Json_Api(401,false,['无发帖权限']);
        }
        return (new CreateTopic())->handler($request);
    }

    #[GetMapping(path:"/topic/{topic_id}/edit")]
    public function edit($topic_id){
        if(!Topic::query()->where("id",$topic_id)->exists()) {
            return admin_abort("帖子不存在",404);
        }
        $data = Topic::query()->where("id",$topic_id)->with("user")->first();
        $quanxian = false;
        if(Authority()->check("admin_topic_edit") && curd()->GetUserClass(auth()->data()->class_id)['permission-value']>curd()->GetUserClass($data->user->class_id)['permission-value']){
            $quanxian =true;
        }else if(Authority()->check("topic_edit") && auth()->id() === $data->user->id){
            $quanxian =true;
        }
        if($quanxian===true){
            return (new EditTopicView())->handler($data);
        }
        return admin_abort("无权限",401);
    }

    #[PostMapping(path:"/topic/edit")]
    public function edit_post(UpdateTopicRequest $request){
        $quanxian = false;
        if(Authority()->check("admin_topic_edit") && curd()->GetUserClass(auth()->data()->class_id)['permission-value']>curd()->GetUserClass($data->user->class_id)['permission-value']){
            $quanxian =true;
        }else if(Authority()->check("topic_edit") && auth()->id() === $data->user->id){
            $quanxian =true;
        }
        if($quanxian===true){
            return (new EditTopic())->handler($request);
        }
        return Json_Api(401,false,['无权限']);
    }
}